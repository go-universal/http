package csrf

import (
	"errors"
	"slices"

	"github.com/go-universal/http/session"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetToken retrieves the token from the session associated with the given Fiber context.
// Returns an empty string if the session is nil or the "csrf" value is not found.
func GetToken(c *fiber.Ctx) string {
	s := session.Parse(c)
	if s == nil {
		return ""
	}
	return s.Cast("csrf").StringSafe("")
}

// RefreshToken generates a new CSRF token and saves it to the session.
// It returns the generated token or an error if the session cannot be resolved.
func RefreshToken(c *fiber.Ctx) (string, error) {
	// Parse session
	s := session.Parse(c)
	if s == nil {
		return "", errors.New("failed to resolve session")
	}

	// Save to session
	return refresh(s), nil
}

// refresh csrf on session
func refresh(s session.Session) string {
	token := uuid.NewString()
	s.Set("csrf", token)
	return token
}

// isRFC9110Method check if request method not GET, HEAD, OPTIONS and TRACE.
// RFC9110#section-9.2.1 safe methods.
func isRFC9110Method(c *fiber.Ctx) bool {
	return slices.Contains(
		[]string{
			fiber.MethodPost,
			fiber.MethodPut,
			fiber.MethodPatch,
			fiber.MethodDelete,
		},
		c.Method(),
	)
}

// getBodyValue get value from request body.
func getBodyValue(ctx *fiber.Ctx, key string) string {
	var body map[string]interface{}
	if err := ctx.BodyParser(&body); err != nil {
		return ""
	}

	value, _ := body[key].(string)
	return value
}
