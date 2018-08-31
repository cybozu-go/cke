package cke

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/cybozu-go/log"
	vault "github.com/hashicorp/vault/api"
)

type anyMap = map[string]interface{}

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

// connectVault creates vault client
func connectVault(ctx context.Context, data []byte) error {
	c := new(VaultConfig)
	err := json.Unmarshal(data, c)
	if err != nil {
		return err
	}

	transport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConnsPerHost:   -1,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if len(c.CACert) > 0 {
		cp := x509.NewCertPool()
		if !cp.AppendCertsFromPEM([]byte(c.CACert)) {
			return errors.New("invalid CA cert")
		}

		transport.TLSClientConfig = &tls.Config{
			RootCAs:    cp,
			MinVersion: tls.VersionTLS12,
		}
	}

	client, err := vault.NewClient(&vault.Config{
		Address: c.Endpoint,
		HttpClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		log.Error("failed to connect to vault", anyMap{
			log.FnError: err,
			"endpoint":  c.Endpoint,
		})
		return err
	}

	secret, err := client.Logical().Write("auth/approle/login", anyMap{
		"role_id":   c.RoleID,
		"secret_id": c.SecretID,
	})
	if err != nil {
		log.Error("failed to login to vault", anyMap{
			log.FnError: err,
			"endpoint":  c.Endpoint,
		})
		return err
	}
	client.SetToken(secret.Auth.ClientToken)

	renewer, err := client.NewRenewer(&vault.RenewerInput{
		Secret: secret,
	})
	if err != nil {
		log.Error("failed to create vault renewer", anyMap{
			log.FnError: err,
			"endpoint":  c.Endpoint,
		})
		return err
	}

	go renewer.Renew()
	go func() {
		<-ctx.Done()
		renewer.Stop()
	}()

	setVaultClient(client)
	log.Info("connected to vault", anyMap{
		"endpoint": c.Endpoint,
	})
	return nil
}
