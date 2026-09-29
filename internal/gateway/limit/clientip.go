package limit

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP is the per-IP limit key: the X-Real-Ip header, else the host of RemoteAddr.
//
// X-Real-Ip is trusted only because Traefik sets it: the traefik Service runs with
// externalTrafficPolicy: Local (../infra infrastructure/configs/traefik-config.yaml) and
// there is no CDN in front, so Traefik sees the real client address and overwrites any
// X-Real-Ip a client sent. In-cluster spoofing (a pod calling the gateway directly) is
// closed by the MI-5a network policy (ADR-0033). NEVER X-Forwarded-For, Forwarded,
// True-Client-Ip or any other client-supplied header: those are appended to, not replaced,
// so their contents are the client's to choose.
//
// The result is a canonical IP string (net.IP.String), so "::1" and "0:0:0:0:0:0:0:1" key
// the same bucket. A malformed X-Real-Ip falls back to RemoteAddr.
func ClientIP(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Real-Ip")); v != "" {
		if ip := net.ParseIP(v); ip != nil {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	host = strings.Trim(host, "[]")
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return host
}
