package cke

import "fmt"

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
