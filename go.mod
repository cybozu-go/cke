module github.com/cybozu-go/cke

replace (
	labix.org/v2/mgo => github.com/globalsign/mgo v0.0.0-20180615134936-113d3961e731
	launchpad.net/gocheck => github.com/go-check/check v0.0.0-20180628173108-788fd7840127
)

require (
	github.com/99designs/gqlgen v0.9.3
	github.com/agnivade/levenshtein v1.0.2 // indirect
	github.com/containernetworking/cni v0.8.0
	github.com/coreos/etcd v3.3.25+incompatible
	github.com/cybozu-go/etcdutil v1.3.5
	github.com/cybozu-go/log v1.5.0
	github.com/cybozu-go/netutil v1.2.0
	github.com/cybozu-go/well v1.10.0
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/etcd-io/gofail v0.0.0-20190801230047-ad7f989257ca
	github.com/evanphx/json-patch v4.2.0+incompatible // indirect
	github.com/google/go-cmp v0.5.2
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/hashicorp/vault/api v1.0.5-0.20200317185738-82f498082f02
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.10.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/vektah/gqlparser v1.1.2
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/text v0.3.3 // indirect
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/apiserver v0.18.8
	k8s.io/client-go v0.18.8
	k8s.io/component-base v0.18.8
	k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29 // indirect
	k8s.io/kube-scheduler v0.18.8
	k8s.io/kubelet v0.18.8
	sigs.k8s.io/yaml v1.2.0
)

go 1.13
