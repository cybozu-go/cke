package cke

import (
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/namespace"
)

const (
	defaultEtcdPrefix = "/cke/"
)

// EtcdConfig represents configuration parameters to access etcd.
type EtcdConfig struct {
	Servers  []string `yaml:"servers"`
	Prefix   string   `yaml:"prefix"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
}

// NewEtcdConfig creates EtcdConfig with default values.
func NewEtcdConfig() *EtcdConfig {
	return &EtcdConfig{
		Prefix: defaultEtcdPrefix,
	}
}

// Client creates etcd client.
func (ec *EtcdConfig) Client() (*clientv3.Client, error) {
	etcdCfg := clientv3.Config{
		Endpoints:   ec.Servers,
		DialTimeout: 2 * time.Second,
		Username:    ec.Username,
		Password:    ec.Password,
	}
	client, err := clientv3.New(etcdCfg)
	if err != nil {
		return nil, err
	}
	client.KV = namespace.NewKV(client.KV, ec.Prefix)
	client.Watcher = namespace.NewWatcher(client.Watcher, ec.Prefix)
	client.Lease = namespace.NewLease(client.Lease, ec.Prefix)

	return client, nil
}
