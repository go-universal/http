package session

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/go-universal/cache"
	"github.com/go-universal/cast"
	"github.com/gofiber/fiber/v2"
)

// Session represents a user session interface with methods to manage session data.
type Session interface {
	// Id returns the session identifier.
	Id() string

	// Context returns the associated Fiber context.
	Context() *fiber.Ctx

	// Set stores a value in the session for the given key.
	Set(key string, value any)

	// Get retrieves a value from the session for the given key.
	Get(key string) any

	// Delete removes a value from the session for the given key.
	Delete(key string)

	// Exists checks if a key exists in the session.
	Exists(key string) bool

	// Cast returns a Caster for the value associated with the given key.
	Cast(key string) cast.Caster

	// CreatedAt retrieves session creation date.
	CreatedAt() *time.Time

	// AddTTL extends the session's time-to-live.
	AddTTL(ttl time.Duration) error

	// SetTTL set session's time-to-live.
	SetTTL(ttl time.Duration) error

	// Destroy terminates the session.
	Destroy() error

	// Save persists the session data to storage if changed.
	// Must be called at the end of middleware.
	Save() error

	// Fresh generates a new session.
	Fresh() error

	// Load retrieves session data from storage.
	// Returns false if the session does not exist.
	Load() (bool, error)

	isHeader() bool
	isNoop() bool
	getName() string
}

// session represents a user session with associated data and metadata.
type session struct {
	id   string         // Unique identifier for the session.
	opt  option         // Configuration options for the session.
	data map[string]any // Key-value store for session data.

	ttl      time.Duration // Additional time-to-live for the session.
	fresh    bool          // Flag indicating if session is fresh.
	modified bool          // Flag indicating if session data has been modified.
	noop     bool          // Flag indicating if session should ignored on readonly mode when session not exists.

	ctx   *fiber.Ctx   // Fiber context associated with the session.
	cache cache.Cache  // Cache for storing session data.
	mutex sync.RWMutex // Mutex for synchronizing access to session data.
}

// New create or parse session driver.
func New(ctx *fiber.Ctx, cache cache.Cache, options ...Option) (Session, error) {
	// Generate option
	option := &option{
		ttl:       24 * time.Hour,
		name:      "session",
		header:    false,
		readOnly:  false,
		cookie:    &fiber.Cookie{},
		generator: UUIDGenerator,
	}
	for _, opt := range options {
		opt(option)
	}

	// Get session id
	var id string
	if option.header {
		id = ctx.Get(option.name)
	} else {
		id = ctx.Cookies(option.name)
	}

	// Generate session
	session := &session{
		id:   id,
		opt:  *option,
		data: make(map[string]any),

		ttl:      0,
		fresh:    false,
		modified: false,
		noop:     false,

		ctx:   ctx,
		cache: cache,
	}

	ok, err := session.Load()
	if err != nil {
		return nil, err
	}

	if !ok {
		// Readonly mode or create fresh
		if option.readOnly {
			session.noop = true
		} else if err := session.Fresh(); err != nil {
			return nil, err
		}
	}

	return session, nil
}

func (s *session) Id() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.id
}

func (s *session) Context() *fiber.Ctx {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.ctx
}

func (s *session) Set(k string, v any) {
	// Ignore not-exists readonly session
	if s.noop {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if k = strings.TrimSpace(k); k != "" {
		s.data[k] = v
		s.modified = true
	}
}

func (s *session) Get(k string) any {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.data[k]
}

func (s *session) Delete(k string) {
	// Ignore not-exists readonly session
	if s.noop {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.data, k)
	s.modified = true
}

func (s *session) Exists(k string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, ok := s.data[k]
	return ok
}

func (s *session) Cast(k string) cast.Caster {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return cast.NewCaster(s.data[k])
}

func (s *session) CreatedAt() *time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	raw, ok := s.data["created_at"].(string)
	if !ok {
		return nil
	}

	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}

	return &t
}

func (s *session) AddTTL(t time.Duration) error {
	// Skip empty ttl and not-exists readonly session
	if t <= 0 || s.noop {
		return nil
	}

	// Safe race condition
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Schedule update
	s.ttl = t
	s.modified = true
	return s.syncLocked()
}

