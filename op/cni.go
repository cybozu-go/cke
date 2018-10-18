package op

import "fmt"

const (
	cniBinDir  = "/opt/cni/bin"
	cniConfDir = "/etc/cni/net.d"
	cniVarDir  = "/var/lib/cni"
)

func cniBridgeConfig(podSubnet string) string {
	return fmt.Sprintf(`{
    "cniVersion": "0.3.1",
    "name": "bridge",
    "type": "bridge",
    "bridge": "cnio0",
    "isGateway": true,
    "ipMasq": true,
    "ipam": {
        "type": "host-local",
        "ranges": [
          [{"subnet": "%s"}]
        ],
        "routes": [{"dst": "0.0.0.0/0"}]
    }
}
`, podSubnet)
}
