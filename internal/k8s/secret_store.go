package k8s

import (
	"fmt"

	api_v1 "k8s.io/api/core/v1"
)

// StoredSecret holds a secret, its validation status and the path on the file system.
type StoredSecret struct {
	Secret        *api_v1.Secret
	Path          string
	ValidationErr error
}

// SecretFileManager manages secrets on the file system.
type SecretFileManager interface {
	AddOrUpdateSecret(secret *api_v1.Secret) string
	DeleteSecret(key string)
}

// SecretStore stores secrets that the Ingress Controller uses.
type SecretStore interface {
	AddOrUpdateSecret(secret *api_v1.Secret)
	DeleteSecret(key string)
	GetSecret(key string) (api_v1.SecretType, string, error)
}

// LocalSecretStore implements SecretStore interface.
// It validates the secrets and manages them on the file system (via SecretFileManager).
type LocalSecretStore struct {
	secrets map[string]*StoredSecret
	manager SecretFileManager
}

// NewLocalSecretStore creates a new LocalSecretStore.
func NewLocalSecretStore(manager SecretFileManager) *LocalSecretStore {
	return &LocalSecretStore{
		secrets: make(map[string]*StoredSecret),
		manager: manager,
	}
}

// AddOrUpdateSecret adds or updates a secret.
// The secret will only be updated on the file system if it is valid and if it is already on the file system.
// If the secret becomes invalid, it will be removed from the filesystem.
func (s *LocalSecretStore) AddOrUpdateSecret(secret *api_v1.Secret) {
	storedSecret, exists := s.secrets[getResourceKey(&secret.ObjectMeta)]
	if !exists {
		storedSecret = &StoredSecret{
			Secret: secret,
		}
	} else {
		storedSecret.Secret = secret
	}

	storedSecret.ValidationErr = ValidateSecret(secret)

	if storedSecret.Path != "" {
		if storedSecret.ValidationErr != nil {
			s.manager.DeleteSecret(getResourceKey(&secret.ObjectMeta))
			storedSecret.Path = ""
		} else {
			storedSecret.Path = s.manager.AddOrUpdateSecret(secret)
		}
	}

	s.secrets[getResourceKey(&secret.ObjectMeta)] = storedSecret
}

// DeleteSecret deletes a secret.
func (s *LocalSecretStore) DeleteSecret(key string) {
	storedSecret, exists := s.secrets[key]
	if !exists {
		return
	}

	delete(s.secrets, key)

	if storedSecret.Path == "" {
		return
	}

	s.manager.DeleteSecret(key)
}

// GetSecret gets the secretType and the path of a requested secret via its namespace/name key.
// If the secret doesn't exist, is of an unsupported type, or invalid, GetSecret will return an error.
// If the secret is valid but isn't present on the file system, the secret will be written to the file system.
func (s *LocalSecretStore) GetSecret(key string) (api_v1.SecretType, string, error) {
	storedSecret, exists := s.secrets[key]
	if !exists {
		return "", "", fmt.Errorf("secret doesn't exist or of an unsupported type")
	}

	if storedSecret.ValidationErr == nil && storedSecret.Path == "" {
		storedSecret.Path = s.manager.AddOrUpdateSecret(storedSecret.Secret)
	}

	return storedSecret.Secret.Type, storedSecret.Path, storedSecret.ValidationErr
}
