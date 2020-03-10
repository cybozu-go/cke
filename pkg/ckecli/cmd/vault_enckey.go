package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cybozu-go/cke"
	vault "github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"
	apiserverv1 "k8s.io/apiserver/pkg/apis/config/v1"
)

var vaultEncKeyCmd = &cobra.Command{
	Use:   "enckey",
	Short: "generate new encryption key for Kubernetes Secrets",
	Long: `Generate or rotate encryption keys for Kubernetes Secrets.

WARNING: Key rotation is not fully implemented in this version!!

This command generates new encryption keys for Kubernetes Secrets and
rotate old keys.  The current key, if any, is retained to decrypt
existing data.  Other old keys are removed.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		vc, err := inf.Vault()
		if err != nil {
			return err
		}
		err = rotateK8sEncryptionKey(vc)
		if err != nil {
			return err
		}

		fmt.Println("succeeded")
		return nil
	},
}

func init() {
	vaultCmd.AddCommand(vaultEncKeyCmd)
}

func rotateK8sEncryptionKey(vc *vault.Client) error {
	secret, err := vc.Logical().Read(cke.K8sSecret)
	if err != nil {
		return err
	}

	var enckeys map[string]interface{}
	if secret != nil && secret.Data != nil {
		enckeys = secret.Data
	} else {
		enckeys = make(map[string]interface{})
	}

	var cfg apiserverv1.AESConfiguration
	if data, ok := enckeys["aescbc"]; ok {
		err = json.Unmarshal([]byte(data.(string)), &cfg)
		if err != nil {
			return err
		}
	}

	newKey, err := generateKey()
	if err != nil {
		return err
	}
	keys := []apiserverv1.Key{
		{
			Name:   time.Now().UTC().Format(time.RFC3339),
			Secret: base64.StdEncoding.EncodeToString(newKey),
		},
	}
	if len(cfg.Keys) > 0 {
		keys = append(keys, cfg.Keys[0])
	}
	cfg.Keys = keys
	cfgData, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	enckeys["aescbc"] = string(cfgData)

	_, err = vc.Logical().Write(cke.K8sSecret, enckeys)
	return err
}

// generateKey generates key for aescbc
// ref: https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/#providers
func generateKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}
