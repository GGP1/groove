package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/apikey"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/userip"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
	"github.com/pkg/errors"
)

var errTooManyRequests = errors.New("Too Many Requests")

const (
	rlRemaining = "RateLimit-Remaining"
	rlLimit     = "RateLimit-Limit"
	rlReset     = "RateLimit-Reset"
	retryAfter  = "Retry-After"
)

// RateLimiter uses a leaky bucket algorithm for limiting the requests to the API from the same host.
type RateLimiter struct {
	limiter *redis_rate.Limiter
	limit   redis_rate.Limit
}

// NewRateLimiter returns a rate limiter with the configuration values passed.
func NewRateLimiter(config config.RateLimiter, rdb *redis.Client) RateLimiter {
	return RateLimiter{
		limiter: redis_rate.NewLimiter(rdb),
		limit:   redis_rate.PerMinute(config.Rate),
	}
}

// Limit make sure no one abuses the API by using token bucket algorithm.
func (rl RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		rl.limit.Period = time.Minute
		key, err := apikey.FromRequest(r)
		if err != nil {
			if errors.Is(err, apikey.ErrInvalidAPIKey) {
				response.Error(w, http.StatusBadRequest, err)
				return
			}
			// If the user is not using an API token, use ip as key and decrease rate limit
			key = userip.Get(ctx, r)
			if key == "" {
				// Try to avoid this at all cost or an attacker able to hide ips will be able to perform DDOS.
				next.ServeHTTP(w, r)
				return
			}
			rl.limit.Period *= 15
		}

		res, err := rl.limiter.Allow(ctx, key, rl.limit)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		header := w.Header()
		header[rlRemaining] = []string{strconv.Itoa(res.Remaining)}

		if res.Allowed == 0 {
			header[rlLimit] = []string{strconv.Itoa(res.Limit.Rate)}
			header[rlReset] = []string{strconv.Itoa(int(res.ResetAfter / time.Second))}
			header[retryAfter] = []string{strconv.Itoa(int(res.RetryAfter / time.Second))}
			response.Error(w, http.StatusTooManyRequests, errTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
