package op

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
	vault "github.com/hashicorp/vault/api"
	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type fakeInfrastructure struct {
	cs kubernetes.Interface
}

func (f *fakeInfrastructure) Close()                              {}
func (f *fakeInfrastructure) Agent(_ string) cke.Agent            { panic("not implemented") }
func (f *fakeInfrastructure) Engine(_ string) cke.ContainerEngine { panic("not implemented") }
func (f *fakeInfrastructure) Vault() (*vault.Client, error)       { panic("not implemented") }
func (f *fakeInfrastructure) Storage() cke.Storage                { panic("not implemented") }
func (f *fakeInfrastructure) NewEtcdClient(_ context.Context, _ []string) (*clientv3.Client, error) {
	panic("not implemented")
}
func (f *fakeInfrastructure) K8sConfig(_ context.Context, _ *cke.Node) (*rest.Config, error) {
	panic("not implemented")
}
func (f *fakeInfrastructure) K8sClient(_ context.Context, _ *cke.Node) (kubernetes.Interface, error) {
	return f.cs, nil
}
func (f *fakeInfrastructure) HTTPClient() *well.HTTPClient { panic("not implemented") }
func (f *fakeInfrastructure) HTTPSClient(_ context.Context) (*well.HTTPClient, error) {
	panic("not implemented")
}
func (f *fakeInfrastructure) ReleaseAgent(_ string) {}
