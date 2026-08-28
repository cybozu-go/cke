# Makefile for cke

ETCD_VERSION = 3.6.11

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
test:
	go test -race -v ./...

.PHONY: lint
lint:
	go tool golangci-lint run

.PHONY: fmt
fmt:
	go tool golangci-lint fmt

.PHONY: install
install:
	go install ./pkg/...

.PHONY: static
static: goimports
	go generate ./static

.PHONY: goimports
goimports:
	if ! which goimports >/dev/null; then \
		env GOFLAGS= go install golang.org/x/tools/cmd/goimports@latest; \
	fi
