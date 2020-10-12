package cke

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/google/go-cmp/cmp"
)

func testConfigVersion(t *testing.T) {
	client := newEtcdClient(t)
	defer client.Close()
	storage := Storage{client}

	ctx := context.Background()
	version, err := storage.GetConfigVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version != "1" {
		t.Errorf("version is not 1: version %s", version)
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
	err = storage.PutConfigVersion(ctx, leaderKey)
	if err != nil {
		t.Fatal(err)
	}
	version, err = storage.GetConfigVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version != ConfigVersion {
		t.Errorf("version is not %s: version %s", ConfigVersion, version)
	}
}

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

	isLeader, err := storage.IsLeader(ctx, leaderKey)
	if err != nil {
		t.Fatal(err)
	}
	if !isLeader {
		t.Error("failed to confirm leadership")
	}

	r := NewRecord(1, "my-operation-1", []string{})
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

	for i := int64(2); i <= 400; i++ {
		record := NewRecord(i, fmt.Sprintf("my-operation-%d", i), []string{})
		err = storage.RegisterRecord(ctx, leaderKey, record)
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err = storage.GetRecords(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 10 {
		t.Error("length mismatch", len(got))
	}

	ch1, err := storage.WatchRecords(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	ch2, err := storage.WatchRecords(ctx, 150)
	if err != nil {
		t.Fatal(err)
	}
	ch3, err := storage.WatchRecords(ctx, 500)
	if err != nil {
		t.Fatal(err)
	}

	for i := int64(401); i <= 600; i++ {
		record := NewRecord(i, fmt.Sprintf("my-operation-%d", i), []string{})
		err = storage.RegisterRecord(ctx, leaderKey, record)
		if err != nil {
			t.Fatal(err)
		}
	}

	expectID := int64(381)
	for result := range ch1 {
		expectOperation := fmt.Sprintf("my-operation-%d", expectID)
		if result.ID != expectID {
			t.Fatalf("drop record: %d", expectID)
		}
		if result.Operation != expectOperation {
			t.Fatalf("invalid record: %v", result)
		}

		if result.ID == 600 {
			break
		}
		expectID++
	}

	expectID = int64(251)
	for result := range ch2 {
		expectOperation := fmt.Sprintf("my-operation-%d", expectID)
		if result.ID != expectID {
			t.Fatalf("drop record: %d", expectID)
		}
		if result.Operation != expectOperation {
			t.Fatalf("invalid record: %v", result)
		}

		if result.ID == 600 {
			break
		}
		expectID++
	}

	expectID = int64(1)
	for result := range ch3 {
		expectOperation := fmt.Sprintf("my-operation-%d", expectID)
		if result.ID != expectID {
			t.Fatalf("drop record: %d", expectID)
		}
		if result.Operation != expectOperation {
			t.Fatalf("invalid record: %v", result)
		}

		if result.ID == 600 {
			break
		}
		expectID++
	}

	record := NewRecord(80, "updated", []string{})
	err = storage.UpdateRecord(ctx, leaderKey, record)
	if err != nil {
		t.Fatal(err)
	}

	result := <-ch1
	if result.ID != 80 {
		t.Fatalf("cannot watch updated record: %v", result)
	}
	if result.Operation != "updated" {
		t.Fatalf("invalid record: %v", result)
	}

	result = <-ch2
	if result.ID != 80 {
		t.Fatalf("cannot watch updated record: %v", result)
	}
	if result.Operation != "updated" {
		t.Fatalf("invalid record: %v", result)
	}

	result = <-ch3
	if result.ID != 80 {
		t.Fatalf("cannot watch updated record: %v", result)
	}
	if result.Operation != "updated" {
		t.Fatalf("invalid record: %v", result)
	}

	err = e.Resign(ctx)
	if err != nil {
		t.Fatal(err)
	}

	isLeader, err = storage.IsLeader(ctx, leaderKey)
	if err != nil {
		t.Fatal(err)
	}
	if isLeader {
		t.Error("failed to confirm loss of leadership")
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
		r := NewRecord(i, "my-operation", []string{})
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

func testStorageResource(t *testing.T) {
	t.Parallel()

	client := newEtcdClient(t)
	defer client.Close()
	storage := Storage{client}
	ctx := context.Background()

	keys, err := storage.ListResources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 0 {
		t.Error(`len(keys) != 0`)
	}

	_, _, err = storage.GetResource(ctx, "Namespace/foo")
	if err != ErrNotFound {
		t.Error(`err != ErrNotFound,`, err)
	}

	err = storage.SetResource(ctx, "Namespace/foo", "bar")
	if err != nil {
		t.Fatal(err)
	}

	fooVal, _, err := storage.GetResource(ctx, "Namespace/foo")
	if err != nil {
		t.Fatal(err)
	}
	if string(fooVal) != "bar" {
		t.Error(`string(fooVal) != "bar",`, string(fooVal))
	}

	err = storage.SetResource(ctx, "Pod/foo/pod1", "test")
	if err != nil {
		t.Fatal(err)
	}

	keys, err = storage.ListResources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	expectedKeys := []string{"Namespace/foo", "Pod/foo/pod1"}
	if !cmp.Equal(expectedKeys, keys) {
		t.Error("unexpected list result:", cmp.Diff(expectedKeys, keys))
	}

	err = storage.SetResource(ctx, "ConfigMap/foo/conf1", "test")
	if err != nil {
		t.Fatal(err)
	}

	err = storage.SetResource(ctx, "ServiceAccount/foo/sa1", "test")
	if err != nil {
		t.Fatal(err)
	}

	resources, err := storage.GetAllResources(ctx)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"Namespace/foo", "ServiceAccount/foo/sa1", "ConfigMap/foo/conf1", "Pod/foo/pod1"}
	actual := make([]string, len(resources))
	for i, r := range resources {
		actual[i] = r.Key
	}
	if !cmp.Equal(expected, actual) {
		t.Error("unexpected resource list", cmp.Diff(expected, actual))
	}

	err = storage.DeleteResource(ctx, "Namespace/foo")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = storage.GetResource(ctx, "Namespace/foo")
	if err != ErrNotFound {
		t.Error(`err != ErrNotFound,`, err)
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

	disabled, err := s.IsSabakanDisabled(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if disabled {
		t.Error("sabakan integration should not be disabled by default")
	}

	err = s.EnableSabakan(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	disabled, err = s.IsSabakanDisabled(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !disabled {
		t.Error("sabakan integration could not be disabled")
	}

	err = s.EnableSabakan(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	disabled, err = s.IsSabakanDisabled(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if disabled {
		t.Error("sabakan integration could not be re-enabled")
	}
}

func testStatus(t *testing.T) {
	t.Parallel()

	client := newEtcdClient(t)
	defer client.Close()
	s := Storage{client}
	ctx := context.Background()

	_, err := s.GetStatus(ctx)
	if err != ErrNotFound {
		t.Error("unexpected error:", err)
	}

	resp, err := client.Grant(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	err = s.SetStatus(ctx, resp.ID, &ServerStatus{Phase: PhaseCompleted, Timestamp: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}

	status, err := s.GetStatus(ctx)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if status.Phase != PhaseCompleted {
		t.Error("wrong phase:", status.Phase)
	}

	_, err = client.Revoke(ctx, resp.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.GetStatus(ctx)
	if err != ErrNotFound {
		t.Error("err is not ErrNotFound. err=", err)
	}
}

func TestStorage(t *testing.T) {
	t.Run("ConfigVersion", testConfigVersion)
	t.Run("Cluster", testStorageCluster)
	t.Run("Constraints", testStorageConstraints)
	t.Run("Record", testStorageRecord)
	t.Run("Maint", testStorageMaint)
	t.Run("Resource", testStorageResource)
	t.Run("Sabakan", testStorageSabakan)
	t.Run("Status", testStatus)
}
