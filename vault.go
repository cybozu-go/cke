package cke

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/url"
)

// VaultConfig is data to store in etcd
type VaultConfig struct {
	// Endpoint is the address of the Vault server.
	Endpoint string `json:"endpoint"`

	// CACert is x509 certificate in PEM format of the endpoint CA.
	CACert string `json:"ca-cert"`

	// RoleID is AppRole ID to login to Vault.
	RoleID string `json:"role-id"`

	// SecretID is AppRole secret to login to Vault.
	SecretID string `json:"secret-id"`
}

// Validate validates the vault configuration
func (c *VaultConfig) Validate() error {
	if len(c.Endpoint) == 0 {
		return errors.New("endpoint is empty")
	}
	_, err := url.Parse(c.Endpoint)
	if err != nil {
		return err
	}
	if len(c.CACert) > 0 {
		block, _ := pem.Decode([]byte(c.CACert))
		if block == nil {
			return errors.New("invalid PEM data")
		}
		_, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			return errors.New("invalid certificate")
		}
	}
	if len(c.RoleID) == 0 {
		return errors.New("role-id is empty")
	}
	if len(c.SecretID) == 0 {
		return errors.New("secret-id is empty")
	}
	return nil
}
