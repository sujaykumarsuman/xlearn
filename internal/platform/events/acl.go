package events

import (
	"fmt"
	"slices"
	"strings"

	"github.com/nats-io/nkeys"
)

// This file renders the NATS `authorization` block from topology.go (ADR-0035 §2): one
// nkey user per service with the fine per-service ACL, the offline `ops` identity, and
// the optional `legacy` bridge user that anonymous (v1.5.2-era, seedless) clients map
// to through no_auth_user during the server-first rollout (N1 allow → N3 deny → N4
// none). The same model renders two forms: NATS server conf (the integration test boots
// it) and the values fragment for the pinned `nats` chart (mi-06's N1 PR pastes it).

// LegacyMode selects the `legacy` bridge stage.
type LegacyMode string

const (
	// LegacyAllow (N1): legacy allows ">" and no_auth_user maps anonymous clients to it.
	LegacyAllow LegacyMode = "allow"
	// LegacyDeny (N3): legacy denies ">", so an anonymous client connects but can do nothing.
	LegacyDeny LegacyMode = "deny"
	// LegacyNone (N4): no legacy user and no no_auth_user; anonymous connects are refused.
	LegacyNone LegacyMode = "none"
)

// ParseLegacyMode parses allow|deny|none.
func ParseLegacyMode(s string) (LegacyMode, error) {
	switch m := LegacyMode(strings.ToLower(strings.TrimSpace(s))); m {
	case LegacyAllow, LegacyDeny, LegacyNone:
		return m, nil
	}
	return "", fmt.Errorf("events: LEGACY must be allow, deny or none, got %q", s)
}

// Identities rendered into the authorization block.
const (
	// OpsIdentity is the break-glass identity (the nats CLI with the offline seed):
	// all of $JS.API.> (purge included) and the default _INBOX.> (ADR-0035 §2).
	OpsIdentity = "ops"
	// LegacyUser is the bridge user no_auth_user points at.
	LegacyUser = "legacy"
	// LegacyPasswordPlaceholder is the golden's legacy password. `make nats-acl-render`
	// replaces it with a random one no client ever uses (no_auth_user needs a user).
	LegacyPasswordPlaceholder = "<LEGACY_PASSWORD>"
)

// ACLServices is the render order of the service identities: the six stream owners.
func ACLServices() []string {
	out := make([]string, 0, len(streams))
	for _, s := range streams {
		out = append(out, s.Owner)
	}
	return out
}

// ACLIdentities is every identity that needs an nkey: the six services plus ops.
func ACLIdentities() []string { return append(ACLServices(), OpsIdentity) }

// PlaceholderKey is the golden's stand-in for an identity's public nkey.
func PlaceholderKey(identity string) string { return "<NKEY_PUB:" + identity + ">" }

// PlaceholderKeys maps every identity to its placeholder (the golden render input).
func PlaceholderKeys() map[string]string {
	keys := map[string]string{}
	for _, id := range ACLIdentities() {
		keys[id] = PlaceholderKey(id)
	}
	return keys
}

// Permissions is one user's publish/subscribe grants.
type Permissions struct {
	PublishAllow, PublishDeny, SubscribeAllow, SubscribeDeny []string
}

// ACLUser is one entry of authorization.users: an nkey user (NKey) or the legacy
// password user (User/Password). Comment is rendered above the entry.
type ACLUser struct {
	Comment        string
	NKey           string
	User, Password string
	Permissions    Permissions
}

// Authorization is the rendered block: the users plus the optional no_auth_user.
type Authorization struct {
	Legacy     LegacyMode
	Users      []ACLUser
	NoAuthUser string
}

// neverPublish is the explicit "never" row of the ADR-0035 §2 table. The allow list
// already excludes these; the deny keeps them out if an allow is ever widened.
var neverPublish = []string{
	"$JS.API.STREAM.DELETE.>",
	"$JS.API.STREAM.PURGE.>",
	"$JS.API.STREAM.MSG.DELETE.>",
}

