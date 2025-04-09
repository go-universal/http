package limiter

import (
	"strconv"
	"strings"
	"time"

	unicache "github.com/go-universal/cache"
	"github.com/gofiber/fiber/v2"
)

// NewMiddleware creates a new rate limiting middleware for Fiber framework.
// It accepts a cache instance and optional configuration options.
// The middleware limits the number of requests a client can make within a specified time period.
func NewMiddleware(cache unicache.Cache, options ...Option) fiber.Handler {
	// Generate option
	option := &option{
		key:      "limiter",
		attempts: 100,
		ttl:      time.Minute,
		fail:     nil,
		next:     nil,
		keys:     nil,
	}
	for _, opt := range options {
		opt(option)
	}

	return func(c *fiber.Ctx) error {
		// Skip
		if option.next != nil && option.next(c) {
			return c.Next()
		}

		// Create limiter
		key := option.key + "-" + c.IP()
		if option.keys != nil {
			for _, k := range option.keys(c) {
				k = strings.TrimSpace(k)
				if k != "" {
					key += "-" + k
				}
			}
		}
		limiter := unicache.NewRateLimiter(
			key,
			uint32(option.attempts),
			option.ttl,
			cache,
		)

		// Lock request
		if lock, err := limiter.MustLock(); err != nil {
			return err
		} else if lock {
			until, err := limiter.AvailableIn()
			if err != nil {
				return err
			}

			c.Append("Access-Control-Expose-Headers", "X-LIMIT-UNTIL")
			c.Set("X-LIMIT-UNTIL", until.String())
			if option.fail != nil {
				return option.fail(until)(c)
			}

			return c.SendStatus(fiber.StatusTooManyRequests)
		}

		// Move on
		err := c.Next()

		// Hit tries
		if !option.skipFail || (option.skipFail && err == nil) {
			err := limiter.Hit()
			if err != nil {
				return err
			}
		}

		// Send left retries to client
		if left, err := limiter.RetriesLeft(); err != nil {
			return err
		} else {
			c.Append("Access-Control-Expose-Headers", "X-LIMIT-REMAIN")
			c.Set("X-LIMIT-REMAIN", strconv.Itoa(int(left)))
		}

		return err
	}
}
