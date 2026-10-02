package etcd

import (
	"slices"
	"strings"
	"testing"

	"github.com/cybozu-go/cke"
)

func TestBuiltInParamsCompaction(t *testing.T) {
	args := BuiltInParams(&cke.Node{Address: "10.0.0.11"}, nil, "").ExtraArguments

	// Compaction must be done only by kube-apiserver.  See docs/etcd.md.
	if i := slices.IndexFunc(args, func(arg string) bool {
		return strings.HasPrefix(arg, "--auto-compaction")
	}); i >= 0 {
		t.Error("etcd must not enable auto-compaction:", args[i])
	}

	// Disabling auto-compaction must not disable data corruption detection.
	for _, expected := range []string{
		"--feature-gates=InitialCorruptCheck=true,CompactHashCheck=true",
		"--corrupt-check-time=3h",
	} {
		if !slices.Contains(args, expected) {
			t.Error("etcd must be started with:", expected)
		}
	}
}
