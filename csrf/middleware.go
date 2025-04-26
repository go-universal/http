package csrf

import (
	"errors"
	"strings"

	"github.com/go-universal/http/session"
	"github.com/gofiber/fiber/v2"
)

// NewMiddleware creates a new CSRF middleware handler with the provided options.
// It validates the CSRF token for incoming requests and generates a new token if needed.
// By default, this middleware generates a 419 HTTP response if CSRF validation fails.
//
// This middleware must be called after the session middleware.
func NewMiddleware(options ...Option) fiber.Handler {
	// Generate option
	option := &option{
		header: false,
		key:    "csrf_token",
		fail:   nil,
		next:   nil,
	}
	for _, opt := range options {
		opt(option)
	}

	return func(c *fiber.Ctx) error {
		// Skip
		if option.next != nil && option.next(c) {
			return c.Next()
		}

		// Parse and generate token
		session := session.Parse(c)
		if session == nil {
			return errors.New("failed to resolve session")
		}

		token := session.Cast("csrf").StringSafe("")
		if token == "" { // Generate or refresh token if needed
			token = refresh(session)
		}

		// Proccess request
		if option.header {
			option.key = strings.ToUpper(option.key)
			c.Append("Access-Control-Allow-Headers", option.key)
			if isRFC9110Method(c) {
				input := c.Get(option.key)
				if token == "" || input != token {
					if option.fail != nil {
						return option.fail(c)
					}
					return c.Status(419).SendString("invalid csrf token")
				}
			}
		} else {
			if isRFC9110Method(c) {
				input := getBodyValue(c, option.key)
				if token == "" || input != token {
					if option.fail != nil {
						return option.fail(c)
					}
					return c.Status(419).SendString("invalid csrf token")
				}
			}
		}

		return c.Next()
	}
}
