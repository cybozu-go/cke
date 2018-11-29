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
	if len(got) != 1 {
		t.Fatal("record was not registered")
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

func testStorageSabakan(t *testing.T) {
	t.Parallel()

	client := newEtcdClient(t)
	defer client.Close()
	s := Storage{client}
	ctx := context.Background()

	_, rev, err := s.GetClusterWithRevision(ctx)
	if err != ErrNotFound {
		t.Error("unexpected error:", err)
	}
	if rev != 0 {
		t.Error("unexpected revision:", rev)
	}

	_, err = s.GetSabakanQueryVariables(ctx)
	if err != ErrNotFound {
		t.Error(`err != ErrNotFound,`, err)
	}

	const vars = `{"having": {"racks": [0, 1, 2]}}`
	err = s.SetSabakanQueryVariables(ctx, vars)
	if err != nil {
		t.Fatal(err)
	}

	vars2, err := s.GetSabakanQueryVariables(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if string(vars2) != vars {
		t.Error("unexpected query variables:", string(vars2))
	}

	_, rev, err = s.GetSabakanTemplate(ctx)
	if err != ErrNotFound {
		t.Error(`err != ErrNotFound,`, err)
	}
	if rev != 0 {
		t.Error(`rev != 0`, rev)
	}

	tmpl := &Cluster{Name: "aaa"}
	err = s.SetSabakanTemplate(ctx, tmpl)
	if err != nil {
		t.Fatal(err)
	}

	tmpl2, rev, err := s.GetSabakanTemplate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rev == 0 {
		t.Error(`rev == 0`)
	}
	if tmpl2.Name != tmpl.Name {
		t.Error(`tmpl2.Name != tmpl.Name`, tmpl2.Name)
	}

	err = s.PutCluster(ctx, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	c, rev2, err := s.GetClusterWithRevision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Error("c is nil")
	}
	if rev2 != 0 {
		t.Error("unexpected revision:", rev2)
	}

	err = s.PutClusterWithTemplateRevision(ctx, tmpl, rev, KeySabakanTemplate)
	if err != nil {
		t.Fatal(err)
	}
	c, rev2, err = s.GetClusterWithRevision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Error("c is nil")
	}
	if rev2 != rev {
		t.Error(`rev2 != rev`, rev2, rev)
	}

	_, err = s.GetSabakanURL(ctx)
	if err != ErrNotFound {
		t.Error(`err != ErrNotFound`, err)
	}

	u := "http://localhost:12345"
	err = s.SetSabakanURL(ctx, u)
	if err != nil {
		t.Fatal(err)
	}

	u2, err := s.GetSabakanURL(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if u2 != u {
		t.Error(`u2 != u`, u2)
	}

	err = s.DeleteSabakanURL(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.GetSabakanURL(ctx)
	if err != ErrNotFound {
		t.Error("URL was not removed")
	}
}

func TestStorage(t *testing.T) {
	t.Run("Cluster", testStorageCluster)
	t.Run("Constraints", testStorageConstraints)
	t.Run("Record", testStorageRecord)
	t.Run("Maint", testStorageMaint)
	t.Run("Sabakan", testStorageSabakan)
}