// ServicePermissions renders svc's grants from topology.go (ADR-0035 §2 ACL table):
// publish its events and manage its own stream, and for each durable it consumes
// create (filtered), inspect, fetch from and ack that durable only; subscribe only to
// its own inbox prefix.
func ServicePermissions(svc string) Permissions {
	pub := []string{"xlearn." + svc + ".>", "$JS.API.INFO"}
	for _, s := range streams {
		if s.Owner == svc {
			pub = append(pub,
				"$JS.API.STREAM.CREATE."+s.Name,
				"$JS.API.STREAM.UPDATE."+s.Name,
				"$JS.API.STREAM.INFO."+s.Name,
			)
		}
	}
	for _, d := range durables {
		if d.Service != svc {
			continue
		}
		sd := d.Stream + "." + d.Name
		pub = append(pub,
			"$JS.API.CONSUMER.CREATE."+sd,
			"$JS.API.CONSUMER.CREATE."+sd+".>", // a filtered create carries the filter subject
			"$JS.API.CONSUMER.INFO."+sd,
			"$JS.API.CONSUMER.MSG.NEXT."+sd,
			"$JS.ACK."+sd+".>",
		)
	}
	return Permissions{
		PublishAllow:   pub,
		PublishDeny:    slices.Clone(neverPublish),
		SubscribeAllow: []string{DefaultInboxPrefix(svc) + ".>"},
	}
}

// OpsPermissions is the break-glass identity's grant.
func OpsPermissions() Permissions {
	return Permissions{PublishAllow: []string{"$JS.API.>"}, SubscribeAllow: []string{"_INBOX.>"}}
}

// RenderAuthorization renders the block for keys (identity → public nkey; every
// service and ops is required) and the legacy stage. A key is either a valid user
// public key or its PlaceholderKey (the golden). The legacy password is the
// placeholder; SetLegacyPassword replaces it.
func RenderAuthorization(keys map[string]string, legacy LegacyMode) (*Authorization, error) {
	if _, err := ParseLegacyMode(string(legacy)); err != nil {
		return nil, err
	}
	a := &Authorization{Legacy: legacy}
	for _, id := range ACLIdentities() {
		key := strings.TrimSpace(keys[id])
		if key == "" {
			return nil, fmt.Errorf("events: no public nkey for %q", id)
		}
		if key != PlaceholderKey(id) && !nkeys.IsValidPublicUserKey(key) {
			return nil, fmt.Errorf("events: %q is not a user public nkey (want U…)", id)
		}
		u := ACLUser{NKey: key}
		if id == OpsIdentity {
			u.Comment = "ops: break-glass only (offline seed); all of $JS.API.>, purge included"
			u.Permissions = OpsPermissions()
		} else {
			u.Comment = id + ": " + ownerComment(id)
			u.Permissions = ServicePermissions(id)
		}
		a.Users = append(a.Users, u)
	}
	for id := range keys {
		if !slices.Contains(ACLIdentities(), id) {
			return nil, fmt.Errorf("events: unknown identity %q (want one of %s)", id, strings.Join(ACLIdentities(), ", "))
		}
	}
	switch legacy {
	case LegacyAllow:
		a.Users = append(a.Users, ACLUser{
			Comment: "legacy (N1): anonymous clients map here via no_auth_user; the password is never used",
			User:    LegacyUser, Password: LegacyPasswordPlaceholder,
			Permissions: Permissions{PublishAllow: []string{">"}, SubscribeAllow: []string{">"}},
		})
		a.NoAuthUser = LegacyUser
	case LegacyDeny:
		a.Users = append(a.Users, ACLUser{
			Comment: "legacy (N3): anonymous clients still map here but may do nothing",
			User:    LegacyUser, Password: LegacyPasswordPlaceholder,
			Permissions: Permissions{PublishDeny: []string{">"}, SubscribeDeny: []string{">"}},
		})
		a.NoAuthUser = LegacyUser
	}
	return a, nil
}

// SetLegacyPassword replaces the legacy user's placeholder password (no-op at N4).
func (a *Authorization) SetLegacyPassword(pw string) {
	for i := range a.Users {
		if a.Users[i].User == LegacyUser {
			a.Users[i].Password = pw
		}
	}
}

func ownerComment(svc string) string {
	parts := []string{}
	for _, s := range streams {
		if s.Owner == svc {
			parts = append(parts, "owns "+s.Name)
		}
	}
	for _, d := range durables {
		if d.Service == svc {
			parts = append(parts, "consumes "+d.Stream+"/"+d.Name)
		}
	}
	return strings.Join(parts, "; ")
}

