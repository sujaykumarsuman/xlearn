// Package admin is the owner's `identity admin …` CLI (sprint m1-04; ADR-0033 §8). It
// runs inside the identity image — distroless, so no shell: the owner calls it with
//
//	ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin <verb> …'
//
// (docs/runbooks/identity-admin.md). It uses the pod's own database credentials
// (identity.LoadConfig().DB) over a 1-connection pool, never runs migrations and adds no
// web surface and no secret. Every verb, reads included, writes one identity.admin_audit
// row in the same transaction as its change (store/admin.go) plus one log line on stderr.
// No secret reaches either: the one-time tester password is printed once, on stdout.
// An exec'd process's output reaches the owner's terminal, not `kubectl logs`, so
// admin_audit is the trail.
package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/identity"
	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
)

// Exit codes (the runbook lists them).
const (
	ExitOK       = 0
	ExitError    = 1 // database or unexpected error
	ExitUsage    = 2 // bad arguments
	ExitNotFound = 3 // <user> matches no account
	ExitRefused  = 4 // a guard refused: last owner, seat cap, email taken
)

// verbTimeout bounds one CLI run (a lock wait included).
const verbTimeout = 60 * time.Second

// Store is the admin persistence the CLI drives (*store.Admin).
type Store interface {
	ListAccounts(ctx context.Context, f store.ListFilter) ([]store.AdminAccount, error)
	Suspend(ctx context.Context, user string) (store.AdminResult, error)
	Reactivate(ctx context.Context, user string, seatCap int) (store.AdminResult, error)
	RevokeSessions(ctx context.Context, user string) (store.AdminResult, error)
	SetRole(ctx context.Context, user, role string, seatCap int) (store.AdminResult, error)
	CreateAccount(ctx context.Context, email, passwordHash, displayName, role string) (store.AdminResult, error)
	Seats(ctx context.Context, seatCap int) (store.SeatReport, error)
}

// Deps are the CLI's seams. Open is called only after the arguments parse.
type Deps struct {
	Open         func(ctx context.Context) (Store, func(), error)
	SeatCap      int
	Now          func() time.Time
	HashPassword func(pw string) (string, error)
	NewPassword  func() (string, error)
}

// Main is `identity admin …` for cmd/identity: the pod's config and database.
func Main(args []string, stdout, stderr io.Writer) int {
	cfg := identity.LoadConfig()
	return Run(context.Background(), Deps{
		Open: func(ctx context.Context) (Store, func(), error) {
			pool, err := openPool(ctx, cfg.DB)
			if err != nil {
				return nil, nil, err
			}
			return store.NewAdmin(pool), pool.Close, nil
		},
		SeatCap:      cfg.Auth.SeatCap,
		Now:          time.Now,
		HashPassword: identity.HashPassword,
		NewPassword:  NewPassword,
	}, args, stdout, stderr)
}

// openPool opens a 1-connection pool with the pod's credentials and search_path.
func openPool(ctx context.Context, db identity.DBConfig) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(db.DSN())
	if err != nil {
		return nil, err
	}
	if db.SearchPath != "" {
		pc.ConnConfig.RuntimeParams["search_path"] = db.SearchPath
	}
	pc.MaxConns = 1
	pc.MinConns = 0
	return pgxpool.NewWithConfig(ctx, pc)
}

