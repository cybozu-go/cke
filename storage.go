package cke

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/coreos/etcd/clientv3"
)

// Storage provides operations to store/retrieve CKE data in etcd.
type Storage struct {
	*clientv3.Client
}

const (
	keyRecords     = "records/"
	keyCluster     = "cluster"
	keyConstraints = "constraints"
)

var (
	// ErrNotFound may be returned by Storage methods when a key is not found.
	ErrNotFound = errors.New("not found")
)

// PutCluster stores *Cluster into etcd.
func (s Storage) PutCluster(ctx context.Context, c *Cluster) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	_, err = s.Put(ctx, keyCluster, string(data))
	return err
}

// GetCluster loads *Cluster from etcd.
// If cluster configuration has not been stored, this returns ErrNotFound.
func (s Storage) GetCluster(ctx context.Context) (*Cluster, error) {
	resp, err := s.Get(ctx, keyCluster)
	if err != nil {
		return nil, err
	}

	if resp.Count == 0 {
		return nil, ErrNotFound
	}

	c := new(Cluster)
	err = json.Unmarshal(resp.Kvs[0].Value, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// PutConstraints stores *Constraints into etcd.
func (s Storage) PutConstraints(ctx context.Context, c *Constraints) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	_, err = s.Put(ctx, keyConstraints, string(data))
	return err
}

// GetConstraints loads *Constraints from etcd.
// If constraints have not been stored, this returns ErrNotFound.
func (s Storage) GetConstraints(ctx context.Context) (*Constraints, error) {
	resp, err := s.Get(ctx, keyConstraints)
	if err != nil {
		return nil, err
	}

	if resp.Count == 0 {
		return nil, ErrNotFound
	}

	c := new(Constraints)
	err = json.Unmarshal(resp.Kvs[0].Value, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
