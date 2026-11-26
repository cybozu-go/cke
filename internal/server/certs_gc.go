package server

import (
	"context"
	"path"
	"time"

	"github.com/cybozu-go/log"
	vault "github.com/hashicorp/vault/api"

	"github.com/cybozu-go/cke"
)

// certsGCLoop runs the certificate GC every CertsGCInterval.
// It returns nil when ctx is canceled.
func (c Controller) certsGCLoop(ctx context.Context) error {
	ticker := time.NewTicker(c.config.CertsGCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		c.gcCerts(ctx)
	}
}

// gcCerts starts removing expired and revoked certificates from Vault.
// It does this for all CAs.
// It does not return errors because the GC is not mandatory.
// Failures are logged as warnings.
func (c Controller) gcCerts(ctx context.Context) {
	storage := cke.Storage{
		Client: c.session.Client(),
	}

	cfg, err := storage.GetVaultConfig(ctx)
	if err != nil {
		log.Warn("failed to get vault config; skip certificate GC", map[string]any{
			log.FnError: err,
		})
		return
	}

	client, _, err := cke.VaultClient(cfg)
	if err != nil {
		log.Warn("failed to connect to vault; skip certificate GC", map[string]any{
			log.FnError: err,
		})
		return
	}

	for _, ca := range cke.CAKeys {
		if err := tidyExpiredCertificates(ctx, client, cke.VaultPKIKey(ca)); err != nil {
			log.Warn("failed to start certificate GC", map[string]any{
				log.FnError: err,
				"ca":        ca,
			})
			continue
		}
		log.Info("started certificate GC", map[string]any{
			"ca": ca,
		})
	}
}

// tidyExpiredCertificates starts the tidy operation of the PKI engine at pkiKey.
// The operation runs in Vault asynchronously.
// It removes expired and revoked certificates.
//
// Vault returns an error if it cannot start the operation.
// If another tidy operation is running, Vault returns only a warning.
// This function returns nil in that case.
// The running operation removes the certificates instead.
func tidyExpiredCertificates(ctx context.Context, client *vault.Client, pkiKey string) error {
	_, err := client.Logical().WriteWithContext(ctx, path.Join(pkiKey, "tidy"), map[string]any{
		"tidy_cert_store":    true,
		"tidy_revoked_certs": true,
		"safety_buffer":      time.Minute.String(),
	})
	return err
}
