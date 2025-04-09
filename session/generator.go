package session

import "github.com/google/uuid"

// IdGenerator is a function type that generates a new session ID as a string.
type IdGenerator func() string

// UUIDGenerator generates a new UUID string using the google/uuid package.
func UUIDGenerator() string {
	return uuid.NewString()
}
