package localproxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op/k8s"
	"github.com/cybozu-go/cke/op/nodedns"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// the current status for running local proxy
type status struct {
	apiServers   []string
	proxyRunning bool
	proxyImage   string

	unboundConf        []byte
	desiredUnboundConf []byte
	unboundRunning     bool
	unboundImage       string
}

var dialer = &net.Dialer{
	Timeout: 5 * time.Second,
}

func isRunning(name string) (bool, string, error) {
	out, err := exec.Command("docker", "container", "inspect", "--format={{.State.Running}} {{.Config.Image}}", name).Output()
	if err != nil {
		// Container does not exist
		return false, "", nil
	}
	fields := strings.Fields(string(out))
	if len(fields) != 2 {
		return false, "", fmt.Errorf("unexpected docker inspect output: %s", out)
	}
	return fields[0] == "true", fields[1], nil
}

func getStatus(ctx context.Context, inf cke.Infrastructure) (*status, error) {
	cluster, err := inf.Storage().GetCluster(ctx)
	if err != nil {
		return nil, err
	}

	var apiServer *cke.Node
	var apiServers []string
	for _, n := range cluster.Nodes {
		if !n.ControlPlane {
			continue
		}
		conn, err := dialer.DialContext(ctx, "tcp", n.Address+":6443")
		if err != nil {
			continue
		}
		conn.Close()
		if apiServer == nil {
			apiServer = n
		}
		apiServers = append(apiServers, n.Address)
	}

	if len(apiServers) == 0 {
		return nil, errors.New("no kube-apiserver is available")
	}

	proxyRunning, proxyImage, err := isRunning("kube-proxy")
	if err != nil {
		return nil, err
	}

	unboundConf, err := os.ReadFile("/etc/unbound/unbound.conf")
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	cs, err := inf.K8sClient(ctx, apiServer)
	if err != nil {
		return nil, err
	}

	clusterDNS, err := cs.CoreV1().Services("kube-system").Get(ctx, "cluster-dns", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster-dns service: %w", err)
	}
	if len(clusterDNS.Spec.ClusterIP) == 0 {
		return nil, errors.New("no clusterIP has been assigned to cluster-dns")
	}

	kubeletConfig := k8s.GenerateKubeletConfiguration(cluster.Options.Kubelet, "0.0.0.0", nil)
	domain := kubeletConfig.ClusterDomain

	dnsServers := cluster.DNSServers
	if cluster.DNSService != "" {
		fields := strings.Split(cluster.DNSService, "/")
		if len(fields) != 2 {
			return nil, fmt.Errorf("invalid service reference in cluster config: %s", cluster.DNSService)
		}

		svc, err := cs.CoreV1().Services(fields[0]).Get(ctx, fields[1], metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get public dns service %s: %w", cluster.DNSService, err)
		}
		if len(svc.Spec.ClusterIP) == 0 {
			dnsServers = nil
		} else {
			dnsServers = []string{svc.Spec.ClusterIP}
		}
	}

	// configuration for cache name server of cke-localproxy should be (almost) same as that for node DNS.
	unboundConfigMap := nodedns.ConfigMap(clusterDNS.Spec.ClusterIP, domain, dnsServers, false)

	unboundRunning, unboundImage, err := isRunning("cke-unbound")
	if err != nil {
		return nil, err
	}

	return &status{
		apiServers:   apiServers,
		proxyRunning: proxyRunning,
		proxyImage:   proxyImage,

		unboundConf:        unboundConf,
		desiredUnboundConf: []byte(unboundConfigMap.Data["unbound.conf"]),
		unboundRunning:     unboundRunning,
		unboundImage:       unboundImage,
	}, nil
}
