package cke

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	vault "github.com/hashicorp/vault/api"
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

// ConnectVault creates vault client
func ConnectVault(c *VaultConfig) (*vault.Client, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if len(c.CACert) > 0 {
		cp := x509.NewCertPool()
		if !cp.AppendCertsFromPEM([]byte(c.CACert)) {
			return nil, errors.New("invalid CA cert")
		}

		transport.TLSClientConfig = &tls.Config{
			RootCAs:    cp,
			MinVersion: tls.VersionTLS12,
		}
	}

	return vault.NewClient(&vault.Config{
		Address: c.Endpoint,
		HttpClient: &http.Client{
			Transport: transport,
		},
	})
}
