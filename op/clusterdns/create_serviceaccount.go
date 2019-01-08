package clusterdns

import "github.com/cybozu-go/cke"

type createServiceAccountOp struct {
	apiserver *cke.Node
	finished  bool
}
