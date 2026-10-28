package cke

import "testing"

func TestPortURLs(t *testing.T) {
	tests := []struct {
		address   string
		apiServer string
		etcd      string
		rivers    string
		etcdRiver string
	}{
		{
			address:   "192.0.2.1",
			apiServer: "https://192.0.2.1:6443",
			etcd:      "https://192.0.2.1:2379",
			rivers:    "https://192.0.2.1:16443",
			etcdRiver: "https://192.0.2.1:12379",
		},
		{
			address:   "2001:db8::1",
			apiServer: "https://[2001:db8::1]:6443",
			etcd:      "https://[2001:db8::1]:2379",
			rivers:    "https://[2001:db8::1]:16443",
			etcdRiver: "https://[2001:db8::1]:12379",
		},
		{
			address:   "localhost",
			apiServer: "https://localhost:6443",
			etcd:      "https://localhost:2379",
			rivers:    "https://localhost:16443",
			etcdRiver: "https://localhost:12379",
		},
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			if got := APIServerURL(tt.address); got != tt.apiServer {
				t.Errorf("APIServerURL() = %q, want %q", got, tt.apiServer)
			}
			if got := EtcdClientURL(tt.address); got != tt.etcd {
				t.Errorf("EtcdClientURL() = %q, want %q", got, tt.etcd)
			}
			if got := RiversURL(tt.address); got != tt.rivers {
				t.Errorf("RiversURL() = %q, want %q", got, tt.rivers)
			}
			if got := EtcdRiversURL(tt.address); got != tt.etcdRiver {
				t.Errorf("EtcdRiversURL() = %q, want %q", got, tt.etcdRiver)
			}
		})
	}
}
