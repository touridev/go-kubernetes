package secrets

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	api_v1 "k8s.io/api/core/v1"
)

// JWTKeyKey is the key of the data field of a Secret where the JWK must be stored.
const JWTKeyKey = "jwk"

// CAKey is the key of the data field of a Secret where the certificate authority must be stored.
const CAKey = "ca.crt"

// SecretTypeCA contains a certificate authority for TLS certificate verification.
const SecretTypeCA api_v1.SecretType = "nginx.org/ca"

// SecretTypeJWK contains a JWK (JSON Web Key) for validating JWTs (JSON Web Tokens).
const SecretTypeJWK api_v1.SecretType = "nginx.org/jwk"

// ValidateTLSSecret validates the secret. If it is valid, the function returns nil.
func ValidateTLSSecret(secret *api_v1.Secret) error {
	if secret.Type != api_v1.SecretTypeTLS {
		return fmt.Errorf("TLS Secret must be of the type %v", api_v1.SecretTypeTLS)
	}

	// Kubernetes ensures that 'tls.crt' and 'tls.key' are present for secrets of api_v1.SecretTypeTLS type

	_, err := tls.X509KeyPair(secret.Data[api_v1.TLSCertKey], secret.Data[api_v1.TLSPrivateKeyKey])
	if err != nil {
		return fmt.Errorf("Failed to validate TLS cert and key: %v", err)
	}

	return nil
}

// ValidateJWKSecret validates the secret. If it is valid, the function returns nil.
func ValidateJWKSecret(secret *api_v1.Secret) error {
	if secret.Type != SecretTypeJWK {
		return fmt.Errorf("JWK secret must be of the type %v", SecretTypeJWK)
	}

	if _, exists := secret.Data[JWTKeyKey]; !exists {
		return fmt.Errorf("JWK secret must have the data field %v", JWTKeyKey)
	}

	// we don't validate the contents of secret.Data[JWTKeyKey], because invalid contents will not make NGINX Plus
	// fail to reload: NGINX Plus will return 500 responses for the affected URLs.

	return nil
}

// ValidateCASecret validates the secret. If it is valid, the function returns nil.
func ValidateCASecret(secret *api_v1.Secret) error {
	if secret.Type != SecretTypeCA {
		return fmt.Errorf("CA secret must be of the type %v", SecretTypeCA)
	}

	if _, exists := secret.Data[CAKey]; !exists {
		return fmt.Errorf("CA secret must have the data field %v", CAKey)
	}

	block, _ := pem.Decode(secret.Data[CAKey])
	if block == nil {
		return fmt.Errorf("The data field %s must hold a valid CERTIFICATE PEM block", CAKey)
	}
	if block.Type != "CERTIFICATE" {
		return fmt.Errorf("The data field %s must hold a valid CERTIFICATE PEM block, but got '%s'", CAKey, block.Type)
	}

	_, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("Failed to validate certificate: %v", err)
	}

	return nil
}

// IsSupportedSecretType checks if the secret type is supported.
func IsSupportedSecretType(secretType api_v1.SecretType) bool {
	return secretType == api_v1.SecretTypeTLS || secretType == SecretTypeCA || secretType == SecretTypeJWK
}

// ValidateSecret validates the secret. If it is valid, the function returns nil.
func ValidateSecret(secret *api_v1.Secret) error {
	switch secret.Type {
	case api_v1.SecretTypeTLS:
		return ValidateTLSSecret(secret)
	case SecretTypeJWK:
		return ValidateJWKSecret(secret)
	case SecretTypeCA:
		return ValidateCASecret(secret)
	}

	return fmt.Errorf("Secret is of the unsupported type %v", secret.Type)
}
