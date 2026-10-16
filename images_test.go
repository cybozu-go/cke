package cke

import "testing"

func TestImageMatchesRunning(t *testing.T) {
	t.Parallel()

	img := newImage("ghcr.io/cybozu/etcd", "3.6.11.1", "sha256:0123")
	tests := []struct {
		name    string
		running string
		want    bool
	}{
		{name: "full reference", running: "ghcr.io/cybozu/etcd:3.6.11.1@sha256:0123", want: true},
		{name: "tag only from older CKE", running: "ghcr.io/cybozu/etcd:3.6.11.1", want: true},
		{name: "other digest", running: "ghcr.io/cybozu/etcd:3.6.11.1@sha256:4567", want: false},
		{name: "other tag", running: "ghcr.io/cybozu/etcd:3.6.10.1", want: false},
		{name: "empty", running: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := img.MatchesRunning(tt.running); got != tt.want {
				t.Errorf("MatchesRunning(%q) = %v, want %v", tt.running, got, tt.want)
			}
		})
	}
}
