package limit

import (
	"net/http/httptest"
	"testing"
)

// TestClientIPTrustsOnlyXRealIp is the X-Real-Ip trust test (ADR-0035 §4; m1-05 task 1):
// the key is X-Real-Ip (Traefik sets it, externalTrafficPolicy: Local, no CDN); adding
// X-Forwarded-For, Forwarded or True-Client-Ip never changes it; with no X-Real-Ip it is
// the RemoteAddr host (IPv4 and IPv6).
func TestClientIPTrustsOnlyXRealIp(t *testing.T) {
	spoofs := map[string]string{
		"X-Forwarded-For":  "198.51.100.66, 10.0.0.1",
		"Forwarded":        "for=198.51.100.66;proto=https",
		"True-Client-Ip":   "198.51.100.66",
		"X-Client-Ip":      "198.51.100.66",
		"Cf-Connecting-Ip": "198.51.100.66",
	}
	for _, tc := range []struct {
		name       string
		remoteAddr string
		realIP     string
		want       string
	}{
		{"x-real-ip v4", "10.42.0.9:51234", "203.0.113.7", "203.0.113.7"},
		{"x-real-ip v6", "10.42.0.9:51234", "2001:db8::7", "2001:db8::7"},
		{"x-real-ip v6 canonicalised", "10.42.0.9:51234", "2001:0db8:0:0:0:0:0:7", "2001:db8::7"},
		{"x-real-ip trimmed", "10.42.0.9:51234", "  203.0.113.7 ", "203.0.113.7"},
		{"no header: remote v4", "192.0.2.10:40000", "", "192.0.2.10"},
		{"no header: remote v6", "[2001:db8::10]:40000", "", "2001:db8::10"},
		{"no header: remote v6 loopback", "[::1]:40000", "", "::1"},
		{"malformed x-real-ip falls back", "192.0.2.10:40000", "not-an-ip", "192.0.2.10"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/u/ada", nil)
			r.RemoteAddr = tc.remoteAddr
			if tc.realIP != "" {
				r.Header.Set("X-Real-Ip", tc.realIP)
			}
			if got := ClientIP(r); got != tc.want {
				t.Fatalf("ClientIP = %q, want %q", got, tc.want)
			}
			// No other client-supplied header may move the key, alone or together.
			for h, v := range spoofs {
				r.Header.Set(h, v)
				if got := ClientIP(r); got != tc.want {
					t.Fatalf("with %s: ClientIP = %q, want %q", h, got, tc.want)
				}
			}
		})
	}
}
