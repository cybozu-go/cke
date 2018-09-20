package cke

import (
	"bytes"
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

// etcd keys and prefixes
const (
	KeyCA                 = "ca/"
	KeyCluster            = "cluster"
	KeyConstraints        = "constraints"
	KeyLeader             = "leader/"
	KeyRecords            = "records/"
	KeyRecordID           = "records"
	KeyServiceAccountCert = "service-account/certificate"
	KeyServiceAccountKey  = "service-account/key"
	KeyVault              = "vault"
)

const maxRecords = 1000

var (
	// ErrNotFound may be returned by Storage methods when a key is not found.
	ErrNotFound = errors.New("not found")
	// ErrNoLeader is returned when the session lost leadership.
	ErrNoLeader = errors.New("lost leadership")
)

// PutCluster stores *Cluster into etcd.
func (s Storage) PutCluster(ctx context.Context, c *Cluster) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	_, err = s.Put(ctx, KeyCluster, string(data))
	return err
}

// GetCluster loads *Cluster from etcd.
// If cluster configuration has not been stored, this returns ErrNotFound.
func (s Storage) GetCluster(ctx context.Context) (*Cluster, error) {
	resp, err := s.Get(ctx, KeyCluster)
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

	_, err = s.Put(ctx, KeyConstraints, string(data))
	return err
}

// GetConstraints loads *Constraints from etcd.
// If constraints have not been stored, this returns ErrNotFound.
func (s Storage) GetConstraints(ctx context.Context) (*Constraints, error) {
	resp, err := s.Get(ctx, KeyConstraints)
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

// PutVaultConfig stores *VaultConfig into etcd.
func (s Storage) PutVaultConfig(ctx context.Context, c *VaultConfig) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	_, err = s.Put(ctx, KeyVault, string(data))
	return err
}

// GetVaultConfig loads *VaultConfig from etcd.
func (s Storage) GetVaultConfig(ctx context.Context) (*VaultConfig, error) {
	resp, err := s.Get(ctx, KeyVault)
	if err != nil {
		return nil, err
	}
	if resp.Count > 1 {
		return nil, errors.New("VaultConfig is something wrong")
	}

	cfg := new(VaultConfig)
	err = json.NewDecoder(bytes.NewReader(resp.Kvs[0].Value)).Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// GetCACertificate loads CA certificate from etcd.
func (s Storage) GetCACertificate(ctx context.Context, name string) (string, error) {
	resp, err := s.Get(ctx, KeyCA+name)
	if err != nil {
		return "", err
	}
	if resp.Count == 0 {
		return "", ErrNotFound
	}

	return string(resp.Kvs[0].Value), nil
}

// PutCACertificate stores CA certificate into etcd.
func (s Storage) PutCACertificate(ctx context.Context, name, pem string) error {
	_, err := s.Put(ctx, KeyCA+name, pem)
	return err
}

func recordKey(r *Record) string {
	return fmt.Sprintf("%s%016x", KeyRecords, r.ID)
}

// GetServiceAccountCert loads x509 certificate for service account.
// The format is PEM.
func (s Storage) GetServiceAccountCert(ctx context.Context) (string, error) {
	resp, err := s.Get(ctx, KeyServiceAccountCert)
	if err != nil {
		return "", err
	}
	if resp.Count == 0 {
		return "", ErrNotFound
	}

	return string(resp.Kvs[0].Value), nil
}

// GetServiceAccountKey loads private key for service account.
// The format is PEM.
func (s Storage) GetServiceAccountKey(ctx context.Context) (string, error) {
	resp, err := s.Get(ctx, KeyServiceAccountKey)
	if err != nil {
		return "", err
	}
	if resp.Count == 0 {
		return "", ErrNotFound
	}

	return string(resp.Kvs[0].Value), nil
}

// PutServiceAccountData stores x509 certificate and private key for service account.
func (s Storage) PutServiceAccountData(ctx context.Context, leaderKey, cert, key string) error {
	resp, err := s.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(
			clientv3.OpPut(KeyServiceAccountCert, cert),
			clientv3.OpPut(KeyServiceAccountKey, key)).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrNoLeader
	}
	return nil
}

// GetRecords loads list of *Record from etcd.
// The returned records are sorted by record ID in decreasing order.
func (s Storage) GetRecords(ctx context.Context, count int64) ([]*Record, error) {
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend),
	}
	if count > 0 {
		opts = append(opts, clientv3.WithLimit(count))
	}
	resp, err := s.Get(ctx, KeyRecords, opts...)
	if err != nil {
		return nil, err
	}

	if resp.Count == 0 {
		return nil, nil
	}

	records := make([]*Record, resp.Count)

	for i, kv := range resp.Kvs {
		r := new(Record)
		err = json.Unmarshal(kv.Value, r)
		if err != nil {
			return nil, err
		}
		records[i] = r
	}

	return records, nil
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
			clientv3.OpPut(KeyRecordID, nextID)).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrNoLeader
	}

	return s.maintRecords(ctx, leaderKey, maxRecords)
}

// UpdateRecord updates existing record
func (s Storage) UpdateRecord(ctx context.Context, leaderKey string, r *Record) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	resp, err := s.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(clientv3.OpPut(recordKey(r), string(data))).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrNoLeader
	}
	return nil
}

// NextRecordID get the next record ID from etcd
func (s Storage) NextRecordID(ctx context.Context) (int64, error) {
	resp, err := s.Get(ctx, KeyRecordID)
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

func (s Storage) maintRecords(ctx context.Context, leaderKey string, max int64) error {
	resp, err := s.Get(ctx, KeyRecords,
		clientv3.WithPrefix(),
		clientv3.WithKeysOnly(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
	)
	if err != nil {
		return err
	}

	if resp.Count <= max {
		return nil
	}

	startKey := string(resp.Kvs[0].Key)
	endKey := string(resp.Kvs[resp.Count-max].Key)

	tresp, err := s.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(clientv3.OpDelete(startKey, clientv3.WithRange(endKey))).
		Commit()
	if !tresp.Succeeded {
		return ErrNoLeader
	}
	return err
}

// GetLeaderHostname returns the current leader's host name.
// It returns non-nil error when there is no leader.
func (s Storage) GetLeaderHostname(ctx context.Context) (string, error) {
	opts := []clientv3.OpOption{clientv3.WithPrefix()}
	opts = append(opts, clientv3.WithFirstCreate()...)
	resp, err := s.Get(ctx, KeyLeader, opts...)
	if err != nil {
		return "", err
	}

	if resp.Count == 0 {
		return "", errors.New("no leader")
	}
	return string(resp.Kvs[0].Value), nil
}
