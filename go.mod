module github.com/cybozu-go/cke

replace (
	labix.org/v2/mgo => github.com/globalsign/mgo v0.0.0-20180615134936-113d3961e731
	launchpad.net/gocheck => github.com/go-check/check v0.0.0-20180628173108-788fd7840127
)

require (
	github.com/99designs/gqlgen v0.9.3
	github.com/containernetworking/cni v0.6.0
	github.com/coreos/etcd v3.3.18+incompatible
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/cybozu-go/etcdutil v1.3.4
	github.com/cybozu-go/log v1.5.0
	github.com/cybozu-go/netutil v1.2.0
	github.com/cybozu-go/sabakan/v2 v2.5.0
	github.com/cybozu-go/well v1.8.1
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/etcd-io/gofail v0.0.0-20180808172546-51ce9a71510a
	github.com/google/go-cmp v0.3.0
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/hashicorp/vault/api v1.0.4
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910
	github.com/prometheus/common v0.0.0-20181126121408-4724e9255275
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/vektah/gqlparser v1.1.2
	golang.org/x/crypto v0.0.0-20190820162420-60c769a6c586
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	google.golang.org/genproto v0.0.0-20190502173448-54afdca5d873 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gotest.tools v2.2.0+incompatible // indirect
	k8s.io/api v0.16.5
	k8s.io/apimachinery v0.16.5
	k8s.io/client-go v0.16.5
	sigs.k8s.io/yaml v1.1.0
)

go 1.13
