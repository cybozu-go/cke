package cli

import (
	"encoding/json"
	"os"
)

// ServiceAccount represents service account key
type ServiceAccount struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

// LoadAccountJSON returns ServiceAccount from its key file JSON
func LoadAccountJSON(filename string) (*ServiceAccount, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ss ServiceAccount
	err = json.NewDecoder(file).Decode(&ss)
	if err != nil {
		return nil, err
	}
	return &ss, nil
}
