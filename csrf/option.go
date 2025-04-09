package csrf

import "github.com/gofiber/fiber/v2"

// option holds the configuration options for CSRF middleware.
type option struct {
	header bool
	key    string
	fail   fiber.Handler
	next   func(*fiber.Ctx) bool
}

// Option defines a function type for configuring CSRF Option.
type Option func(*option)

// WithFail sets a custom failure handler for CSRF validation.
func WithFail(handler fiber.Handler) Option {
	return func(o *option) {
		o.fail = handler
	}
}

// WithNext sets a custom function can be used to skip CSRF validation for certain requests.
func WithNext(handler func(*fiber.Ctx) bool) Option {
	return func(o *option) {
		o.next = handler
	}
}

// WithHeader configures the CSRF middleware to check CSRF token from header.
func WithHeader(name string) Option {
	return func(o *option) {
		if name != "" {
			o.header = true
			o.key = name
		}
	}
}

// WithForm configures the CSRF middleware to check CSRF token from form field.
func WithForm(name string) Option {
	return func(o *option) {
		if name != "" {
			o.header = false
			o.key = name
		}
	}
}
