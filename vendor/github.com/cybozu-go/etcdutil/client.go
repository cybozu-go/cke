package etcdutil

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/namespace"
)

// NewClient creates etcd client.
func NewClient(c *Config) (*clientv3.Client, error) {
	timeout, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return nil, err
	}

	cfg := clientv3.Config{
		Endpoints:   c.Endpoints,
		DialTimeout: timeout,
		Username:    c.Username,
		Password:    c.Password,
	}

	tlsCfg := &tls.Config{}
	if len(c.TLSCA) != 0 {
		rootCACert, err := ioutil.ReadFile(c.TLSCA)
		if err != nil {
			return nil, err
		}
		rootCAs := x509.NewCertPool()
		ok := rootCAs.AppendCertsFromPEM(rootCACert)
		if !ok {
			return nil, errors.New("Failed to parse PEM file")
		}
		tlsCfg.RootCAs = rootCAs
		cfg.TLS = tlsCfg
	}
	if len(c.TLSCert) != 0 && len(c.TLSKey) != 0 {
		cert, err := tls.LoadX509KeyPair(c.TLSCert, c.TLSKey)
		if err != nil {
			return nil, err
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
		cfg.TLS = tlsCfg
	}

	client, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}
	if c.Prefix != "" {
		client.KV = namespace.NewKV(client.KV, c.Prefix)
		client.Watcher = namespace.NewWatcher(client.Watcher, c.Prefix)
		client.Lease = namespace.NewLease(client.Lease, c.Prefix)
	}

	return client, nil
}
