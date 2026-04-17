package vault

import "errors"

// ErrKeyNotFound is returned when a vault key does not exist.
var ErrKeyNotFound = errors.New("vault: key not found")

// Vault provides platform-specific secure storage.
// Windows: DPAPI-protected files.  Linux: libsecret (future).
type Vault interface {
	Save(key string, data []byte) error
	Load(key string) ([]byte, error)
	Delete(key string) error
}
