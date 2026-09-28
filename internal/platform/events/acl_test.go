package events

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nkeys"
	"gopkg.in/yaml.v3"
)

// update rewrites the ACL goldens: go test ./internal/platform/events -run Golden -update
var update = flag.Bool("update", false, "rewrite the ACL golden files under testdata/")

const (
	goldenConf = "testdata/nats-authorization.golden.conf"
	goldenYAML = "testdata/nats-authorization.golden.yaml"
)

// TestACLGolden pins both renders (NATS conf and the nats chart values fragment) with
// placeholder keys at LEGACY=allow — the N1 form mi-06 pastes. A topology.go change
// that isn't re-rendered fails here, which keeps the infra ACL PR in step with the code.
func TestACLGolden(t *testing.T) {
	a, err := RenderAuthorization(PlaceholderKeys(), LegacyAllow)
	if err != nil {
		t.Fatal(err)
	}
	for path, got := range map[string]string{goldenConf: a.Conf(), goldenYAML: a.ChartValues()} {
		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read golden (regenerate with -update): %v", err)
		}
		if string(want) != got {
			t.Errorf("%s is stale: topology.go or acl.go changed. Regenerate with\n"+
				"  go test ./internal/platform/events -run Golden -update\n"+
				"and ship the re-rendered block in the infra ACL PR before the consuming tag (ADR-0035 §2).", path)
		}
	}
}

// TestACLDollarQuoted asserts every $-prefixed subject is quoted in both forms: NATS
// conf reads an unquoted $x as a variable reference.
func TestACLDollarQuoted(t *testing.T) {
	for _, mode := range []LegacyMode{LegacyAllow, LegacyDeny, LegacyNone} {
		a, err := RenderAuthorization(PlaceholderKeys(), mode)
		if err != nil {
			t.Fatal(err)
		}
		for form, text := range map[string]string{"conf": a.Conf(), "yaml": a.ChartValues()} {
			n := 0
			for i, line := range strings.Split(text, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "#") {
					continue
				}
				for j := strings.IndexByte(line, '$'); j >= 0; j = nextDollar(line, j) {
					n++
					if j == 0 || line[j-1] != '"' {
						t.Errorf("%s %s line %d: unquoted $ subject: %s", mode, form, i+1, line)
					}
				}
			}
			if n == 0 {
				t.Errorf("%s %s: no $ subjects rendered at all", mode, form)
			}
		}
	}
}

func nextDollar(s string, j int) int {
	k := strings.IndexByte(s[j+1:], '$')
	if k < 0 {
		return -1
	}
	return j + 1 + k
}

// TestACLConfLoads parses the conf render with the NATS server's own config loader
// (the 2.14 module), with real throwaway keys, in all three legacy stages: the block
// is valid server configuration, the $ subjects come through literally, and the
// legacy / no_auth_user shape matches the stage.
func TestACLConfLoads(t *testing.T) {
	keys, _ := throwawayKeys(t)
	for _, mode := range []LegacyMode{LegacyAllow, LegacyDeny, LegacyNone} {
		t.Run(string(mode), func(t *testing.T) {
			a, err := RenderAuthorization(keys, mode)
			if err != nil {
				t.Fatal(err)
			}
			a.SetLegacyPassword("not-a-real-password")
			path := filepath.Join(t.TempDir(), "nats.conf")
			if err := os.WriteFile(path, []byte(a.Conf()), 0o600); err != nil {
				t.Fatal(err)
			}
			opts, err := server.ProcessConfigFile(path)
			if err != nil {
				t.Fatalf("server rejects the render: %v", err)
			}
			if got, want := len(opts.Nkeys), len(ACLIdentities()); got != want {
				t.Errorf("%d nkey users, want %d", got, want)
			}
			byKey := map[string]*server.NkeyUser{}
			for _, u := range opts.Nkeys {
				byKey[u.Nkey] = u
			}
			practice := byKey[keys["practice"]]
			if practice == nil || practice.Permissions == nil || practice.Permissions.Publish == nil {
				t.Fatal("practice user or its publish permissions missing")
			}
			if !slices.Contains(practice.Permissions.Publish.Allow, "$JS.API.STREAM.UPDATE.XLEARN_PRACTICE") {
				t.Errorf("practice publish allow = %v; $ subjects must come through literally", practice.Permissions.Publish.Allow)
			}
			if !slices.Contains(practice.Permissions.Publish.Deny, "$JS.API.STREAM.PURGE.>") {
				t.Errorf("practice publish deny = %v", practice.Permissions.Publish.Deny)
			}
			switch mode {
			case LegacyNone:
				if len(opts.Users) != 0 || opts.NoAuthUser != "" {
					t.Errorf("N4 render still has users %d / no_auth_user %q", len(opts.Users), opts.NoAuthUser)
				}
			default:
				if len(opts.Users) != 1 || opts.Users[0].Username != LegacyUser || opts.NoAuthUser != LegacyUser {
					t.Fatalf("legacy user / no_auth_user missing: users %d, no_auth_user %q", len(opts.Users), opts.NoAuthUser)
				}
				p := opts.Users[0].Permissions
				if mode == LegacyAllow && !slices.Equal(p.Publish.Allow, []string{">"}) {
					t.Errorf("N1 legacy publish = %+v, want allow >", p.Publish)
				}
				if mode == LegacyDeny && !slices.Equal(p.Publish.Deny, []string{">"}) {
					t.Errorf("N3 legacy publish = %+v, want deny >", p.Publish)
				}
			}
		})
	}
}

