package localproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/internal/op/k8s"
	"github.com/cybozu-go/cke/internal/op/nodedns"
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

// isRunning returns whether the container named name is running and its image.
// The image is the digest-pinned reference recorded in the CKE label when the
// container was started by this version, or the tag reference otherwise.
func isRunning(name string) (bool, string, error) {
	stdout, err := exec.Command("docker", "ps", "--format={{.Names}}\t{{.Image}}\t{{.Label \""+cke.CKELabelName+"\"}}").Output()
	if err != nil {
		return false, "", fmt.Errorf("failed to run docker ps: %w", err)
	}

	for line := range strings.SplitSeq(string(stdout), "\n") {
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 || fields[0] != name {
			continue
		}
		var label struct {
			Image string `json:"image"`
		}
		if fields[2] != "" {
			if err := json.Unmarshal([]byte(fields[2]), &label); err != nil {
				return false, "", fmt.Errorf("failed to parse the CKE label of %s: %w", name, err)
			}
		}
		if label.Image != "" {
			return true, label.Image, nil
		}
		return true, fields[1], nil
	}
	return false, "", nil
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
