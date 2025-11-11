package op

import (
	"context"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op/common"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kubeEndpointSliceCreateOp struct {
	apiserver     *cke.Node
	endpointslice *discoveryv1.EndpointSlice
	step          int
}

var (
	// Wait time for endpoint changes to propagate to each node.
	waitTimeEndpointChangePropagate = 300 * time.Millisecond
)

// KubeEndpointSliceCreateOp returns an Operator to create EndpointSlice resource.
func KubeEndpointSliceCreateOp(apiserver *cke.Node, eps *discoveryv1.EndpointSlice) cke.Operator {
	return &kubeEndpointSliceCreateOp{
		apiserver:     apiserver,
		endpointslice: eps,
	}
}

func (o *kubeEndpointSliceCreateOp) Name() string {
	return "create-endpointslice"
}

func (o *kubeEndpointSliceCreateOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return createEndpointSliceCommand{o.apiserver, o.endpointslice}
	case 1:
		o.step++
		return common.WaitCommand(waitTimeEndpointChangePropagate)
	default:
		return nil
	}
}

func (o *kubeEndpointSliceCreateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type kubeEndpointSliceUpdateOp struct {
	apiserver     *cke.Node
	endpointslice *discoveryv1.EndpointSlice
	step          int
}

// KubeEndpointSliceUpdateOp returns an Operator to update Endpoints resource.
func KubeEndpointSliceUpdateOp(apiserver *cke.Node, eps *discoveryv1.EndpointSlice) cke.Operator {
	return &kubeEndpointSliceUpdateOp{
		apiserver:     apiserver,
		endpointslice: eps,
	}
}

func (o *kubeEndpointSliceUpdateOp) Name() string {
	return "update-endpointslice"
}

func (o *kubeEndpointSliceUpdateOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return updateEndpointSliceCommand{o.apiserver, o.endpointslice}
	case 1:
		o.step++
		return common.WaitCommand(waitTimeEndpointChangePropagate)
	default:
		return nil
	}
}

func (o *kubeEndpointSliceUpdateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type createEndpointSliceCommand struct {
	apiserver     *cke.Node
	endpointslice *discoveryv1.EndpointSlice
}

func (c createEndpointSliceCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	_, err = cs.DiscoveryV1().EndpointSlices(c.endpointslice.Namespace).Create(ctx, c.endpointslice, metav1.CreateOptions{})

	return err
}

func (c createEndpointSliceCommand) Command() cke.Command {
	var addresses []string
	for _, e := range c.endpointslice.Endpoints {
		addresses = append(addresses, e.Addresses...)
	}
	return cke.Command{
		Name:   "createEndpointSliceCommand",
		Target: strings.Join(addresses, ","),
	}
}

type updateEndpointSliceCommand struct {
	apiserver     *cke.Node
	endpointslice *discoveryv1.EndpointSlice
}

func (c updateEndpointSliceCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	_, err = cs.DiscoveryV1().EndpointSlices(c.endpointslice.Namespace).Update(ctx, c.endpointslice, metav1.UpdateOptions{})

	return err
}

func (c updateEndpointSliceCommand) Command() cke.Command {
	var addresses []string
	for _, e := range c.endpointslice.Endpoints {
		addresses = append(addresses, e.Addresses...)
	}
	return cke.Command{
		Name:   "updateEndpointSliceCommand",
		Target: strings.Join(addresses, ","),
	}
}
