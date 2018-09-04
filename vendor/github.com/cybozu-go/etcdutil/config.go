package etcdutil

const (
	// DefaultTimeout is default etcd connection timeout.
	DefaultTimeout = "2s"
)

var (
	// DefaultEndpoints is default etcd servers.
	DefaultEndpoints = []string{"http://127.0.0.1:2379", "http://127.0.0.1:4001"}
)

// Config represents configuration parameters to access etcd.
type Config struct {
	// Endpoints are etcd servers.
	Endpoints []string `yaml:"endpoints" json:"endpoints" toml:"endpoints"`
	// Prefix is etcd prefix key.
	Prefix string `yaml:"prefix" json:"prefix" toml:"prefix"`
	// Timeout is dial timeout of the etcd client connection.
	Timeout string `yaml:"timeout" json:"timeout" toml:"timeout"`
	// Username is username for loging in to the etcd.
	Username string `yaml:"username" json:"username" toml:"username"`
	// Password is password for loging in to the etcd.
	Password string `yaml:"password" json:"password" toml:"password"`
	// TLSCA is root CA path.
	TLSCA string `yaml:"tls-ca" json:"tls-ca" toml:"tls-ca"`
	// TLSCert is TLS client certificate path.
	TLSCert string `yaml:"tls-cert" json:"tls-cert" toml:"tls-cert"`
	// TLSKey is TLS client private key path.
	TLSKey string `yaml:"tls-key" json:"tls-key" toml:"tls-key"`
}

// NewConfig creates Config with default values.
func NewConfig(prefix string) *Config {
	return &Config{
		Endpoints: DefaultEndpoints,
		Prefix:    prefix,
		Timeout:   DefaultTimeout,
	}
}
