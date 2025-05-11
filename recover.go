package http

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/go-universal/logger"
	"github.com/gofiber/fiber/v2"
)

// ErrorCallback is a function type that handles custom error responses.
type ErrorCallback func(ctx *fiber.Ctx, err HttpError) error

// NewFiberErrorHandler creates a new Fiber error handler with logging and custom error response capabilities.
// It takes a logger, an optional error callback, and a list of status codes to log.
// If the error matches one of the provided status codes, it will be logged using the provided logger.
// If an error callback is provided, it will be used to handle the error response; otherwise, a default plain text response will be sent.
// For relative file name in log use os.Setenv("APP_ROOT", "your/project/root") to define your project root.
func NewFiberErrorHandler(l logger.Logger, cb ErrorCallback, codes ...int) fiber.ErrorHandler {
	// Helper function to get the relative path of a file
	relative := func(path string) string {
		root := filepath.ToSlash(os.Getenv("APP_ROOT"))
		path = filepath.ToSlash(path)
		if root != "" {
			if p, err := filepath.Rel(root, path); err == nil {
				return p
			}
		}

		return path
	}

	return func(ctx *fiber.Ctx, err error) error {
		// Initialize error details
		var (
			file    string
			line    int
			body    map[string]any
			status  = fiber.StatusInternalServerError
			message = "Internal Server Error"
		)

		if fe, ok := err.(*fiber.Error); ok { // Parse Fiber error
			status = fe.Code
			message = fe.Error()
		} else if he, ok := err.(HttpError); ok { // Parse custom HttpError
			file = he.File
			line = he.Line
			message = he.Error()
			status = he.Status
			body = he.Body
		} else { // Parse regular errors
			message = err.Error()
		}

		// Log the error if logger is provided and status matches the specified codes
		if l != nil && (len(codes) == 0 || slices.Contains(codes, status)) {
			params := []logger.LogOptions{
				logger.With("file", relative(file)),
				logger.With("line", line),
				logger.With("status", status),
				logger.With("ip", ctx.IP()),
				logger.With("path", ctx.Path()),
				logger.With("method", ctx.Method()),
				logger.WithMessage(message),
			}
			for k, v := range body {
				params = append(params, logger.With(k, v))
			}
			l.Error(params...)
		}

		// Return error response
		if cb != nil {
			return cb(ctx, HttpError{
				Line:    line,
				File:    file,
				Body:    body,
				Status:  status,
				Message: message,
			})
		}

		// Default plain text response
		ctx.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		return ctx.Status(status).SendString(message)
	}
}
