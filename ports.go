package cke

import (
	"net"
	"strconv"
)

// Port numbers of the control plane components.
const (
	// APIServerPort is the port number that kube-apiserver listens on
	APIServerPort = 6443
	// EtcdClientPort is the port number that etcd listens on for clients
	EtcdClientPort = 2379
)

// Port numbers that rivers and etcd-rivers listen on.
const (
	// RiversListenPort is listen port of rivers container
	RiversListenPort = 16443
	// EtcdRiversListenPort is listen port of etcd-rivers container
	EtcdRiversListenPort = 12379
)

// APIServerURL returns the URL of kube-apiserver running on the given address.
func APIServerURL(address string) string {
	return "https://" + net.JoinHostPort(address, strconv.Itoa(APIServerPort))
}

// EtcdClientURL returns the URL of etcd for clients running on the given address.
func EtcdClientURL(address string) string {
	return "https://" + net.JoinHostPort(address, strconv.Itoa(EtcdClientPort))
}

// RiversURL returns the URL of rivers listening on the given address.
func RiversURL(address string) string {
	return "https://" + net.JoinHostPort(address, strconv.Itoa(RiversListenPort))
}

// EtcdRiversURL returns the URL of etcd-rivers listening on the given address.
func EtcdRiversURL(address string) string {
	return "https://" + net.JoinHostPort(address, strconv.Itoa(EtcdRiversListenPort))
}
