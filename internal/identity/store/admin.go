package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store/gen"
)

// The owner admin CLI's persistence (sprint m1-04; ADR-0033 §7, §8, §3). The CLI
// (internal/identity/admin) parses argv and prints; every change, its guards and its
// identity.admin_audit row happen here in ONE transaction. Reads (list, seats) are
// audited too. A refused or failed verb rolls its change back and still leaves an audit
// row saying why (a separate, audit-only transaction), so the trail shows attempts.
//
// Audit rows never hold a secret (no password, no invite code) and never an email:
// target is the account id or "-" (ADR-0033 §8), and detail holds ids, roles, statuses
// and counts only. admin_audit is an honest-operator trail, not tamper-proof: anyone
// with exec in the pod is already root (accepted in ADR-0033).

// Advisory lock names (pg_advisory_xact_lock(hashtext(name))). A verb that needs both
// takes owners before seats; account rows are locked after both.
const (
	LockOwners = "identity.owners"
	LockSeats  = "identity.seats"
)

// Admin CLI refusals, mapped to the CLI's "refused" exit code.
var (
	// ErrLastOwner refuses to suspend or demote the last active owner.
	ErrLastOwner = errors.New("refused: this is the last active owner")
	// ErrSeatCap refuses to add an active learner when every seat is taken.
	ErrSeatCap = errors.New("refused: the seat cap is reached")
)

// Admin is the pgx-backed admin CLI store.
type Admin struct {
	pool *pgxpool.Pool
	q    *gen.Queries
}

// NewAdmin wraps a pool (the CLI opens a 1-connection pool with the pod's own
// credentials and never runs migrations).
func NewAdmin(pool *pgxpool.Pool) *Admin { return &Admin{pool: pool, q: gen.New(pool)} }

// AdminAccount is one account as the CLI shows it. LastSessionAt is zero when the
// account never started a session (or the row came from a write verb).
type AdminAccount struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Username      string    `json:"username"`
	Role          string    `json:"role"`
	Status        string    `json:"status"`
	AdmittedVia   string    `json:"admitted_via"`
	CreatedAt     time.Time `json:"created_at"`
	LastSessionAt time.Time `json:"last_session_at"`
}

// AdminResult is a write verb's outcome.
type AdminResult struct {
	Account AdminAccount
	// FromRole / FromStatus are the values before the verb.
	FromRole   string
	FromStatus string
	// Changed reports whether the row changed (false for an idempotent repeat).
	Changed bool
	// RevokedSessions counts sessions the verb ended.
	RevokedSessions int64
}

// ListFilter narrows `account list`. Empty strings / a zero time mean "any".
type ListFilter struct {
	Role   string
	Status string
	// DormantBefore keeps accounts with no session created at or after it.
	DormantBefore time.Time
	// Dormant is the operator's window as typed (e.g. "30d"), for the audit detail.
	Dormant string
}

// SeatReport is `seats`: active learners against SEAT_CAP.
type SeatReport struct {
	Used int64
	Cap  int
}

// AuditRow is one admin_audit row (tests and the runbook read it back).
type AuditRow struct {
	At     time.Time
	Verb   string
	Target string
	Detail map[string]any
}

// Verbs as written to admin_audit.verb.
const (
	VerbAccountList           = "account.list"
	VerbAccountSuspend        = "account.suspend"
	VerbAccountReactivate     = "account.reactivate"
	VerbAccountRevokeSessions = "account.revoke-sessions"
	VerbAccountSetRole        = "account.set-role"
	VerbAccountCreate         = "account.create"
	VerbSeats                 = "seats"
)

// ListAccounts is `account list`.
func (a *Admin) ListAccounts(ctx context.Context, f ListFilter) ([]AdminAccount, error) {
	var out []AdminAccount
	err := a.run(ctx, VerbAccountList, func(q *gen.Queries) (string, map[string]any, error) {
		rows, err := q.AdminListAccounts(ctx, gen.AdminListAccountsParams{
			Role:          textOrNull(f.Role),
			Status:        textOrNull(f.Status),
			DormantBefore: pgtype.Timestamptz{Time: f.DormantBefore, Valid: !f.DormantBefore.IsZero()},
		})
		if err != nil {
			return "-", nil, fmt.Errorf("list accounts: %w", err)
		}
		out = make([]AdminAccount, 0, len(rows))
		for _, r := range rows {
			out = append(out, AdminAccount{
				ID:            uuidString(r.ID),
				Email:         r.Email.String,
				Username:      r.Username.String,
				Role:          r.Role,
				Status:        r.Status,
				AdmittedVia:   r.AdmittedVia.String,
				CreatedAt:     r.CreatedAt.Time,
				LastSessionAt: r.LastSessionAt.Time,
			})
		}
		return "-", map[string]any{
			"role":    nullable(f.Role),
			"status":  nullable(f.Status),
			"dormant": nullable(f.Dormant),
			"count":   len(out),
		}, nil
	})
	return out, err
}

