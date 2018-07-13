package cke

import (
	"context"
	"os"

	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
)

// Controller manage operations
type Controller struct {
	session  *concurrency.Session
	interval time.Duration
}

// Run execute procedures with leader elections
func (c Controller) Run(ctx context.Context) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	storage := Storage{c.session.Client()}
	e := concurrency.NewElection(c.session, keyLeader)

	for {
		err = e.Campaign(ctx, hostname)
		if err != nil {
			return err
		}
		leaderKey := e.Key()
		cluster, err := storage.GetCluster(ctx)
		if err != nil {
			err2 := e.Resign(ctx)
			if err2 != nil {
				return err2
			}
			continue
		}

		status, err := GetClusterStatus(ctx, cluster)
		if err != nil {
			err2 := e.Resign(ctx)
			if err2 != nil {
				return err2
			}
			continue
		}

		op := DecideToDo(cluster, status)

		// register operation record
		id, err := storage.NextRecordID(ctx)
		if err != nil {
			return err
		}
		record := op.NewRecord(id)
		err = storage.RegisterRecord(ctx, leaderKey, record)
		if err != nil {
			err2 := e.Resign(ctx)
			if err2 != nil {
				return err2
			}
			continue
		}
	}
}
