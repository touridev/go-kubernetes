package k8s

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeSecretFileManager struct {
	AddedOrUpdatedSecret *api_v1.Secret
	DeletedSecret        string
}

func (m *fakeSecretFileManager) AddOrUpdateSecret(secret *api_v1.Secret) string {
	m.AddedOrUpdatedSecret = secret
	return "testpath"
}

func (m *fakeSecretFileManager) DeleteSecret(key string) {
	m.DeletedSecret = key
}

func (m *fakeSecretFileManager) Reset() {
	m.AddedOrUpdatedSecret = nil
	m.DeletedSecret = ""
}

var (
	validSecret = &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "tls-secret",
			Namespace: "default",
		},
		Type: api_v1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": validCert,
			"tls.key": validKey,
		},
	}
	invalidSecret = &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "tls-secret",
			Namespace: "default",
		},
		Type: api_v1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": invalidCert,
			"tls.key": validKey,
		},
	}
)

func TestAddOrUpdateSecret(t *testing.T) {
	manager := &fakeSecretFileManager{}

	store := NewLocalSecretStore(manager)

	// Add the valid secret

	expectedManager := &fakeSecretFileManager{}

	store.AddOrUpdateSecret(validSecret)

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("AddOrUpdateSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Get the secret

	expectedSecretType := api_v1.SecretTypeTLS
	expectedSecretPath := "testpath"
	expectedManager = &fakeSecretFileManager{
		AddedOrUpdatedSecret: validSecret,
	}

	manager.Reset()
	secretType, secretPath, err := store.GetSecret("default/tls-secret")

	if secretType != expectedSecretType {
		t.Errorf("GetSecret() returned %v but expected %v", secretType, expectedSecretType)
	}
	if secretPath != expectedSecretPath {
		t.Errorf("GetSecret() returned %v but expected %v", secretPath, expectedSecretPath)
	}
	if err != nil {
		t.Errorf("GetSecret() returned unexpected error %v", err)
	}
	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Make the secret invalid

	expectedManager = &fakeSecretFileManager{
		DeletedSecret: "default/tls-secret",
	}

	manager.Reset()
	store.AddOrUpdateSecret(invalidSecret)

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("AddOrUpdateSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Get the secret

	expectedSecretType = api_v1.SecretTypeTLS
	expectedSecretPath = ""
	expectedManager = &fakeSecretFileManager{}
	expectedErrorMsg := "Failed to validate TLS cert and key: asn1: syntax error: sequence truncated"

	manager.Reset()
	secretType, secretPath, err = store.GetSecret("default/tls-secret")

	if secretType != expectedSecretType {
		t.Errorf("GetSecret() returned %v but expected %v", secretType, expectedSecretType)
	}
	if secretPath != expectedSecretPath {
		t.Errorf("GetSecret() returned %v but expected %v", secretPath, expectedSecretPath)
	}
	if err == nil || err.Error() != expectedErrorMsg {
		t.Errorf("GetSecret() returned error %v but expected %s", err, expectedErrorMsg)
	}
	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Restore the valid secret

	expectedManager = &fakeSecretFileManager{}

	manager.Reset()
	store.AddOrUpdateSecret(validSecret)

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("AddOrUpdateSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Get the secret

	expectedSecretType = api_v1.SecretTypeTLS
	expectedSecretPath = "testpath"
	expectedManager = &fakeSecretFileManager{
		AddedOrUpdatedSecret: validSecret,
	}

	manager.Reset()
	secretType, secretPath, err = store.GetSecret("default/tls-secret")

	if secretType != expectedSecretType {
		t.Errorf("GetSecret() returned %v but expected %v", secretType, expectedSecretType)
	}
	if secretPath != expectedSecretPath {
		t.Errorf("GetSecret() returned %v but expected %v", secretPath, expectedSecretPath)
	}
	if err != nil {
		t.Errorf("GetSecret() returned unexpected error %v", err)
	}
	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Update the secret

	expectedManager = &fakeSecretFileManager{
		AddedOrUpdatedSecret: validSecret,
	}

	manager.Reset()
	// for the test, it is ok to use the same version
	store.AddOrUpdateSecret(validSecret)

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("AddOrUpdateSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Get the secret

	expectedSecretType = api_v1.SecretTypeTLS
	expectedSecretPath = "testpath"
	expectedManager = &fakeSecretFileManager{}

	manager.Reset()
	secretType, secretPath, err = store.GetSecret("default/tls-secret")

	if secretType != expectedSecretType {
		t.Errorf("GetSecret() returned %v but expected %v", secretType, expectedSecretType)
	}
	if secretPath != expectedSecretPath {
		t.Errorf("GetSecret() returned %v but expected %v", secretPath, expectedSecretPath)
	}
	if err != nil {
		t.Errorf("GetSecret() returned unexpected error %v", err)
	}
	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestDeleteSecretNonExisting(t *testing.T) {
	manager := &fakeSecretFileManager{}
	store := NewLocalSecretStore(manager)

	expectedManager := &fakeSecretFileManager{}

	store.DeleteSecret("default/tls-secret")

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("DeleteSecret() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestDeleteSecretValidSecret(t *testing.T) {
	manager := &fakeSecretFileManager{}
	store := NewLocalSecretStore(manager)

	// Add the valid secret

	expectedManager := &fakeSecretFileManager{}

	store.AddOrUpdateSecret(validSecret)

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("AddOrUpdateSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Get the secret

	expectedSecretType := api_v1.SecretTypeTLS
	expectedSecretPath := "testpath"
	expectedManager = &fakeSecretFileManager{
		AddedOrUpdatedSecret: validSecret,
	}

	manager.Reset()
	secretType, secretPath, err := store.GetSecret("default/tls-secret")

	if secretType != expectedSecretType {
		t.Errorf("GetSecret() returned %v but expected %v", secretType, expectedSecretType)
	}
	if secretPath != expectedSecretPath {
		t.Errorf("GetSecret() returned %v but expected %v", secretPath, expectedSecretPath)
	}
	if err != nil {
		t.Errorf("GetSecret() returned unexpected error %v", err)
	}
	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Delete the secret

	expectedManager = &fakeSecretFileManager{
		DeletedSecret: "default/tls-secret",
	}

	manager.Reset()
	store.DeleteSecret("default/tls-secret")

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("DeleteSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Get the secret

	expectedSecretType = ""
	expectedSecretPath = ""
	expectedManager = &fakeSecretFileManager{}
	expectedErrorMsg := "secret doesn't exist or of an unsupported type"

	manager.Reset()
	secretType, secretPath, err = store.GetSecret("default/tls-secret")

	if secretType != expectedSecretType {
		t.Errorf("GetSecret() returned %v but expected %v", secretType, expectedSecretType)
	}
	if secretPath != expectedSecretPath {
		t.Errorf("GetSecret() returned %v but expected %v", secretPath, expectedSecretPath)
	}
	if err == nil || err.Error() != expectedErrorMsg {
		t.Errorf("GetSecret() returned error %v but expected %s", err, expectedErrorMsg)
	}
	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestDeleteSecretInvalidSecret(t *testing.T) {
	manager := &fakeSecretFileManager{}
	store := NewLocalSecretStore(manager)

	// Add invalid secret

	expectedManager := &fakeSecretFileManager{}

	store.AddOrUpdateSecret(invalidSecret)

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("AddOrUpdateSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Delete invalid secret

	expectedManager = &fakeSecretFileManager{}

	manager.Reset()
	store.DeleteSecret("default/tls-secret")

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("DeleteSecret() returned unexpected result (-want +got):\n%s", diff)
	}
}

type fakeSecretStore struct {
	secrets map[string]*StoredSecret
}

func newFakeSecretsStore(secrets map[string]*StoredSecret) *fakeSecretStore {
	return &fakeSecretStore{
		secrets: secrets,
	}
}

func (s *fakeSecretStore) AddOrUpdateSecret(secret *api_v1.Secret) {
}

func (s *fakeSecretStore) DeleteSecret(key string) {
}

func (s *fakeSecretStore) GetSecret(key string) (api_v1.SecretType, string, error) {
	storedSecret, exists := s.secrets[key]
	if !exists {
		return "", "", fmt.Errorf("secret doesn't exist")
	}

	return storedSecret.Secret.Type, storedSecret.Path, storedSecret.ValidationErr
}
