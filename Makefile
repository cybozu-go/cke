# Makefile for cke

GOFLAGS = -mod=vendor
export GOFLAGS
ETCD_VERSION = 3.3.25

.PHONY: all
all: test

.PHONY: setup
setup:
	curl -fsL https://github.com/etcd-io/etcd/releases/download/v$(ETCD_VERSION)/etcd-v$(ETCD_VERSION)-linux-amd64.tar.gz | sudo tar -xzf - --strip-components=1 -C /usr/local/bin etcd-v$(ETCD_VERSION)-linux-amd64/etcd etcd-v$(ETCD_VERSION)-linux-amd64/etcdctl

.PHONY: test
test: test-tools
	test -z "$$(gofmt -s -l . | grep -v '^vendor' | tee /dev/stderr)"
	staticcheck ./...
	test -z "$$(nilerr ./... 2>&1 | tee /dev/stderr)"
	test -z "$$(custom-checker -restrictpkg.packages=html/template,log ./... 2>&1 | tee /dev/stderr)"
	go install ./pkg/...
	go test -race -v ./...
	go vet ./...

.PHONY: static
static:
	go generate ./static
	git add ./static/resources.go

.PHONY: mod
mod:
	go mod tidy
	go mod vendor
	git add -f vendor
	git add go.mod

.PHONY: test-tools
test-tools: staticcheck

.PHONY: staticcheck nilerr
staticcheck:
	if ! which staticcheck >/dev/null; then \
		cd /tmp; env GOFLAGS= GO111MODULE=on go get honnef.co/go/tools/cmd/staticcheck; \
	fi

.PHONY: nilerr
nilerr:
	if ! which nilerr >/dev/null; then \
		cd /tmp; env GOFLAGS= GO111MODULE=on go get github.com/gostaticanalysis/nilerr/cmd/nilerr; \
	fi