// Suspend sets status='suspended' and revokes every session, in one transaction. It
// refuses the last active owner. A repeat on a suspended account is a no-op that still
// revokes (there is nothing left to revoke) and audits.
func (a *Admin) Suspend(ctx context.Context, user string) (AdminResult, error) {
	var res AdminResult
	err := a.run(ctx, VerbAccountSuspend, func(q *gen.Queries) (string, map[string]any, error) {
		if err := q.AdminXactLock(ctx, LockOwners); err != nil {
			return "-", nil, err
		}
		acct, err := lockAccount(ctx, q, user)
		if err != nil {
			return "-", nil, err
		}
		res = AdminResult{Account: adminAccount(acct), FromRole: acct.Role, FromStatus: acct.Status}
		target := uuidString(acct.ID)
		if acct.Role == RoleOwner && acct.Status == StatusActive {
			if err := guardLastOwner(ctx, q); err != nil {
				return target, map[string]any{"role": acct.Role}, err
			}
		}
		row := acct
		if acct.Status != StatusSuspended {
			if row, err = q.AdminSetStatus(ctx, gen.AdminSetStatusParams{ID: acct.ID, Status: StatusSuspended}); err != nil {
				return target, nil, fmt.Errorf("suspend: %w", err)
			}
			res.Changed = true
		}
		if res.RevokedSessions, err = q.RevokeAllSessions(ctx, acct.ID); err != nil {
			return target, nil, fmt.Errorf("revoke sessions: %w", err)
		}
		res.Account = adminAccount(row)
		return target, map[string]any{
			"from_status":      res.FromStatus,
			"changed":          res.Changed,
			"revoked_sessions": res.RevokedSessions,
		}, nil
	})
	return res, err
}

// Reactivate sets status='active'. Reactivating a learner takes a seat, so it takes the
// seats lock and refuses when SEAT_CAP is reached (ADR-0033 §3).
func (a *Admin) Reactivate(ctx context.Context, user string, seatCap int) (AdminResult, error) {
	var res AdminResult
	err := a.run(ctx, VerbAccountReactivate, func(q *gen.Queries) (string, map[string]any, error) {
		if err := q.AdminXactLock(ctx, LockSeats); err != nil {
			return "-", nil, err
		}
		acct, err := lockAccount(ctx, q, user)
		if err != nil {
			return "-", nil, err
		}
		res = AdminResult{Account: adminAccount(acct), FromRole: acct.Role, FromStatus: acct.Status}
		target := uuidString(acct.ID)
		detail := map[string]any{"from_status": acct.Status, "role": acct.Role}
		row := acct
		if acct.Status != StatusActive {
			if acct.Role == RoleLearner {
				used, err := guardSeat(ctx, q, seatCap)
				detail["seats_used"], detail["seat_cap"] = used, seatCap
				if err != nil {
					return target, detail, err
				}
			}
			if row, err = q.AdminSetStatus(ctx, gen.AdminSetStatusParams{ID: acct.ID, Status: StatusActive}); err != nil {
				return target, detail, fmt.Errorf("reactivate: %w", err)
			}
			res.Changed = true
		}
		detail["changed"] = res.Changed
		res.Account = adminAccount(row)
		return target, detail, nil
	})
	return res, err
}

// RevokeSessions revokes every live session of the account.
func (a *Admin) RevokeSessions(ctx context.Context, user string) (AdminResult, error) {
	var res AdminResult
	err := a.run(ctx, VerbAccountRevokeSessions, func(q *gen.Queries) (string, map[string]any, error) {
		acct, err := lockAccount(ctx, q, user)
		if err != nil {
			return "-", nil, err
		}
		res = AdminResult{FromRole: acct.Role, FromStatus: acct.Status, Account: adminAccount(acct)}
		if res.RevokedSessions, err = q.RevokeAllSessions(ctx, acct.ID); err != nil {
			return uuidString(acct.ID), nil, fmt.Errorf("revoke sessions: %w", err)
		}
		return uuidString(acct.ID), map[string]any{"revoked_sessions": res.RevokedSessions}, nil
	})
	return res, err
}

