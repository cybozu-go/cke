# Makefile for cke

TESTBIN := $(CURDIR)/testbin

.PHONY: all
all: test

.PHONY: setup
setup: $(TESTBIN)/etcd

# Install etcd of the same version as go.etcd.io/etcd/server/v3 in go.mod.
$(TESTBIN)/etcd: go.mod
	mkdir -p $(TESTBIN)
	v="$$(go list -f '{{.Version}}' -m go.etcd.io/etcd/server/v3)"; \
	curl -fsL https://github.com/etcd-io/etcd/releases/download/$${v}/etcd-$${v}-linux-amd64.tar.gz | tar -xzf - --strip-components=1 -C $(TESTBIN) etcd-$${v}-linux-amd64/etcd etcd-$${v}-linux-amd64/etcdctl
	touch $@

.PHONY: check-generate
check-generate:
	# gqlgen needs additional dependencies that does not exist in go.mod.
	cd sabakan/mock; go run github.com/99designs/gqlgen@"$$(go list -f '{{.Version}}' -m github.com/99designs/gqlgen)" generate
	go mod tidy
	$(MAKE) static
	git diff --exit-code --name-only

.PHONY: test
test: setup
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

.PHONY: images
images:
	go run ./hack/update-images

.PHONY: static
static:
	go generate ./static