// NewPassword is a random 24-character one-time password: 18 bytes of crypto/rand,
// base64url (144 bits).
func NewPassword() (string, error) {
	var b [18]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

const usage = `usage: identity admin <verb> …

  account list [--dormant 30d] [--role learner|tester|owner] [--status active|suspended] [--json]
  account suspend <user>
  account reactivate <user>
  account revoke-sessions <user>
  account set-role <user> learner|tester|owner
  account create --role tester --email <email>
  seats

<user> is an email (contains @), an account id (UUID) or a username; email and username
match case-insensitively. Every verb is recorded in identity.admin_audit.
exit codes: 0 ok · 1 error · 2 usage · 3 user not found · 4 refused (last owner, seat cap, email taken)
`

// cli is one run's state.
type cli struct {
	deps   Deps
	stdout io.Writer
	stderr io.Writer
	log    *slog.Logger
}

// Run executes one `identity admin` invocation and returns its exit code.
func Run(ctx context.Context, deps Deps, args []string, stdout, stderr io.Writer) int {
	c := &cli{deps: deps, stdout: stdout, stderr: stderr,
		log: slog.New(slog.NewJSONHandler(stderr, nil)).With("component", "identity-admin")}
	if deps.Now == nil {
		c.deps.Now = time.Now
	}
	if deps.SeatCap <= 0 {
		c.deps.SeatCap = identity.DefaultSeatCap
	}
	ctx, cancel := context.WithTimeout(ctx, verbTimeout)
	defer cancel()

	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		_, _ = io.WriteString(stderr, usage)
		if len(args) == 0 {
			return ExitUsage
		}
		return ExitOK
	}
	switch args[0] {
	case "seats":
		if len(args) != 1 {
			return c.usageErr("seats takes no arguments")
		}
		return c.seats(ctx)
	case "account":
		if len(args) < 2 {
			return c.usageErr("account needs a verb")
		}
		verb, rest := args[1], args[2:]
		switch verb {
		case "list":
			return c.list(ctx, rest)
		case "suspend", "reactivate", "revoke-sessions":
			if len(rest) != 1 || strings.HasPrefix(rest[0], "-") {
				return c.usageErr("account " + verb + " takes exactly one <user>")
			}
			return c.userVerb(ctx, verb, rest[0])
		case "set-role":
			if len(rest) != 2 {
				return c.usageErr("account set-role takes <user> and a role")
			}
			if !validRole(rest[1]) {
				return c.usageErr("role must be learner, tester or owner")
			}
			return c.setRole(ctx, rest[0], rest[1])
		case "create":
			return c.create(ctx, rest)
		}
		return c.usageErr("unknown account verb " + strconv.Quote(verb))
	}
	return c.usageErr("unknown verb " + strconv.Quote(args[0]))
}

func (c *cli) usageErr(msg string) int {
	_, _ = fmt.Fprintf(c.stderr, "identity admin: %s\n\n%s", msg, usage)
	return ExitUsage
}

// open connects on demand (after the arguments parsed).
func (c *cli) open(ctx context.Context) (Store, func(), int) {
	st, closeFn, err := c.deps.Open(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(c.stderr, "identity admin: connect to postgres: %v\n", err)
		return nil, nil, ExitError
	}
	if closeFn == nil {
		closeFn = func() {}
	}
	return st, closeFn, ExitOK
}

// done logs the verb's one stderr line and maps its error to an exit code.
func (c *cli) done(verb, target string, err error, attrs ...any) int {
	if target == "" {
		target = "-"
	}
	code, outcome := exitFor(err)
	args := append([]any{"verb", verb, "target", target, "outcome", outcome}, attrs...)
	if err != nil {
		c.log.Warn("identity admin", args...)
		_, _ = fmt.Fprintf(c.stderr, "identity admin: %v\n", err)
		return code
	}
	c.log.Info("identity admin", args...)
	return code
}

func exitFor(err error) (int, string) {
	switch {
	case err == nil:
		return ExitOK, "ok"
	case errors.Is(err, store.ErrNotFound):
		return ExitNotFound, "user_not_found"
	case errors.Is(err, store.ErrLastOwner):
		return ExitRefused, "refused_last_owner"
	case errors.Is(err, store.ErrSeatCap):
		return ExitRefused, "refused_seat_cap"
	case errors.Is(err, store.ErrEmailTaken):
		return ExitRefused, "refused_email_taken"
	default:
		return ExitError, "error"
	}
}

// --- verbs ---

