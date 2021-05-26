module github.com/cybozu-go/cke

go 1.16

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	github.com/99designs/gqlgen v0.13.0
	github.com/containernetworking/cni v0.8.1
	github.com/cybozu-go/etcdutil v1.4.0
	github.com/cybozu-go/log v1.6.0
	github.com/cybozu-go/netutil v1.4.1
	github.com/cybozu-go/well v1.8.1
	github.com/etcd-io/gofail v0.0.0-20190801230047-ad7f989257ca
	github.com/google/go-cmp v0.5.5
	github.com/hashicorp/vault/api v1.1.0
	github.com/howeyc/gopass v0.0.0-20190910152052-7cb4b85ec19c
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.21.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/vektah/gqlparser/v2 v2.2.0
	go.etcd.io/etcd v0.5.0-alpha.5.0.20210512015243-d19fbe541bf9
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/apiserver v0.20.6
	k8s.io/client-go v0.20.6
	k8s.io/kube-proxy v0.20.6
	k8s.io/kube-scheduler v0.20.6
	k8s.io/kubelet v0.20.6
	k8s.io/utils v0.0.0-20210305010621-2afb4311ab10
	sigs.k8s.io/yaml v1.2.0
)
