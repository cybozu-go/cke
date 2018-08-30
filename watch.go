package cke

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"time"

	"encoding/json"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/log"
	vault "github.com/hashicorp/vault/api"
)

// connectVault creates vault client
func connectVault(data []byte) error {
	c := new(VaultConfig)
	err := json.Unmarshal(data, c)
	if err != nil {
		return err
	}

	transport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if len(c.CACert) > 0 {
		cp := x509.NewCertPool()
		if !cp.AppendCertsFromPEM([]byte(c.CACert)) {
			return errors.New("invalid CA cert")
		}

		transport.TLSClientConfig = &tls.Config{
			RootCAs:    cp,
			MinVersion: tls.VersionTLS12,
		}
	}

	client, err := vault.NewClient(&vault.Config{
		Address: c.Endpoint,
		HttpClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		log.Error("failed to connect to vault", map[string]interface{}{
			log.FnError: err,
			"endpoint":  c.Endpoint,
		})
		return err
	}

	setVaultClient(client)
	return nil
}

func initStateless(ctx context.Context, etcd *clientv3.Client, ch chan<- struct{}) (int64, error) {
	defer func() {
		// notify the caller of the readiness
		ch <- struct{}{}
	}()

	resp, err := etcd.Get(ctx, "/vault")
	if err != nil {
		return 0, err
	}
	rev := resp.Header.Revision

	if resp.Count == 1 {
		err = connectVault(resp.Kvs[0].Value)
		if err != nil {
			return 0, err
		}
	}

	return rev, nil
}

func startWatcher(ctx context.Context, etcd *clientv3.Client, ch chan<- struct{}) error {
	rev, err := initStateless(ctx, etcd, ch)
	if err != nil {
		return err
	}

	wch := etcd.Watch(ctx, "", clientv3.WithPrefix(), clientv3.WithRev(rev+1))
	for resp := range wch {
		for _, ev := range resp.Events {
			if ev.Type != clientv3.EventTypePut {
				continue
			}

			key := string(ev.Kv.Key)
			switch key {
			case KeyCluster:
				//TODO
			case KeyVault:
				err = connectVault(ev.Kv.Value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
