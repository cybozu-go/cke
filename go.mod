module github.com/cybozu-go/cke

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	github.com/99designs/gqlgen v0.9.3
	github.com/agnivade/levenshtein v1.0.2 // indirect
	github.com/containernetworking/cni v0.8.0
	github.com/coreos/etcd v3.3.25+incompatible
	github.com/cybozu-go/etcdutil v1.3.5
	github.com/cybozu-go/log v1.5.0
	github.com/cybozu-go/netutil v1.2.0
	github.com/cybozu-go/well v1.10.0
	github.com/etcd-io/gofail v0.0.0-20190801230047-ad7f989257ca
	github.com/google/go-cmp v0.5.2
	github.com/hashicorp/vault/api v1.0.5-0.20200317185738-82f498082f02
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.10.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/vektah/gqlparser v1.1.2
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/apiserver v0.19.7
	k8s.io/client-go v0.19.7
	k8s.io/kube-scheduler v0.19.7
	k8s.io/kubelet v0.19.7
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/yaml v1.2.0
)

go 1.13