// renderHeader is the shared provenance comment of both forms.
func renderHeader(b *strings.Builder, legacy LegacyMode, extra ...string) {
	b.WriteString("# Code generated from internal/platform/events/topology.go by acl.go\n")
	b.WriteString("# (make nats-acl-render; golden: go test ./internal/platform/events -run Golden -update).\n")
	b.WriteString("# DO NOT EDIT. ADR-0035 §2 ACL table. LEGACY=" + string(legacy) + ".\n")
	b.WriteString("# Every $-prefixed subject is quoted: NATS conf reads an unquoted $x as a variable.\n")
	for _, l := range extra {
		b.WriteString("# " + l + "\n")
	}
}

// Conf renders the block as NATS server configuration.
func (a *Authorization) Conf() string {
	var b strings.Builder
	renderHeader(&b, a.Legacy)
	b.WriteString("authorization {\n  users = [\n")
	for i, u := range a.Users {
		b.WriteString("    # " + u.Comment + "\n    {\n")
		if u.NKey != "" {
			b.WriteString("      nkey: " + quote(u.NKey) + "\n")
		} else {
			b.WriteString("      user: " + quote(u.User) + "\n")
			b.WriteString("      password: " + quote(u.Password) + "\n")
		}
		b.WriteString("      permissions: {\n")
		confDirection(&b, "publish", u.Permissions.PublishAllow, u.Permissions.PublishDeny)
		confDirection(&b, "subscribe", u.Permissions.SubscribeAllow, u.Permissions.SubscribeDeny)
		b.WriteString("      }\n    }")
		if i < len(a.Users)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("  ]\n}\n")
	if a.NoAuthUser != "" {
		b.WriteString("no_auth_user: " + quote(a.NoAuthUser) + "\n")
	}
	return b.String()
}

func confDirection(b *strings.Builder, name string, allow, deny []string) {
	if len(allow) == 0 && len(deny) == 0 {
		return
	}
	b.WriteString("        " + name + ": {\n")
	confList(b, "allow", allow)
	confList(b, "deny", deny)
	b.WriteString("        }\n")
}

func confList(b *strings.Builder, name string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("          " + name + ": [\n")
	for i, s := range items {
		b.WriteString("            " + quote(s))
		if i < len(items)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("          ]\n")
}

// ChartValues renders the block as the values fragment for the `nats` chart pinned in
// ../infra infrastructure/messaging/release.yaml: config.merge is merged into the
// chart's nats.conf (rendered as JSON, so every string stays quoted).
func (a *Authorization) ChartValues() string {
	var b strings.Builder
	renderHeader(&b, a.Legacy,
		"Values fragment for the nats chart (../infra infrastructure/messaging/release.yaml):",
		"merge `merge:` under spec.values.config (the chart's config.merge key).")
	b.WriteString("config:\n  merge:\n    authorization:\n      users:\n")
	for _, u := range a.Users {
		b.WriteString("        # " + u.Comment + "\n")
		if u.NKey != "" {
			b.WriteString("        - nkey: " + quote(u.NKey) + "\n")
		} else {
			b.WriteString("        - user: " + quote(u.User) + "\n")
			b.WriteString("          password: " + quote(u.Password) + "\n")
		}
		b.WriteString("          permissions:\n")
		yamlDirection(&b, "publish", u.Permissions.PublishAllow, u.Permissions.PublishDeny)
		yamlDirection(&b, "subscribe", u.Permissions.SubscribeAllow, u.Permissions.SubscribeDeny)
	}
	if a.NoAuthUser != "" {
		b.WriteString("    no_auth_user: " + quote(a.NoAuthUser) + "\n")
	}
	return b.String()
}

func yamlDirection(b *strings.Builder, name string, allow, deny []string) {
	if len(allow) == 0 && len(deny) == 0 {
		return
	}
	b.WriteString("            " + name + ":\n")
	yamlList(b, "allow", allow)
	yamlList(b, "deny", deny)
}

func yamlList(b *strings.Builder, name string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("              " + name + ":\n")
	for _, s := range items {
		b.WriteString("                - " + quote(s) + "\n")
	}
}

// quote double-quotes s for both NATS conf and YAML. Every rendered value is a
// subject, key, user or password made of printable ASCII without quotes or
// backslashes, so no escaping is needed; anything else is a bug and panics.
func quote(s string) string {
	for _, r := range s {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			panic(fmt.Sprintf("events: unquotable ACL value %q", s))
		}
	}
	return `"` + s + `"`
}
