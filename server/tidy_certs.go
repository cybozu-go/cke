package server

import (
	"context"
	"errors"
	"path"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	vault "github.com/hashicorp/vault/api"
)

func addRole(client *vault.Client, ca, role string, data map[string]interface{}) error {
	l := client.Logical()
	rpath := path.Join(ca, "roles", role)
	secret, err := l.Read(rpath)
	if err != nil {
		return err
	}
	if secret != nil {
		// already exists
		return nil
	}

	_, err = l.Write(rpath, data)
	if err != nil {
		log.Error("failed to create vault role", map[string]interface{}{
			log.FnError: err,
			"ca":        ca,
			"role":      role,
		})
	}
	return err
}

// TidyExpiredCertificates call tidy endpoints of Vault API
func (c Controller) TidyExpiredCertificates(ctx context.Context, inf cke.Infrastructure, ca, role string, roleOpts map[string]interface{}) error {
	client, err := inf.Vault()
	if err != nil {
		return err
	}

	err = addRole(client, ca, role, roleOpts)
	if err != nil {
		return err
	}

	tidyParams := make(map[string]interface{})
	tidyParams["tidy_cert_store"] = true
	tidyParams["tidy_revocation_list"] = true
	tidyParams["safety_buffer"] = (1 * time.Minute).String()
	res, err := client.Logical().Write(path.Join(ca, "tidy"), tidyParams)
	if err != nil {
		return err
	}

	if len(res.Warnings) == 0 {
		log.Warn("failed to tidy certs without any response message", nil)
		return errors.New("failed to tidy certs without any response message")
	}

	if res.Warnings[0] != "Tidy operation successfully started. Any information from the operation will be printed to Vault's server logs." {
		log.Warn("failed to tidy certs: ", map[string]interface{}{
			"message": strings.Join(res.Warnings, ", "),
		})
		return errors.New("failed to tidy certs: " + strings.Join(res.Warnings, ", "))
	}

	return nil
}
