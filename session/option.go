package session

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// option represents configuration options for a session.
type option struct {
	ttl       time.Duration // ttl specifies the time-to-live duration for the session.
	name      string        // name is the name of the session.
	header    bool          // header indicates whether the session should be stored in the header.
	readOnly  bool          // not generate session if not exists
	cookie    *fiber.Cookie // cookie represents the session cookie settings.
	generator IdGenerator   // generator is the function used to generate session IDs.
}

// Option is a function type that modifies an Option.
type Option func(*option)

// WithTTL returns an Options function that sets the TTL of an Option.
func WithTTL(ttl time.Duration) Option {
	return func(o *option) {
		if ttl > 0 {
			o.ttl = ttl
		}
	}
}

// WithHeader sets the header name for the Option if the provided name is not empty.
// to indicate that a header is being used. It also clears any existing cookie settings.
func WithHeader(name string) Option {
	return func(o *option) {
		name := strings.TrimSpace(name)
		if name != "" {
			o.name = name
			o.header = true
			o.cookie = nil
		}
	}
}

// WithCookie sets the cookie name for the Option if the provided name is not empty.
// to indicate that a cookie is being used. It also clears any existing header settings.
func WithCookie(name string, cookie fiber.Cookie) Option {
	return func(o *option) {
		name := strings.TrimSpace(name)
		if name != "" {
			o.name = name
			o.cookie = &cookie
			o.header = false
		}
	}
}

// WithReadonly returns an Option that sets the session to read-only mode.
// When enabled, a session will not be generated if it does not already exist.
func WithReadonly() Option {
	return func(o *option) {
		o.readOnly = true
	}
}

// WithGenerator returns an Options function that sets the Generator of an Option.
func WithGenerator(generator IdGenerator) Option {
	return func(o *option) {
		if generator != nil {
			o.generator = generator
		}
	}
}