// SetRole changes the account's role (never admitted_via). Demoting the last active
// owner is refused (owners lock); an active account becoming a learner takes a seat
// (seats lock, SEAT_CAP). Promoting is always allowed: the first owner is set this way.
func (a *Admin) SetRole(ctx context.Context, user, role string, seatCap int) (AdminResult, error) {
	var res AdminResult
	err := a.run(ctx, VerbAccountSetRole, func(q *gen.Queries) (string, map[string]any, error) {
		if !validRole(role) {
			return "-", map[string]any{"to_role": "invalid"}, fmt.Errorf("unknown role %q (learner|tester|owner)", role)
		}
		if err := q.AdminXactLock(ctx, LockOwners); err != nil {
			return "-", nil, err
		}
		if err := q.AdminXactLock(ctx, LockSeats); err != nil {
			return "-", nil, err
		}
		acct, err := lockAccount(ctx, q, user)
		if err != nil {
			return "-", nil, err
		}
		res = AdminResult{Account: adminAccount(acct), FromRole: acct.Role, FromStatus: acct.Status}
		target := uuidString(acct.ID)
		detail := map[string]any{"from_role": acct.Role, "to_role": role, "status": acct.Status}
		row := acct
		if acct.Role != role {
			if acct.Role == RoleOwner && acct.Status == StatusActive {
				if err := guardLastOwner(ctx, q); err != nil {
					return target, detail, err
				}
			}
			if role == RoleLearner && acct.Status == StatusActive {
				used, err := guardSeat(ctx, q, seatCap)
				detail["seats_used"], detail["seat_cap"] = used, seatCap
				if err != nil {
					return target, detail, err
				}
			}
			if row, err = q.AdminSetRole(ctx, gen.AdminSetRoleParams{ID: acct.ID, Role: role}); err != nil {
				return target, detail, fmt.Errorf("set role: %w", err)
			}
			res.Changed = true
		}
		detail["changed"] = res.Changed
		res.Account = adminAccount(row)
		return target, detail, nil
	})
	return res, err
}

// CreateAccount is `account create --role tester --email <e>`: a password account with
// admitted_via='cli', its onboarding row and the account_created outbox row, in one
// transaction with its audit row. passwordHash is the bcrypt of the one-time password
// the CLI prints; neither the password nor the email reaches the audit row.
func (a *Admin) CreateAccount(ctx context.Context, email, passwordHash, displayName, role string) (AdminResult, error) {
	var res AdminResult
	err := a.run(ctx, VerbAccountCreate, func(q *gen.Queries) (string, map[string]any, error) {
		detail := map[string]any{"role": role, "admitted_via": AdmittedViaCLI}
		if role != RoleTester {
			return "-", detail, fmt.Errorf("account create supports --role tester only (got %q)", role)
		}
		row, err := q.AdminCreateAccount(ctx, gen.AdminCreateAccountParams{
			DisplayName:  displayName,
			Email:        textOrNull(email),
			PasswordHash: textOrNull(passwordHash),
			AdmittedVia:  textOrNull(AdmittedViaCLI),
			Role:         role,
		})
		if err != nil {
			if isUniqueViolation(err) {
				detail["error"] = "email_taken"
				return "-", detail, ErrEmailTaken
			}
			return "-", detail, fmt.Errorf("create account: %w", err)
		}
		target := uuidString(row.ID)
		if _, err := q.CreateOnboarding(ctx, row.ID); err != nil {
			return target, detail, fmt.Errorf("create onboarding: %w", err)
		}
		// The event's provider is the sign-in method (a password account), as email signup's.
		eventID := newUUIDv4()
		payload, err := marshalAccountCreated(eventID, target, "email", displayName, row.CreatedAt.Time)
		if err != nil {
			return target, detail, fmt.Errorf("marshal account_created: %w", err)
		}
		if err := q.InsertOutbox(ctx, gen.InsertOutboxParams{
			EventID:     mustUUID(eventID),
			Subject:     SubjectAccountCreated,
			PayloadJson: payload,
		}); err != nil {
			return target, detail, fmt.Errorf("insert outbox: %w", err)
		}
		res = AdminResult{Account: adminAccount(row), Changed: true}
		return target, detail, nil
	})
	return res, err
}

// Seats is `seats`: active learners against SEAT_CAP (invites join the count at L-A).
func (a *Admin) Seats(ctx context.Context, seatCap int) (SeatReport, error) {
	var rep SeatReport
	err := a.run(ctx, VerbSeats, func(q *gen.Queries) (string, map[string]any, error) {
		used, err := q.AdminCountSeatsUsed(ctx)
		if err != nil {
			return "-", nil, fmt.Errorf("count seats: %w", err)
		}
		rep = SeatReport{Used: used, Cap: seatCap}
		return "-", map[string]any{"used": used, "cap": seatCap}, nil
	})
	return rep, err
}

