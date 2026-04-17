//go:build windows

package platform

import "waddle/pkg/infra/vault"

// windowsSecretStore delegates to the vault package (DPAPI-backed on Windows).
type windowsSecretStore struct {
	v vault.Vault
}

func newWindowsSecretStore(dataDir string) *windowsSecretStore {
	return &windowsSecretStore{
		v: vault.New(dataDir),
	}
}

func (w *windowsSecretStore) Save(key string, data []byte) error {
	return w.v.Save(key, data)
}

func (w *windowsSecretStore) Load(key string) ([]byte, error) {
	return w.v.Load(key)
}

func (w *windowsSecretStore) Delete(key string) error {
	return w.v.Delete(key)
}
