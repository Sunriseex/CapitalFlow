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
	publicOrigin := canonicalOriginString(cfg.PublicOrigin)
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
			directIPHost := ""
			if hostIsIP(r.Host) {
				if !cfg.AllowDirectIPLogin {
					writeJSONError(w, http.StatusForbidden, "forbidden_host", "Forbidden host", nil)
					return
				}
				directIPHost = r.Host
			} else if !hostMatches(r.Host, publicHost) {
				writeJSONError(w, http.StatusForbidden, "forbidden_host", "Forbidden host", nil)
				return
			}
			if !originHeadersAllowed(r, publicOrigin, directIPHost) {
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

func originHeadersAllowed(r *http.Request, publicOrigin, directIPHost string) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" && !originAllowed(origin, publicOrigin, directIPHost) {
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
	return originAllowed(refererURL.Scheme+"://"+refererURL.Host, publicOrigin, directIPHost)
}

func hostIsIP(host string) bool {
	host = strings.TrimSpace(host)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	return net.ParseIP(host) != nil
}

func sameOrigin(left, right string) bool {
	leftOrigin := canonicalOriginString(left)
	rightOrigin := canonicalOriginString(right)
	return leftOrigin != "" && strings.EqualFold(leftOrigin, rightOrigin)
}

func originAllowed(origin, publicOrigin, directIPHost string) bool {
	if sameOrigin(origin, publicOrigin) {
		return true
	}
	return directIPHost != "" && originHostMatches(origin, directIPHost)
}

func originHostMatches(origin, requestHost string) bool {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	originHost, ok := canonicalHostForScheme(parsed.Host, parsed.Scheme)
	if !ok {
		return false
	}
	host, ok := canonicalHostForScheme(requestHost, parsed.Scheme)
	return ok && strings.EqualFold(originHost, host)
}

func canonicalHostForScheme(host, scheme string) (string, bool) {
	parsed, err := url.Parse("//" + strings.TrimSpace(host))
	if err != nil || parsed.Host == "" || parsed.Hostname() == "" {
		return "", false
	}
	port := parsed.Port()
	if port == "" {
		switch scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			return "", false
		}
	}
	return strings.ToLower(parsed.Hostname()) + ":" + port, true
}

func canonicalOriginString(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Host)
	switch {
	case scheme == "https" && strings.HasSuffix(host, ":443"):
		host = strings.TrimSuffix(host, ":443")
	case scheme == "http" && strings.HasSuffix(host, ":80"):
		host = strings.TrimSuffix(host, ":80")
	}
	return scheme + "://" + host
}
