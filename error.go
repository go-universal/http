package http

import (
	"fmt"
	"mime/multipart"
	"runtime"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofiber/fiber/v2"
	"github.com/inhies/go-bytesize"
)

// HttpError represents an HTTP error with additional context.
type HttpError struct {
	Line    int            // Line number where the error occurred.
	File    string         // File name where the error occurred.
	Body    map[string]any // Request body data (if available).
	Status  int            // HTTP status code.
	Message string         // Error message.
}

// Error returns the error message as a string.
func (he HttpError) Error() string {
	return he.Message
}

// NewError creates an HttpError with a message and optional status code.
// Defaults to status 500 if none is provided.
func NewError(e string, status ...int) error {
	file, line, _ := realCaller()
	return HttpError{
		Line:    line,
		File:    file,
		Body:    nil,
		Status:  realStatus(status...),
		Message: e,
	}
}

// NewFormError creates an HttpError with a message, request context, and optional status code.
// Includes request body data if available.
func NewFormError(e string, ctx *fiber.Ctx, status ...int) error {
	file, line, _ := realCaller()
	return HttpError{
		Line:    line,
		File:    file,
		Body:    extractRequestBody(ctx),
		Status:  realStatus(status...),
		Message: e,
	}
}

// extractRequestBody extracts request body data from the Fiber context.
// Handles both form data and JSON body parsing.
func extractRequestBody(ctx *fiber.Ctx) map[string]any {
	if ctx == nil {
		return nil
	}

	body := make(map[string]any)
	if form, err := ctx.MultipartForm(); err == nil && form != nil {
		// Extract form values
		for k, v := range form.Value {
			if len(v) == 1 {
				body["form."+k] = v[0]
			} else if len(v) > 1 {
				body["form."+k] = v
			} else {
				body["form."+k] = nil
			}
		}

		// Extract uploaded files
		for k, files := range form.File {
			values := make([]string, 0, len(files))
			for _, file := range files {
				size := bytesize.New(float64(file.Size))
				mime := detectMime(file)
				values = append(values, fmt.Sprintf("%s [%s] (%s)", file.Filename, size, mime))
			}

			if len(values) == 0 {
				body["file."+k] = nil
			} else {
				body["file."+k] = values
			}
		}
	} else {
		// Parse JSON body
		var form map[string]any
		if err := ctx.BodyParser(&form); err != nil {
			body["form"] = err.Error()
		} else if len(form) == 0 {
			body["form"] = nil
		} else {
			for k, v := range form {
				body["form."+k] = v
			}
		}
	}

	return body
}

// detectMime determines the MIME type of a file.
// Returns "?" if the MIME type cannot be determined.
func detectMime(file *multipart.FileHeader) string {
	f, err := file.Open()
	if err != nil {
		return "?"
	}
	defer f.Close()

	if mime, err := mimetype.DetectReader(f); err == nil && mime != nil {
		return mime.String()
	}

	return "?"
}

// realCaller retrieves the file name and line number of the caller.
func realCaller() (string, int, bool) {
	if _, f, l, ok := runtime.Caller(2); ok {
		return f, l, true
	}
	return "", 0, false
}

// realStatus validates and returns an HTTP status code.
// Defaults to 500 if the provided status is invalid.
func realStatus(statuses ...int) int {
	if len(statuses) > 0 && statuses[0] > 399 && statuses[0] < 600 {
		return statuses[0]
	}

	return 500
}
