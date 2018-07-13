package cke

import (
	"context"
	"encoding/json"
	"errors"

	"fmt"
	"strconv"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/clientv3util"
)

// Storage provides operations to store/retrieve CKE data in etcd.
type Storage struct {
	*clientv3.Client
}

const (
	keyRecords     = "records/"
	keyRecordID    = "records"
	keyCluster     = "cluster"
	keyConstraints = "constraints"
	keyLeader      = "leader/"
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

func recordKey(r *Record) string {
	return fmt.Sprintf("%s%016x", keyRecords, r.ID)
}

// RegisterRecord stores *Record if the leaderKey exists
func (s Storage) RegisterRecord(ctx context.Context, leaderKey string, r *Record) error {
	nextID := strconv.FormatInt(r.ID+1, 10)
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	resp, err := s.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(
			clientv3.OpPut(recordKey(r), string(data)),
			clientv3.OpPut(keyRecordID, nextID)).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return errors.New("lose leadership")
	}
	return nil
}

// NextRecordID get the next record ID from etcd
func (s Storage) NextRecordID(ctx context.Context) (int64, error) {
	resp, err := s.Get(ctx, keyRecordID)
	if err != nil {
		return 0, err
	}
	if resp.Count == 0 {
		return 1, nil
	}

	id, err := strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}
