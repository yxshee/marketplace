package router

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type rateVisitor struct {
	limiter *rate.Limiter
	seenAt  time.Time
}

type requestRateLimiter struct {
	mu         sync.Mutex
	visitors   map[string]*rateVisitor
	limit      rate.Limit
	burst      int
	ttl        time.Duration
	maxEntries int
}

func newRequestRateLimiter(rps, burst int, ttl time.Duration) *requestRateLimiter {
	if rps <= 0 {
		rps = 1
	}
	if burst <= 0 {
		burst = rps
	}
	if ttl <= 0 {
		ttl = time.Minute
	}

	return &requestRateLimiter{
		visitors:   make(map[string]*rateVisitor),
		limit:      rate.Limit(rps),
		burst:      burst,
		ttl:        ttl,
		maxEntries: 50_000,
	}
}

func (l *requestRateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(requestIPKey(r)) {
			w.Header().Set("Retry-After", "1")
			writeError(w, http.StatusTooManyRequests, "too many requests, please retry shortly")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *requestRateLimiter) allow(key string) bool {
	now := time.Now().UTC()

	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.visitors) > l.maxEntries {
		l.pruneExpired(now)
	}

	visitor, ok := l.visitors[key]
	if !ok {
		visitor = &rateVisitor{
			limiter: rate.NewLimiter(l.limit, l.burst),
			seenAt:  now,
		}
		l.visitors[key] = visitor
	} else {
		visitor.seenAt = now
	}

	return visitor.limiter.Allow()
}

func (l *requestRateLimiter) pruneExpired(now time.Time) {
	for key, visitor := range l.visitors {
		if now.Sub(visitor.seenAt) > l.ttl {
			delete(l.visitors, key)
		}
	}
}

func requestIPKey(r *http.Request) string {
	remote := strings.TrimSpace(r.RemoteAddr)
	if remote == "" {
		return "unknown"
	}

	host, _, err := net.SplitHostPort(remote)
	if err == nil && strings.TrimSpace(host) != "" {
		return strings.TrimSpace(host)
	}
	return remote
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		next.ServeHTTP(w, r)
	})
}
