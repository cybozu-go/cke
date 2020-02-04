package mock

import (
	"context"
	"errors"
	"path"
	"strings"

	"github.com/cybozu-go/sabakan/v2"
)

// GetEncryptionKey implements sabakan.StorageModel
func (d *driver) GetEncryptionKey(ctx context.Context, serial string, diskByPath string) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	target := path.Join(serial, diskByPath)
	key, ok := d.storage[target]
	if !ok {
		return nil, nil
	}

	return key, nil
}

// PutEncryptionKey implements sabakan.StorageModel
func (d *driver) PutEncryptionKey(ctx context.Context, serial string, diskByPath string, key []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	m, ok := d.machines[serial]
	if !ok {
		return sabakan.ErrNotFound
	}
	if m.Status.State == sabakan.StateRetiring || m.Status.State == sabakan.StateRetired {
		return errors.New("machine was retiring or retired")
	}

	target := path.Join(serial, diskByPath)
	_, ok = d.storage[target]
	if ok {
		return sabakan.ErrConflicted
	}
	d.storage[target] = key

	return nil
}

// DeleteEncryptionKeys implements sabakan.StorageModel
func (d *driver) DeleteEncryptionKeys(ctx context.Context, serial string) ([]string, error) {
	prefix := serial + "/"

	d.mu.Lock()
	defer d.mu.Unlock()

	m, ok := d.machines[serial]
	if !ok {
		return nil, sabakan.ErrNotFound
	}
	if m.Status.State != sabakan.StateRetiring {
		return nil, errors.New("machine is not retiring")
	}

	var resp []string
	for k := range d.storage {
		if strings.HasPrefix(k, prefix) {
			delete(d.storage, k)
			resp = append(resp, k[len(serial)+1:])
		}
	}

	return resp, nil
}