func (c *cli) list(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("account list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dormant := fs.String("dormant", "", "no session created within this window (e.g. 30d, 72h)")
	role := fs.String("role", "", "learner|tester|owner")
	status := fs.String("status", "", "active|suspended")
	asJSON := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return c.usageErr(err.Error())
	}
	if fs.NArg() != 0 {
		return c.usageErr("account list takes flags only")
	}
	if *role != "" && !validRole(*role) {
		return c.usageErr("--role must be learner, tester or owner")
	}
	if *status != "" && *status != store.StatusActive && *status != store.StatusSuspended {
		return c.usageErr("--status must be active or suspended")
	}
	f := store.ListFilter{Role: *role, Status: *status, Dormant: *dormant}
	if *dormant != "" {
		d, err := parseWindow(*dormant)
		if err != nil {
			return c.usageErr("--dormant: " + err.Error())
		}
		f.DormantBefore = c.deps.Now().Add(-d)
	}
	st, closeFn, code := c.open(ctx)
	if code != ExitOK {
		return code
	}
	defer closeFn()
	rows, err := st.ListAccounts(ctx, f)
	if err != nil {
		return c.done(store.VerbAccountList, "-", err)
	}
	if *asJSON {
		enc := json.NewEncoder(c.stdout)
		enc.SetIndent("", "  ")
		if rows == nil {
			rows = []store.AdminAccount{}
		}
		_ = enc.Encode(rows)
	} else {
		tw := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(tw, "ID\tEMAIL\tUSERNAME\tROLE\tSTATUS\tADMITTED\tCREATED\tLAST SESSION")
		for _, a := range rows {
			_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", a.ID, dash(a.Email), dash(a.Username), a.Role, a.Status,
				dash(a.AdmittedVia), a.CreatedAt.UTC().Format("2006-01-02"), lastSession(a.LastSessionAt))
		}
		_ = tw.Flush()
		_, _ = fmt.Fprintf(c.stdout, "%d account(s)\n", len(rows))
	}
	return c.done(store.VerbAccountList, "-", nil, "count", len(rows))
}

func (c *cli) userVerb(ctx context.Context, verb, user string) int {
	st, closeFn, code := c.open(ctx)
	if code != ExitOK {
		return code
	}
	defer closeFn()
	var (
		res   store.AdminResult
		err   error
		audit string
	)
	switch verb {
	case "suspend":
		audit = store.VerbAccountSuspend
		res, err = st.Suspend(ctx, user)
	case "reactivate":
		audit = store.VerbAccountReactivate
		res, err = st.Reactivate(ctx, user, c.deps.SeatCap)
	case "revoke-sessions":
		audit = store.VerbAccountRevokeSessions
		res, err = st.RevokeSessions(ctx, user)
	}
	if err != nil {
		return c.done(audit, res.Account.ID, err)
	}
	who := describe(res.Account)
	switch verb {
	case "suspend":
		if res.Changed {
			_, _ = fmt.Fprintf(c.stdout, "suspended %s (was %s); revoked %d session(s)\n", who, res.FromStatus, res.RevokedSessions)
		} else {
			_, _ = fmt.Fprintf(c.stdout, "%s was already suspended; revoked %d session(s)\n", who, res.RevokedSessions)
		}
	case "reactivate":
		if res.Changed {
			_, _ = fmt.Fprintf(c.stdout, "reactivated %s (role %s)\n", who, res.Account.Role)
		} else {
			_, _ = fmt.Fprintf(c.stdout, "%s was already active\n", who)
		}
	case "revoke-sessions":
		_, _ = fmt.Fprintf(c.stdout, "revoked %d session(s) of %s\n", res.RevokedSessions, who)
	}
	return c.done(audit, res.Account.ID, nil, "changed", res.Changed, "revoked_sessions", res.RevokedSessions)
}

