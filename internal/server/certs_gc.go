package server

import (
	"context"
	"errors"
	"path"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	vault "github.com/hashicorp/vault/api"

	"github.com/cybozu-go/cke"
)

// certsGCLoop runs the certificate GC every CertsGCInterval.
// It stops when ctx is canceled.
func (c Controller) certsGCLoop(ctx context.Context) error {
	ticker := time.NewTicker(c.config.CertsGCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		if err := c.gcCerts(ctx); err != nil {
			return err
		}
	}
}

// gcCerts starts removing expired and revoked certificates from Vault.
// It does this for all CAs.
func (c Controller) gcCerts(ctx context.Context) error {
	storage := cke.Storage{
		Client: c.session.Client(),
	}

	cfg, err := storage.GetVaultConfig(ctx)
	if err != nil {
		log.Warn("failed to get vault config. skip tidy", map[string]any{
			log.FnError: err,
		})
		// lint:ignore nilerr  Tidy is not mandatory.
		return nil
	}

	client, _, err := cke.VaultClient(cfg)
	if err != nil {
		return err
	}

	for _, ca := range cke.CAKeys {
		if err := c.TidyExpiredCertificates(ctx, client, cke.VaultPKIKey(ca)); err != nil {
			log.Warn("failed to tidy expired certificates", map[string]any{
				log.FnError: err,
				"ca":        ca,
			})
		}
	}

	return nil
}

// TidyExpiredCertificates call tidy endpoints of Vault API
func (c Controller) TidyExpiredCertificates(ctx context.Context, client *vault.Client, ca string) error {
	tidyParams := make(map[string]any)
	tidyParams["tidy_cert_store"] = true
	tidyParams["tidy_revocation_list"] = true
	tidyParams["safety_buffer"] = (time.Minute).String()
	res, err := client.Logical().Write(path.Join(ca, "tidy"), tidyParams)
	if err != nil {
		return err
	}

	// TODO: More appropriate error detection.
	// Vault client does not provide an interface to detect whether errors have occurred or not.
	if len(res.Warnings) == 0 {
		log.Warn("may be failed to tidy certs, since an empty message is returned", map[string]any{
			"tidy_params": tidyParams,
			"ca":          ca,
			"res":         res,
		})
		return errors.New("may be failed to tidy certs, since an empty message is returned")
	}
	// TODO: More appropriate error detection.
	// Currently, use this message. https://github.com/hashicorp/vault/blob/975db34faf38f5bf564d13da38e141975d9f0fe3/builtin/credential/approle/path_tidy_user_id.go#L240
	if res.Warnings[0] != "Tidy operation successfully started. Any information from the operation will be printed to Vault's server logs." {
		log.Warn("may be failed to tidy certs, since the expected warning message is not found", map[string]any{
			"tidy_params":         tidyParams,
			"ca":                  ca,
			"vault_resp_warnings": strings.Join(res.Warnings, ", "),
		})
		return errors.New("failed to tidy certs: " + strings.Join(res.Warnings, ", "))
	}

	log.Info("invoked vault tidy", map[string]any{
		"tidy_params":         tidyParams,
		"ca":                  ca,
		"vault_resp_warnings": strings.Join(res.Warnings, ", "),
	})

	return nil
}
