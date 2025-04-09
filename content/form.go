package content

import "github.com/gofiber/fiber/v2"

// FormOnly is a middleware that ensures the request's Content-Type is either "multipart/form-data" or
// "application/x-www-form-urlencoded". If the Content-Type is neither of these, it will execute the
// optional onFail handler if provided, or return a 406 Not Acceptable status by default.
func FormOnly(onFail ...fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !isValidContent(c.Get(fiber.HeaderContentType), fiber.MIMEMultipartForm, fiber.MIMEApplicationForm) {
			if len(onFail) > 0 && onFail[0] != nil {
				return onFail[0](c)
			}
			return c.Status(fiber.StatusNotAcceptable).SendString("Not Acceptable")
		}
		return c.Next()
	}
}
