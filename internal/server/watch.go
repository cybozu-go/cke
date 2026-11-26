package server

import (
	"context"
	"strings"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/cybozu-go/cke"
)

// startVaultWatcher connects to Vault with the configuration in etcd.
// It reconnects when the configuration is updated.
//
// ready is closed after the first connection succeeds.
// If Vault is not configured, ready is closed without connecting.
// If the first connection fails, it returns the error without closing ready.
func startVaultWatcher(ctx context.Context, etcd *clientv3.Client, ready chan<- struct{}) error {
	resp, err := etcd.Get(ctx, cke.KeyVault)
	if err != nil {
		return err
	}
	if len(resp.Kvs) == 1 {
		if err := cke.ConnectVault(ctx, resp.Kvs[0].Value); err != nil {
			return err
		}
	}
	close(ready)

	wch := etcd.Watch(ctx, cke.KeyVault,
		clientv3.WithRev(resp.Header.Revision+1),
		clientv3.WithFilterDelete(),
	)
	for resp := range wch {
		if err := resp.Err(); err != nil {
			return err
		}
		for _, ev := range resp.Events {
			if err := cke.ConnectVault(ctx, ev.Kv.Value); err != nil {
				return err
			}
		}
	}
	return nil
}

// startConfigWatcher watches the cluster configuration and user-defined resources.
// It notifies ch when they are created or updated in etcd.
// Deletions are not notified.
//
// ready is closed when the starting revision of the watch is fixed.
// The watch detects every update after that.
// However, a notification is dropped if no one is waiting on ch.
func startConfigWatcher(ctx context.Context, etcd *clientv3.Client, ready chan<- struct{}, ch chan<- struct{}) error {
	resp, err := etcd.Get(ctx, cke.KeyCluster, clientv3.WithCountOnly())
	if err != nil {
		return err
	}

	wch := etcd.Watch(ctx, "",
		clientv3.WithPrefix(),
		clientv3.WithRev(resp.Header.Revision+1),
		clientv3.WithFilterDelete(),
	)
	close(ready)

	for resp := range wch {
		if err := resp.Err(); err != nil {
			return err
		}
		for _, ev := range resp.Events {
			key := string(ev.Kv.Key)
			if key != cke.KeyCluster && !strings.HasPrefix(key, cke.KeyResourcePrefix) {
				continue
			}
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}
	return nil
}
