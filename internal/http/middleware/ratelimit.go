package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
)

type rateLimitBucket struct {
	windowStart time.Time
	count       int
}

const (
	minRateLimitBuckets = 1024
	maxRateLimitBuckets = 65536
)

func RateLimitByIP(limit int, window time.Duration) func(http.Handler) http.Handler {
	return RateLimitByIPWithTrustedProxies(limit, window, nil)
}

func RateLimitByIPWithTrustedProxies(limit int, window time.Duration, trustedProxies []string) func(http.Handler) http.Handler {
	var (
		mu      sync.Mutex
		buckets = make(map[string]rateLimitBucket)
		now     = time.Now
		proxies = parseTrustedProxies(trustedProxies)
	)

	return rateLimitByIP(limit, window, now, &mu, buckets, proxies)
}

func rateLimitByIP(limit int, window time.Duration, now func() time.Time, mu *sync.Mutex, buckets map[string]rateLimitBucket, trusted trustedProxySet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limit <= 0 || window <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			key := clientIP(r, trusted)
			current := now()

			mu.Lock()
			capacity := rateLimitBucketCapacity(limit)
			if _, exists := buckets[key]; !exists && len(buckets) >= capacity {
				removeExpiredRateLimitBuckets(buckets, current, window)
				if len(buckets) >= capacity {
					removeOldestRateLimitBucket(buckets)
				}
			}
			bucket := buckets[key]
			if bucket.windowStart.IsZero() || current.Sub(bucket.windowStart) >= window {
				bucket = rateLimitBucket{windowStart: current}
			}
			bucket.count++
			buckets[key] = bucket
			allowed := bucket.count <= limit
			mu.Unlock()

			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(max(1, int(window.Seconds()))))
				writeRateLimitError(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitBucketCapacity(limit int) int {
	if limit <= 0 {
		return minRateLimitBuckets
	}
	if limit > maxRateLimitBuckets/8 {
		return maxRateLimitBuckets
	}
	return min(max(limit*8, minRateLimitBuckets), maxRateLimitBuckets)
}

func removeExpiredRateLimitBuckets(buckets map[string]rateLimitBucket, current time.Time, window time.Duration) {
	for key, bucket := range buckets {
		if current.Sub(bucket.windowStart) >= window {
			delete(buckets, key)
		}
	}
}

func removeOldestRateLimitBucket(buckets map[string]rateLimitBucket) {
	var oldestKey string
	var oldest time.Time
	for key, bucket := range buckets {
		if oldestKey == "" || bucket.windowStart.Before(oldest) {
			oldestKey = key
			oldest = bucket.windowStart
		}
	}
	delete(buckets, oldestKey)
}

type trustedProxySet struct {
	prefixes []netip.Prefix
}

func parseTrustedProxies(values []string) trustedProxySet {
	trusted := trustedProxySet{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if prefix, err := netip.ParsePrefix(value); err == nil {
			trusted.prefixes = append(trusted.prefixes, prefix)
			continue
		}
		if addr, err := netip.ParseAddr(value); err == nil {
			trusted.prefixes = append(trusted.prefixes, netip.PrefixFrom(addr, addr.BitLen()))
			continue
		}
		slog.Warn("invalid trusted proxy ignored", "trusted_proxy", value)
	}
	return trusted
}

func (s trustedProxySet) contains(addr netip.Addr) bool {
	for _, prefix := range s.prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func clientIP(r *http.Request, trusted trustedProxySet) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		if remoteAddr, parseErr := netip.ParseAddr(host); parseErr == nil && trusted.contains(remoteAddr) {
			if forwarded := forwardedClientIP(r, trusted); forwarded != "" {
				return forwarded
			}
		}
		return host
	}
	return r.RemoteAddr
}

func forwardedClientIP(r *http.Request, trusted trustedProxySet) string {
	parts := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(parts) - 1; i >= 0; i-- {
		addr, err := netip.ParseAddr(strings.TrimSpace(parts[i]))
		if err == nil && !trusted.contains(addr) {
			return addr.String()
		}
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if addr, err := netip.ParseAddr(realIP); err == nil {
		return addr.String()
	}
	return ""
}

func writeRateLimitError(w http.ResponseWriter) {
	writeJSONError(w, http.StatusTooManyRequests, "rate_limited", "Rate limit exceeded", nil)
}
