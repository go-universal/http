package uploader

import (
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-universal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/inhies/go-bytesize"
	"github.com/valyala/fasthttp"
)

// Uploader is an interface that defines methods for handling file uploads.
type Uploader interface {
	// IsNil checks if the uploader is nil.
	IsNil() bool

	// ValidateSize checks if the file size is within the specified limit.
	// Use B, KB, MB, GB for size string
	ValidateSize(min, max string) (bool, error)

	// ValidateMime checks if the file MIME type is among the allowed types.
	ValidateMime(mimes ...string) (bool, error)

	// Path returns the file path where the uploaded file is stored.
	Path() string

	// URL returns the URL where the uploaded file can be accessed.
	URL() string

	// Save stores the uploaded file.
	Save() error

	// Delete removes the uploaded file.
	Delete() error

	// SafeDelete removes the uploaded file safely, queueing the file name on failure.
	SafeDelete()
}

type uploader struct {
	opt   option
	file  *multipart.FileHeader
	name  string
	root  string
	saved bool
}

// NewUploader creates a new Uploader instance with the given root directory and file header.
func NewUploader(root string, file *multipart.FileHeader, options ...Option) (Uploader, error) {
	// Initialize and normalize
	var name string
	root = strings.TrimSpace(root)

	// Create option with default values.
	option := &option{
		queue:    nil,
		numbered: false,
		prefix:   "",
	}
	for _, opt := range options {
		opt(option)
	}

	// Generate file name
	if file != nil {
		if option.numbered {
			n, err := utils.NumberedFile(root, file.Filename)
			if err != nil {
				return nil, err
			}
			name = n
		} else {
			name = utils.TimestampedFile(file.Filename)
		}
	}

	// Create and return the uploader instance.
	u := &uploader{
		opt:  *option,
		file: file,
		name: name,
		root: root,
	}
	return u, nil
}

// NewFiberUploader creates a new Uploader instance for a Fiber context.
func NewFiberUploader(root string, c *fiber.Ctx, name string, options ...Option) (Uploader, error) {
	file, err := c.FormFile(name)
	if err == fasthttp.ErrMissingFile {
		return NewUploader(root, nil, options...)
	}

	if err != nil {
		return nil, err
	}

	return NewUploader(root, file, options...)
}

// FiberFile retrieves a file from a Fiber context by its form field name.
// If the file is not found, it returns nil without an error.
// If another error occurs, it returns the error.
func FiberFile(c *fiber.Ctx, name string) (*multipart.FileHeader, error) {
	f, err := c.FormFile(name)
	if err == fasthttp.ErrMissingFile {
		return nil, nil
	}
	return f, err
}

func (u *uploader) IsNil() bool {
	return u.file == nil
}

func (u *uploader) ValidateSize(min, max string) (bool, error) {
	// Invalidate nil file
	if u.IsNil() {
		return false, nil
	}

	// Parse min string
	minSize, err := bytesize.Parse(min)
	if err != nil {
		return false, err
	}

	// Parse max string
	maxSize, err := bytesize.Parse(max)
	if err != nil {
		return false, err
	}

	// Validate
	size := u.file.Size
	return size >= int64(minSize) && size <= int64(maxSize), nil
}

func (u *uploader) ValidateMime(mimes ...string) (bool, error) {
	// Invalidate nil file
	if u.IsNil() {
		return false, nil
	}

	// Read file content
	f, err := u.file.Open()
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Validate mime
	mime, err := mimetype.DetectReader(f)
	if err != nil {
		return false, err
	}

	return mimetype.EqualsAny(mime.String(), mimes...), nil
}

func (u *uploader) Path() string {
	// Skip nil file
	if u.IsNil() {
		return ""
	}

	return utils.NormalizePath(u.root, u.name)
}

func (u *uploader) URL() string {
	// Skip nil file
	if u.IsNil() {
		return ""
	}

	return utils.AbsoluteURL(u.opt.prefix, u.Path())
}

func (u *uploader) Save() error {
	// Skip nil file or saved
	if u.IsNil() || u.saved {
		return nil
	}

	dest := u.Path()

	// Check if exists
	exists, err := utils.FileExists(dest)
	if err != nil {
		return err
	} else if exists {
		return fmt.Errorf("%s file exists", dest)
	}

	// Save
	err = fasthttp.SaveMultipartFile(u.file, dest)
	if err != nil {
		return err
	}

	u.saved = true
	return nil
}

func (u *uploader) Delete() error {
	// Skip nil file or not saved
	if u.IsNil() || !u.saved {
		return nil
	}

	// Delete
	err := os.Remove(u.Path())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}

func (u *uploader) SafeDelete() {
	err := u.Delete()
	if u.opt.queue == nil {
		return
	}

	if err != nil {
		u.opt.queue.Push(u.Path())
	}
}