func (s *session) SetTTL(t time.Duration) error {
	// Skip empty ttl and not-exists readonly session
	if t <= 0 || s.noop {
		return nil
	}

	// Safe race condition
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Schedule update
	s.ttl = -t
	s.modified = true
	return s.syncLocked()
}

func (s *session) Destroy() error {
	// Skip empty session and not-exists readonly session
	if s.id == "" || s.noop {
		return nil
	}

	// Safe race condition
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Delete from cache
	err := s.cache.Forget(s.k())
	if err != nil {
		return err
	}

	// Clear data
	s.id = ""
	s.data = make(map[string]any)
	s.ttl = 0
	s.fresh = false
	s.modified = false
	return nil
}

func (s *session) Save() error {
	// Skip un-initialized, unchanged, destroyed and not-exists readonly session
	if s.id == "" || (!s.fresh && !s.modified) || s.noop {
		return nil
	}

	// Safe race condition
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Encode data
	encoded, err := json.Marshal(s.data)
	if err != nil {
		return err
	}

	// Store New
	if s.fresh {
		if err := s.cache.Put(s.k(), encoded, &s.opt.ttl); err != nil {
			return err
		}
	} else if s.ttl != 0 {
		var ttl time.Duration

		if s.ttl > 0 {
			if current, err := s.cache.TTL(s.k()); err != nil {
				return err
			} else if current <= 0 {
				ttl = s.ttl
			} else {
				ttl = current + s.ttl
			}
		} else {
			ttl = -s.ttl
		}

		if err := s.cache.Put(s.k(), encoded, &ttl); err != nil {
			return err
		}
	} else {
		if _, err = s.cache.Update(s.k(), encoded); err != nil {
			return err
		}
	}

	s.ttl = 0
	s.fresh = false
	s.modified = false
	return nil
}

func (s *session) Fresh() error {
	// Ignore not-exists readonly session
	if s.noop {
		return nil
	}

	// Safe race condition
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Destroy old session
	if s.id != "" {
		err := s.cache.Forget(s.k())
		if err != nil {
			return err
		}
	}

	// Set identifier and created at
	s.id = s.opt.generator()
	s.ttl = s.opt.ttl
	s.data = make(map[string]any)
	s.fresh = true
	s.modified = true
	s.data["created_at"] = time.Now().Format(time.RFC3339)
	return s.syncLocked()
}

func (s *session) Load() (bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Not generated or empty id
	if s.id == "" {
		return false, nil
	}

	// Check if session exists
	exists, err := s.cache.Exists(s.k())
	if err != nil {
		return false, err
	} else if !exists {
		return false, nil
	}

	// Parse data and decode data
	caster, err := s.cache.Cast(s.k())
	if err != nil {
		return false, err
	}

	encoded, err := caster.String()
	if err != nil {
		return false, err
	}

	s.data = make(map[string]any)
	err = json.Unmarshal([]byte(encoded), &s.data)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *session) isHeader() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.opt.header
}

func (s *session) isNoop() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.noop
}

func (s *session) getName() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.opt.name
}

func (s *session) k() string {
	return "ses-" + s.id
}

func (s *session) syncLocked() error {
	// Ignore empty and not-exists readonly session
	if s.id == "" || s.noop {
		return nil
	}

	// Send header
	if s.opt.header {
		s.ctx.Set(s.opt.name, s.id)
		return nil
	}

	// Send cookie
	ttl := s.ttl
	if !s.fresh {
		if s.ttl < 0 {
			ttl = -s.ttl
		} else if s.ttl > 0 {
			if cacheTTL, err := s.cache.TTL(s.k()); err != nil {
				return err
			} else if cacheTTL > 0 {
				ttl += cacheTTL
			}
		}
	}

	s.ctx.Cookie(&fiber.Cookie{
		Name:        s.opt.name,
		Value:       s.id,
		Expires:     time.Now().Add(ttl),
		Secure:      s.opt.cookie.Secure,
		Domain:      s.opt.cookie.Domain,
		SameSite:    s.opt.cookie.SameSite,
		Path:        s.opt.cookie.Path,
		MaxAge:      s.opt.cookie.MaxAge,
		HTTPOnly:    s.opt.cookie.HTTPOnly,
		SessionOnly: s.opt.cookie.SessionOnly,
	})

	return nil
}
