package content

import (
	"github.com/gofiber/fiber/v2"
)

// MultipartOnly is a middleware that ensures the request's Content-Type is "multipart/form-data"
// If the Content-Type is neither of these, it will execute the optional onFail handler if provided,
// or return a 406 Not Acceptable status by default.
func MultipartOnly(onFail ...fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !isValidContent(c.Get(fiber.HeaderContentType), fiber.MIMEMultipartForm) {
			if len(onFail) > 0 && onFail[0] != nil {
				return onFail[0](c)
			}
			return c.Status(fiber.StatusNotAcceptable).SendString("Not Acceptable")
		}
		return c.Next()
	}
}
