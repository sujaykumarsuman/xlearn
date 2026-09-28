package events

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

// Client options, all optional env (ADR-0035 §1 item 4). Unset, a connection is
// exactly the v1 anonymous one, so they are a no-op against today's server.
const (
	// EnvNkeySeedFile is the path of the service's nkey user seed (mounted from its
	// SOPS secret at N2). When set the connection authenticates with that nkey.
	EnvNkeySeedFile = "NATS_NKEY_SEED_FILE"
	// EnvInboxPrefix overrides the reply-inbox prefix. Under the per-service ACL a
	// service may subscribe only to _INBOX_<svc>.>, so it defaults to DefaultInboxPrefix
	// when a seed is set.
	EnvInboxPrefix = "NATS_INBOX_PREFIX"
)

// DefaultInboxPrefix is the inbox prefix a service uses once it has a seed: the only
// subscribe grant it holds under the rendered ACL.
func DefaultInboxPrefix(svc string) string { return "_INBOX_" + svc }

// connName is the per-connection name /connz shows: xlearn-<svc>:<stream>:pub|cons.
func connName(svc, stream, role string) string {
	return fmt.Sprintf("xlearn-%s:%s:%s", svc, stream, role)
}

// Dial connects svc to the NATS server at url with the shared client options:
//   - NATS_NKEY_SEED_FILE → nkey auth from the seed file;
//   - NATS_INBOX_PREFIX → a custom inbox prefix (default _INBOX_<svc> with a seed);
//   - an ErrorHandler that logs permission violations at ERROR (with the subject) and
//     every other async error at WARN;
//   - auto-reconnect forever, as in v1, so a broker outage is transparent.
//
// extra options are applied last (the constructors pass the connection name). An
// error here is a configuration error (a missing or bad seed, a bad prefix): callers
// with NATS_URL set fail closed on it. A reachable-or-not broker never errors, since
// the connection retries in the background.
func Dial(ctx context.Context, svc, url string, log *slog.Logger, extra ...nats.Option) (*nats.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if svc == "" {
		return nil, errors.New("events: Dial needs the service name")
	}
	opts := []nats.Option{
		nats.Name("xlearn-" + svc),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2 * time.Second),
		nats.ErrorHandler(errorHandler(svc, log)),
	}

	seed := strings.TrimSpace(os.Getenv(EnvNkeySeedFile))
	prefix := strings.TrimSpace(os.Getenv(EnvInboxPrefix))
	if seed != "" {
		nkeyOpt, err := nats.NkeyOptionFromSeed(seed)
		if err != nil {
			return nil, fmt.Errorf("events: %s: %w", EnvNkeySeedFile, err)
		}
		opts = append(opts, nkeyOpt)
		if prefix == "" {
			prefix = DefaultInboxPrefix(svc)
			log.Info("nats inbox prefix defaulted for nkey auth", "svc", svc, "inbox_prefix", prefix)
		}
	}
	if prefix != "" {
		opts = append(opts, nats.CustomInboxPrefix(prefix))
	}
	opts = append(opts, extra...)

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats connect %q: %w", url, err)
	}
	return nc, nil
}

// permSubjectRe pulls the subject out of a server permissions-violation error, e.g.
// `nats: permissions violation: Permissions Violation for Publish to "xlearn.review.x"`.
var permSubjectRe = regexp.MustCompile(`"([^"]+)"`)

// errorHandler is the async error callback: a permission violation (a publish or
// subscribe the ACL denies) is an ERROR naming the subject — it is how a missing ACL
// grant shows after N1 (ADR-0035 §2) — and anything else is a WARN.
func errorHandler(svc string, log *slog.Logger) nats.ErrHandler {
	return func(nc *nats.Conn, sub *nats.Subscription, err error) {
		name := ""
		if nc != nil {
			name = nc.Opts.Name
		}
		if errors.Is(err, nats.ErrPermissionViolation) {
			subject := ""
			if m := permSubjectRe.FindStringSubmatch(err.Error()); len(m) == 2 {
				subject = m[1]
			} else if sub != nil {
				subject = sub.Subject
			}
			log.Error("nats permission violation", "svc", svc, "conn", name, "subject", subject, "err", err)
			return
		}
		attrs := []any{"svc", svc, "conn", name, "err", err}
		if sub != nil {
			attrs = append(attrs, "subject", sub.Subject)
		}
		log.Warn("nats async error", attrs...)
	}
}