// ListAudit returns the newest admin_audit rows (tests; a future `audit` read).
func (a *Admin) ListAudit(ctx context.Context, limit int32) ([]AuditRow, error) {
	rows, err := a.q.ListAdminAudit(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list admin_audit: %w", err)
	}
	out := make([]AuditRow, 0, len(rows))
	for _, r := range rows {
		var d map[string]any
		_ = json.Unmarshal(r.Detail, &d)
		out = append(out, AuditRow{At: r.At.Time, Verb: r.Verb, Target: r.Target, Detail: d})
	}
	return out, nil
}

// run executes one verb: fn's change and its audit row commit together. When fn fails,
// the change rolls back and an audit-only transaction records the refusal or error
// (detail.refused for a guard, detail.error otherwise; never the raw user argument).
func (a *Admin) run(ctx context.Context, verb string, fn func(q *gen.Queries) (target string, detail map[string]any, err error)) error {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := a.q.WithTx(tx)

	target, detail, ferr := fn(qtx)
	if detail == nil {
		detail = map[string]any{}
	}
	if target == "" {
		target = "-"
	}
	if ferr != nil {
		_ = tx.Rollback(ctx)
		detail["outcome"] = outcomeOf(ferr)
		if aerr := insertAudit(ctx, a.q, verb, target, detail); aerr != nil {
			return errors.Join(ferr, fmt.Errorf("audit the refusal: %w", aerr))
		}
		return ferr
	}
	detail["outcome"] = "ok"
	if err := insertAudit(ctx, qtx, verb, target, detail); err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func insertAudit(ctx context.Context, q *gen.Queries, verb, target string, detail map[string]any) error {
	raw, err := json.Marshal(detail)
	if err != nil {
		return err
	}
	_, err = q.InsertAdminAudit(ctx, gen.InsertAdminAuditParams{Verb: verb, Target: target, Detail: raw})
	return err
}

// outcomeOf classifies a failed verb for the audit row (no free text from the caller).
func outcomeOf(err error) string {
	switch {
	case errors.Is(err, ErrLastOwner):
		return "refused_last_owner"
	case errors.Is(err, ErrSeatCap):
		return "refused_seat_cap"
	case errors.Is(err, ErrNotFound):
		return "user_not_found"
	case errors.Is(err, ErrEmailTaken):
		return "refused_email_taken"
	default:
		return "error"
	}
}

// lockAccount resolves <user> — an email (contains '@', case-insensitive), an account
// id (a UUID) or a username (case-insensitive) — and locks its row.
func lockAccount(ctx context.Context, q *gen.Queries, user string) (gen.IdentityAccount, error) {
	user = strings.TrimSpace(user)
	if user == "" {
		return gen.IdentityAccount{}, ErrNotFound
	}
	var (
		row gen.IdentityAccount
		err error
	)
	switch {
	case strings.Contains(user, "@"):
		row, err = q.AdminLockAccountByEmail(ctx, user)
	default:
		if uid, perr := parseUUID(user); perr == nil {
			row, err = q.AdminLockAccountByID(ctx, uid)
		} else {
			row, err = q.AdminLockAccountByUsername(ctx, user)
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return gen.IdentityAccount{}, ErrNotFound
	}
	return row, err
}

// guardLastOwner refuses when at most one active owner remains (the caller holds the
// owners lock and the target is an active owner).
func guardLastOwner(ctx context.Context, q *gen.Queries) error {
	n, err := q.AdminCountActiveOwners(ctx)
	if err != nil {
		return fmt.Errorf("count owners: %w", err)
	}
	if n <= 1 {
		return ErrLastOwner
	}
	return nil
}

// guardSeat refuses when adding one more active learner would exceed seatCap (the
// caller holds the seats lock). It returns the seats in use.
func guardSeat(ctx context.Context, q *gen.Queries, seatCap int) (int64, error) {
	used, err := q.AdminCountSeatsUsed(ctx)
	if err != nil {
		return 0, fmt.Errorf("count seats: %w", err)
	}
	if used+1 > int64(seatCap) {
		return used, ErrSeatCap
	}
	return used, nil
}

func validRole(r string) bool { return r == RoleLearner || r == RoleTester || r == RoleOwner }

func adminAccount(a gen.IdentityAccount) AdminAccount {
	return AdminAccount{
		ID:          uuidString(a.ID),
		Email:       a.Email.String,
		Username:    a.Username.String,
		Role:        a.Role,
		Status:      a.Status,
		AdmittedVia: a.AdmittedVia.String,
		CreatedAt:   a.CreatedAt.Time,
	}
}

// nullable renders an empty filter as JSON null in the audit detail.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