func (c *cli) setRole(ctx context.Context, user, role string) int {
	st, closeFn, code := c.open(ctx)
	if code != ExitOK {
		return code
	}
	defer closeFn()
	res, err := st.SetRole(ctx, user, role, c.deps.SeatCap)
	if err != nil {
		return c.done(store.VerbAccountSetRole, res.Account.ID, err)
	}
	if res.Changed {
		_, _ = fmt.Fprintf(c.stdout, "role of %s: %s -> %s\n", describe(res.Account), res.FromRole, res.Account.Role)
	} else {
		_, _ = fmt.Fprintf(c.stdout, "%s is already %s\n", describe(res.Account), res.Account.Role)
	}
	return c.done(store.VerbAccountSetRole, res.Account.ID, nil, "from_role", res.FromRole, "to_role", res.Account.Role, "changed", res.Changed)
}

func (c *cli) create(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("account create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	role := fs.String("role", "", "tester (the only role CLI provisioning creates in M1b)")
	email := fs.String("email", "", "the tester's email")
	if err := fs.Parse(args); err != nil {
		return c.usageErr(err.Error())
	}
	if fs.NArg() != 0 {
		return c.usageErr("account create takes --role and --email only")
	}
	if *role != store.RoleTester {
		return c.usageErr("account create supports --role tester only (learners arrive by invite from L-A)")
	}
	addr := identity.NormalizeEmail(*email)
	if !identity.ValidEmail(addr) {
		return c.usageErr("--email must be a valid email address")
	}
	pw, err := c.deps.NewPassword()
	if err != nil {
		return c.done(store.VerbAccountCreate, "-", fmt.Errorf("generate password: %w", err))
	}
	hash, err := c.deps.HashPassword(pw)
	if err != nil {
		return c.done(store.VerbAccountCreate, "-", fmt.Errorf("hash password: %w", err))
	}
	st, closeFn, code := c.open(ctx)
	if code != ExitOK {
		return code
	}
	defer closeFn()
	res, err := st.CreateAccount(ctx, addr, hash, identity.DisplayNameFromEmail(addr), store.RoleTester)
	if err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			err = fmt.Errorf("an account with that email already exists: %w", err)
		}
		return c.done(store.VerbAccountCreate, "-", err)
	}
	// The ONLY place the password appears: the owner's terminal, once.
	_, _ = fmt.Fprintf(c.stdout, "created tester %s\none-time password (shown once; the tester changes it in Settings):\n%s\n", describe(res.Account), pw)
	return c.done(store.VerbAccountCreate, res.Account.ID, nil, "role", res.Account.Role)
}

func (c *cli) seats(ctx context.Context) int {
	st, closeFn, code := c.open(ctx)
	if code != ExitOK {
		return code
	}
	defer closeFn()
	rep, err := st.Seats(ctx, c.deps.SeatCap)
	if err != nil {
		return c.done(store.VerbSeats, "-", err)
	}
	_, _ = fmt.Fprintf(c.stdout, "seats: %d used / %d (SEAT_CAP) — active learners; the owner and testers are outside the cap\ninvites: n/a until L-A\n", rep.Used, rep.Cap)
	return c.done(store.VerbSeats, "-", nil, "used", rep.Used, "cap", rep.Cap)
}

// --- helpers ---

func validRole(r string) bool {
	return r == store.RoleLearner || r == store.RoleTester || r == store.RoleOwner
}

// parseWindow parses a --dormant window: Nd (days) or a Go duration (e.g. 72h).
func parseWindow(s string) (time.Duration, error) {
	if n, ok := strings.CutSuffix(s, "d"); ok {
		days, err := strconv.Atoi(n)
		if err != nil || days <= 0 {
			return 0, fmt.Errorf("want a positive number of days like 30d, got %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("want 30d or a duration like 72h, got %q", s)
	}
	return d, nil
}

func describe(a store.AdminAccount) string {
	switch {
	case a.Email != "":
		return a.ID + " <" + a.Email + ">"
	case a.Username != "":
		return a.ID + " (@" + a.Username + ")"
	default:
		return a.ID
	}
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func lastSession(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.UTC().Format("2006-01-02 15:04Z")
}
