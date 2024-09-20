# Makefile for cke

ETCD_VERSION = 3.5.15

.PHONY: all
all: test

.PHONY: setup
setup:
	curl -fsL https://github.com/etcd-io/etcd/releases/download/v$(ETCD_VERSION)/etcd-v$(ETCD_VERSION)-linux-amd64.tar.gz | sudo tar -xzf - --strip-components=1 -C /usr/local/bin etcd-v$(ETCD_VERSION)-linux-amd64/etcd etcd-v$(ETCD_VERSION)-linux-amd64/etcdctl

.PHONY: check-generate
check-generate:
	# gqlgen needs additional dependencies that does not exist in go.mod.
	cd sabakan/mock; go run github.com/99designs/gqlgen@"$$(go list -f '{{.Version}}' -m github.com/99designs/gqlgen)" generate
	go mod tidy
	$(MAKE) static
	git diff --exit-code --name-only

.PHONY: test
test: test-tools
	test -z "$$(gofmt -s -l . | tee /dev/stderr)"
	staticcheck ./...
	# temporarily disable nilerr due to a false positive
	# https://github.com/cybozu-go/cke/runs/4298557316?check_suite_focus=true
	#test -z "$$(nilerr ./... 2>&1 | tee /dev/stderr)"
	test -z "$$(custom-checker -restrictpkg.packages=html/template,log ./... 2>&1 | tee /dev/stderr)"
	go vet ./...
	go test -race -v ./...

.PHONY: install
install:
	go install ./pkg/...

.PHONY: static
static: goimports
	go generate ./static

.PHONY: test-tools
test-tools: staticcheck nilerr goimports custom-checker

.PHONY: staticcheck
staticcheck:
	if ! which staticcheck >/dev/null; then \
		env GOFLAGS= go install honnef.co/go/tools/cmd/staticcheck@latest; \
	fi

.PHONY: nilerr
nilerr:
	if ! which nilerr >/dev/null; then \
		env GOFLAGS= go install github.com/gostaticanalysis/nilerr/cmd/nilerr@latest; \
	fi

.PHONY: goimports
goimports:
	if ! which goimports >/dev/null; then \
		env GOFLAGS= go install golang.org/x/tools/cmd/goimports@latest; \
	fi

.PHONY: custom-checker
custom-checker:
	if ! which custom-checker >/dev/null; then \
		env GOFLAGS= go install github.com/cybozu-go/golang-custom-analyzer/cmd/custom-checker@latest; \
	fi
