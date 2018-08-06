package cke

import "github.com/cybozu-go/etcdutil"

const (
	defaultEtcdPrefix = "/cke/"
)

// NewEtcdConfig creates Config with default prefix.
func NewEtcdConfig() *etcdutil.Config {
	return etcdutil.NewConfig(defaultEtcdPrefix)
}
