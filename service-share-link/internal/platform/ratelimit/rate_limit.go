package ratelimit

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"service-share-link/internal/platform/http_request"
	"strconv"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

const (
	HEADER_REAL_IP = "X-Real-IP"
)

// Rule allows Limit requests per Window for each key that Key returns.
// Name keeps the counters of different rules apart.
//
// The limit is enforced with GCRA: a client can spend up to Limit requests at
// once, after which it gets one more every Window/Limit. Unlike a fixed window
// there is no reset moment, so a client cannot fire a full limit at the end of
// one window and another full limit at the start of the next.
type Rule struct {
	Name   string
	Limit  int
	Window time.Duration
	Key    func(r *http.Request) string
}

func (rule Rule) gcra() redis_rate.Limit {
	return redis_rate.Limit{
		Rate:   rule.Limit,
		Burst:  rule.Limit,
		Period: rule.Window,
	}
}

// Limiter keeps its state in Redis, so every instance shares the same limits.
// Redis keys are "rate:{rule name}:{key}".
type Limiter struct {
	limiter *redis_rate.Limiter
}

func NewLimiter(client redis.Cmdable) *Limiter {
	return &Limiter{
		limiter: redis_rate.NewLimiter(client),
	}
}

func (l *Limiter) Allow(ctx context.Context, rule Rule, key string) (*redis_rate.Result, error) {
	res, err := l.limiter.Allow(ctx, rule.Name+":"+key, rule.gcra())
	if err != nil {
		return nil, fmt.Errorf("[RateLimit] Allow rule=%s %w", rule.Name, err)
	}

	return res, nil
}

// Wrap answers 429 once a key goes over the rule's limit. When Redis fails the
// request goes through: a short window without limits hurts less than turning
// every public request into an error.
func (l *Limiter) Wrap(rule Rule, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := l.Allow(r.Context(), rule, rule.Key(r))
		if err != nil {
			log.Printf("[RateLimit] Letting request through: %v", err)
			next(w, r)
			return
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rule.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		if res.Allowed == 0 {
			retryAfterSec := (res.RetryAfter.Milliseconds() + 999) / 1000
			w.Header().Set("Retry-After", strconv.FormatInt(retryAfterSec, 10))
			http_request.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, retry later")
			return
		}

		next(w, r)
	}
}

// ClientIP prefers the address the gateway puts in X-Real-IP. Behind a load
// balancer the TCP peer is the balancer itself, which would put every client
// in one bucket. Like X-User-Id, the header is trusted because only the gateway
// reaches this service.
func ClientIP(r *http.Request) string {
	if ip := net.ParseIP(r.Header.Get(HEADER_REAL_IP)); ip != nil {
		return ip.String()
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
