package cke

import (
	"context"
	"reflect"
	"testing"

	"github.com/coreos/etcd/clientv3/concurrency"
)

func testStorageCluster(t *testing.T) {
	t.Parallel()

	client := newEtcdClient(t)
	defer client.Close()
	storage := Storage{client}
	ctx := context.Background()

	_, err := storage.GetCluster(ctx)
	if err != ErrNotFound {
		t.Fatal("cluster found.")
	}
	c := &Cluster{
		Name: "my-cluster",
		Nodes: []*Node{
			{
				Address:  "10.0.1.2",
				Hostname: "node1",
				User:     "cybozu",
			},
		},
		SSHKey: "aaa",
		DNSServers: []string{
			"8.8.8.8",
			"8.8.4.4",
		},
	}
	err = storage.PutCluster(ctx, c)
	if err != nil {
		t.Fatal(err)
	}

	got, err := storage.GetCluster(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(c, got) {
		t.Fatalf("got invalid cluster: %v", got)
	}
}

func testStorageConstraints(t *testing.T) {
	t.Parallel()

	client := newEtcdClient(t)
	defer client.Close()
	storage := Storage{client}
	ctx := context.Background()

	_, err := storage.GetConstraints(ctx)
	if err != ErrNotFound {
		t.Fatal("constraints found.")
	}
	c := &Constraints{
		ControlPlaneCount: 3,
		MinimumWorkers:    3,
		MaximumWorkers:    100,
	}
	err = storage.PutConstraints(ctx, c)
	if err != nil {
		t.Fatal(err)
	}

	got, err := storage.GetConstraints(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(c, got) {
		t.Fatalf("got invalid constraints: %v", got)
	}
}

func testStorageRecord(t *testing.T) {
	t.Parallel()

	client := newEtcdClient(t)
	defer client.Close()
	storage := Storage{client}
	ctx := context.Background()

	rs, err := storage.GetRecords(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 0 {
		t.Fatalf("records found.%d", len(rs))
	}

	s, err := concurrency.NewSession(client)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	e := concurrency.NewElection(s, KeyLeader)
	err = e.Campaign(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}

	leaderKey := e.Key()
	r := NewRecord(1, "my-operation")
	err = storage.RegisterRecord(ctx, leaderKey, r)
	if err != nil {
		t.Fatal(err)
	}

	got, err := storage.GetRecords(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(r, got[0]) {
		t.Fatalf("got invalid record: %#v, %#v", r, got[0])
	}

	nextID, err := storage.NextRecordID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if nextID != r.ID+1 {
		t.Fatalf("nextID was not incremented: %d", nextID)
	}

	newR := *r
	newR.Complete()
	err = storage.UpdateRecord(ctx, leaderKey, r)
	if err != nil {
		t.Fatal(err)
	}

	got, err = storage.GetRecords(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(r, got[0]) {
		t.Fatalf("got invalid record: %v", got[0])
	}

	err = e.Resign(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = storage.RegisterRecord(ctx, leaderKey, r)
	if err != ErrNoLeader {
		t.Fatal("leader did not resign")
	}

	err = storage.UpdateRecord(ctx, leaderKey, &newR)
	if err != ErrNoLeader {
		t.Fatal("leader did not resign")
	}
}

func testStorageMaint(t *testing.T) {
	t.Parallel()

	client := newEtcdClient(t)
	defer client.Close()
	storage := Storage{client}
	ctx := context.Background()

	s, err := concurrency.NewSession(client)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	e := concurrency.NewElection(s, KeyLeader)
	err = e.Campaign(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}

	leaderKey := e.Key()
	for i := int64(1); i < 11; i++ {
		r := NewRecord(i, "my-operation")
		err = storage.RegisterRecord(ctx, leaderKey, r)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = storage.maintRecords(ctx, leaderKey, 100)
	if err != nil {
		t.Fatal(err)
	}

	records, err := storage.GetRecords(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 10 {
		t.Error(`len(records) != 10`)
	}

	err = storage.maintRecords(ctx, leaderKey, 8)
	if err != nil {
		t.Fatal(err)
	}

	records, err = storage.GetRecords(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 8 {
		t.Fatal(`len(records) != 8`)
	}

	if records[7].ID != 3 {
		t.Error(`records[7].ID != 3`)
	}
}

func TestStorage(t *testing.T) {
	t.Run("Cluster", testStorageCluster)
	t.Run("Constraints", testStorageConstraints)
	t.Run("Record", testStorageRecord)
	t.Run("Maint", testStorageMaint)
}
