package localproxy

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op/k8s"
)

func decideOps(c *cke.Cluster, currentAP string, st *status) (newAP string, ops []cke.Operator) {
	if len(st.apiServers) == 0 {
		return
	}

	newAP = st.apiServers[0]
	if slices.Contains(st.apiServers, currentAP) {
		newAP = currentAP
	}

	apURL := fmt.Sprintf("https://%s:6443", newAP)

	if !st.proxyRunning {
		ops = append(ops, k8s.KubeProxyBootOp(ckeNodes, c.Name, apURL, c.Options.Proxy))
	} else {
		if newAP != currentAP || st.proxyImage != cke.KubernetesImage.Name() {
			ops = append(ops, k8s.KubeProxyRestartOp(ckeNodes, c.Name, apURL, c.Options.Proxy))
		}
	}

	if !st.unboundRunning {
		ops = append(ops, &unboundBootOp{conf: st.desiredUnboundConf})
		return
	}

	if !bytes.Equal(st.unboundConf, st.desiredUnboundConf) || st.unboundImage != cke.UnboundImage.Name() {
		ops = append(ops, &unboundRestartOp{conf: st.desiredUnboundConf})
	}

	return
}
