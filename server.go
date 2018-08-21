package cke

import (
	"net/http"

	"github.com/coreos/etcd/clientv3"
)

// Server is the cke server.
type Server struct {
	EtcdClient *clientv3.Client
}

type version struct {
	Version string `json:"version"`
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/version" {
		s.handleVersion(w, r)
	} else {
		renderError(r.Context(), w, APIErrNotFound)
	}
}

func (s Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	renderJSON(w, version{
		VERSION,
	}, http.StatusOK)
}
