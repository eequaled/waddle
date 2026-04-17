//go:build !windows

package vault

// stubVault returns ErrKeyNotFound for all operations on non-Windows platforms.
type stubVault struct{}

// New returns a stub vault on non-Windows platforms.
func New(storePath string) Vault {
	return &stubVault{}
}

func (s *stubVault) Save(key string, data []byte) error  { return ErrKeyNotFound }
func (s *stubVault) Load(key string) ([]byte, error)     { return nil, ErrKeyNotFound }
func (s *stubVault) Delete(key string) error              { return nil }