// TestACLChartValuesShape parses the chart fragment and checks it carries the same
// users and grants as the model, under config.merge (the pinned chart's merge key).
func TestACLChartValuesShape(t *testing.T) {
	a, err := RenderAuthorization(PlaceholderKeys(), LegacyAllow)
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Config struct {
			Merge struct {
				Authorization struct {
					Users []struct {
						NKey        string `yaml:"nkey"`
						User        string `yaml:"user"`
						Password    string `yaml:"password"`
						Permissions struct {
							Publish, Subscribe struct{ Allow, Deny []string }
						} `yaml:"permissions"`
					} `yaml:"users"`
				} `yaml:"authorization"`
				NoAuthUser string `yaml:"no_auth_user"`
			} `yaml:"merge"`
		} `yaml:"config"`
	}
	if err := yaml.Unmarshal([]byte(a.ChartValues()), &v); err != nil {
		t.Fatalf("chart values don't parse: %v", err)
	}
	users := v.Config.Merge.Authorization.Users
	if len(users) != len(a.Users) || v.Config.Merge.NoAuthUser != LegacyUser {
		t.Fatalf("%d users / no_auth_user %q, want %d / legacy", len(users), v.Config.Merge.NoAuthUser, len(a.Users))
	}
	for i, u := range users {
		m := a.Users[i]
		if u.NKey != m.NKey || u.User != m.User || u.Password != m.Password ||
			!slices.Equal(u.Permissions.Publish.Allow, m.Permissions.PublishAllow) ||
			!slices.Equal(u.Permissions.Publish.Deny, m.Permissions.PublishDeny) ||
			!slices.Equal(u.Permissions.Subscribe.Allow, m.Permissions.SubscribeAllow) ||
			!slices.Equal(u.Permissions.Subscribe.Deny, m.Permissions.SubscribeDeny) {
			t.Errorf("chart user %d differs from the model: %+v vs %+v", i, u, m)
		}
	}
}

// TestServicePermissions spot-checks the ADR-0035 §2 table against topology.go.
func TestServicePermissions(t *testing.T) {
	review := ServicePermissions("review")
	for _, want := range []string{
		"xlearn.review.>", "$JS.API.INFO",
		"$JS.API.STREAM.CREATE.XLEARN_REVIEW", "$JS.API.STREAM.UPDATE.XLEARN_REVIEW", "$JS.API.STREAM.INFO.XLEARN_REVIEW",
		"$JS.API.CONSUMER.CREATE.XLEARN_PRACTICE.review", "$JS.API.CONSUMER.CREATE.XLEARN_PRACTICE.review.>",
		"$JS.API.CONSUMER.INFO.XLEARN_PRACTICE.review", "$JS.API.CONSUMER.MSG.NEXT.XLEARN_PRACTICE.review",
		"$JS.ACK.XLEARN_PRACTICE.review.>",
		"$JS.API.CONSUMER.CREATE.XLEARN_REVIEW.notifications.>", "$JS.ACK.XLEARN_REVIEW.notifications.>",
	} {
		if !slices.Contains(review.PublishAllow, want) {
			t.Errorf("review lacks %s", want)
		}
	}
	for _, p := range review.PublishAllow {
		if strings.Contains(p, "XLEARN_PRACTICE.assessment") || strings.Contains(p, "STREAM.UPDATE.XLEARN_PRACTICE") ||
			p == "$JS.API.>" || p == ">" {
			t.Errorf("review must not be granted %s", p)
		}
	}
	if !slices.Equal(review.SubscribeAllow, []string{"_INBOX_review.>"}) {
		t.Errorf("review subscribe = %v", review.SubscribeAllow)
	}
	judge := ServicePermissions("judge")
	for _, p := range judge.PublishAllow {
		if strings.Contains(p, "CONSUMER") || strings.Contains(p, "$JS.ACK") {
			t.Errorf("judge has no durable yet, but is granted %s", p)
		}
	}
	if !slices.Contains(judge.PublishAllow, "$JS.API.STREAM.UPDATE.XLEARN_JUDGE") {
		t.Error("judge must manage its own stream")
	}
}

func TestRenderAuthorizationRejectsBadKeys(t *testing.T) {
	keys := PlaceholderKeys()
	delete(keys, "coach")
	if _, err := RenderAuthorization(keys, LegacyAllow); err == nil {
		t.Error("a missing identity must fail")
	}
	keys = PlaceholderKeys()
	keys["practice"] = "SUAxxxxxxxx" // a seed-looking value
	if _, err := RenderAuthorization(keys, LegacyAllow); err == nil {
		t.Error("a non-public key must fail")
	}
	keys = PlaceholderKeys()
	keys["gateway"] = "<NKEY_PUB:gateway>"
	if _, err := RenderAuthorization(keys, LegacyAllow); err == nil {
		t.Error("an unknown identity must fail")
	}
	if _, err := RenderAuthorization(PlaceholderKeys(), LegacyMode("maybe")); err == nil {
		t.Error("an unknown legacy mode must fail")
	}
}

// throwawayKeys generates a user nkey per identity: public keys for the render, seeds
// for the clients. Nothing here is ever a real credential.
func throwawayKeys(t *testing.T) (pub map[string]string, seeds map[string][]byte) {
	t.Helper()
	pub, seeds = map[string]string{}, map[string][]byte{}
	for _, id := range ACLIdentities() {
		kp, err := nkeys.CreateUser()
		if err != nil {
			t.Fatal(err)
		}
		pk, err := kp.PublicKey()
		if err != nil {
			t.Fatal(err)
		}
		seed, err := kp.Seed()
		if err != nil {
			t.Fatal(err)
		}
		pub[id], seeds[id] = pk, seed
	}
	return pub, seeds
}
