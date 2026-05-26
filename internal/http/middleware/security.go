package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

type SecurityHeadersConfig struct {
	PublicOrigin string
	CookieSecure bool
}

func SecurityHeaders(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	hsts := false
	if cfg.CookieSecure {
		if publicOrigin, err := url.Parse(cfg.PublicOrigin); err == nil && publicOrigin.Scheme == "https" {
			hsts = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'self'")
			if hsts {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

type HostPolicyConfig struct {
	AppEnv             string
	PublicOrigin       string
	PublicOriginHost   string
	AllowDirectIPLogin bool
}

func AuthHostPolicy(cfg HostPolicyConfig) func(http.Handler) http.Handler {
	production := strings.EqualFold(strings.TrimSpace(cfg.AppEnv), "production")
	publicOrigin := strings.TrimSpace(cfg.PublicOrigin)
	publicHost := strings.TrimSpace(cfg.PublicOriginHost)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authSensitivePath(r.URL.Path) || !production {
				next.ServeHTTP(w, r)
				return
			}
			if publicOrigin == "" || publicHost == "" {
				writeJSONError(w, http.StatusServiceUnavailable, "security_not_configured", "Security origin is not configured", nil)
				return
			}
			if !hostMatches(r.Host, publicHost) {
				writeJSONError(w, http.StatusForbidden, "forbidden_host", "Forbidden host", nil)
				return
			}
			if hostIsIP(r.Host) && (!cfg.AllowDirectIPLogin || !hostIsLoopback(r.Host)) {
				writeJSONError(w, http.StatusForbidden, "forbidden_host", "Forbidden host", nil)
				return
			}
			if !originHeadersAllowed(r, publicOrigin) {
				writeJSONError(w, http.StatusForbidden, "forbidden_origin", "Forbidden origin", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func authSensitivePath(path string) bool {
	switch path {
	case "/auth/setup", "/auth/login", "/auth/refresh", "/auth/logout",
		"/api/v1/auth/password", "/api/v1/auth/sessions":
		return true
	default:
		return strings.HasPrefix(path, "/api/v1/auth/sessions/")
	}
}

func hostMatches(requestHost, publicHost string) bool {
	requestHost = strings.TrimSpace(requestHost)
	publicHost = strings.TrimSpace(publicHost)
	if requestHost == "" || publicHost == "" {
		return false
	}
	if strings.EqualFold(requestHost, publicHost) {
		return true
	}
	requestName, _, requestErr := net.SplitHostPort(requestHost)
	publicName, _, publicErr := net.SplitHostPort(publicHost)
	if requestErr == nil && publicErr == nil {
		return strings.EqualFold(requestName, publicName)
	}
	if requestErr == nil {
		return strings.EqualFold(requestName, publicHost)
	}
	if publicErr == nil {
		return strings.EqualFold(requestHost, publicName)
	}
	return false
}

func originHeadersAllowed(r *http.Request, publicOrigin string) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" && !strings.EqualFold(origin, publicOrigin) {
		return false
	}
	referer := strings.TrimSpace(r.Header.Get("Referer"))
	if referer == "" {
		return true
	}
	refererURL, err := url.Parse(referer)
	if err != nil || refererURL.Scheme == "" || refererURL.Host == "" {
		return false
	}
	return strings.EqualFold(refererURL.Scheme+"://"+refererURL.Host, publicOrigin)
}

func hostIsIP(host string) bool {
	host = strings.TrimSpace(host)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	return net.ParseIP(host) != nil
}

func hostIsLoopback(host string) bool {
	host = strings.TrimSpace(host)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
