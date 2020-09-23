# Makefile for multi-host testing using ginkgo

# configuration variables
BRIDGE_ADDRESS = 10.0.0.1
## 'HOST' runs CKE and management etcd
HOST1 = 10.0.0.11
HOST2 = 10.0.0.12
## 'NODE' runs etcd and k8s, managed by CKE
NODE1 = 10.0.0.101
NODE2 = 10.0.0.102
NODE3 = 10.0.0.103
NODE4 = 10.0.0.104
NODE5 = 10.0.0.105
NODE6 = 10.0.0.106
CONTAINER_RUNTIME = remote
ifeq ($(CONTAINER_RUNTIME),remote)
RUNTIME_ENDPOINT = unix:///var/run/k8s-containerd.sock
endif
PLACEMAT = /usr/bin/placemat
GINKGO = env GOFLAGS=-mod=vendor $(GOPATH)/bin/ginkgo --failFast -v
CURL = curl -fsL
MANAGEMENT_ETCD_VERSION = 3.3.25
VAULT_VERSION = 1.2.2
K8S_VERSION = 1.18.9
PLACEMAT_DATADIR = /var/scratch/placemat
CONTAINERD_VERSION = 1.4.1
CRITOOLS_VERSION = 1.18.0
CT_VERSION = 0.9.0
CT = /usr/local/bin/ct
SUDO = sudo
PACKAGES = libseccomp-dev automake autoconf libtool
SUITE = ./functions
ifeq ($(findstring /,$(SUITE)),)
  SUITE_PACKAGE = ./$(SUITE)
else
  SUITE_PACKAGE = $(SUITE)
endif
CLUSTER := cke-cluster.yml

export HOST1 HOST2 NODE1 NODE2 NODE3 NODE4 NODE5 NODE6 PLACEMAT GINKGO CONTAINER_RUNTIME SUITE_PACKAGE

# non-configuration variables
MANAGEMENT_ETCD_ARCHIVE := etcd-v$(MANAGEMENT_ETCD_VERSION)-linux-amd64.tar.gz
VAULT_ARCHIVE := vault_$(VAULT_VERSION)_linux_amd64.zip
KUBECTL_URL := https://storage.googleapis.com/kubernetes-release/release/v$(K8S_VERSION)/bin/linux/amd64/kubectl
SSH_PRIVKEY = $(realpath ./mtest_key)
OUTPUT := ./output
COREOS_IMAGE := coreos_production_qemu_image.img
CKE_IMAGE := $(abspath $(OUTPUT))/cke.img
CKE_IMAGE_URL := quay.io/cybozu/cke:dev
CKECLUSTER := $(abspath $(OUTPUT))/cke-cluster.yml
CKECONFIG := $(abspath $(OUTPUT))/cke.yml
ETCDCTL := $(abspath $(OUTPUT))/etcdctl
KUBECTL := $(abspath $(OUTPUT))/kubectl
LOCALPVYAML := $(realpath ./local-pv.yml)
NGINXYAML := $(realpath ./nginx.yml)
POLICYYAML := $(realpath ./mtest-policy.yml)

export SSH_PRIVKEY CKE_IMAGE CKE_IMAGE_URL CKECLUSTER CKECONFIG KUBECTL ETCDCTL POLICYYAML NGINXYAML LOCALPVYAML

CONTAINERD_FILES = \
	$(OUTPUT)/containerd \
	$(OUTPUT)/containerd-shim \
	$(OUTPUT)/containerd-shim-runc-v1 \
	$(OUTPUT)/containerd-shim-runc-v2 \
	$(OUTPUT)/ctr

TEST_DEPS = \
	$(OUTPUT)/cke.img \
	$(OUTPUT)/cke.yml

PLACEMAT_DEPS = \
	$(CONTAINERD_FILES) \
	$(COREOS_IMAGE) \
	$(ETCD_BACKUP) \
	$(OUTPUT)/cke-cluster.yml \
	$(OUTPUT)/cluster.yml \
	$(OUTPUT)/crictl \
	$(OUTPUT)/etcd \
	$(OUTPUT)/etcdctl \
	$(OUTPUT)/host1.ign \
	$(OUTPUT)/host2.ign \
	$(OUTPUT)/kubectl \
	$(OUTPUT)/mtest_key \
	$(OUTPUT)/node1.ign \
	$(OUTPUT)/node2.ign \
	$(OUTPUT)/node3.ign \
	$(OUTPUT)/node4.ign \
	$(OUTPUT)/node5.ign \
	$(OUTPUT)/node6.ign \
	$(OUTPUT)/setup-cke.sh \
	$(OUTPUT)/vault

all:
	@echo "Specify one of these targets:"
	@echo
	@echo "    placemat  - run VMs in placemat."
	@echo "    stop      - stop placemat."
	@echo "    test      - run test in placemat env."
	@echo "    bootstrap - create k8s cluster by CKE but not run all test."
	@echo "    setup     - install dependencies."
	@echo
	@echo "'make test' accepts SUITE."

$(MANAGEMENT_ETCD_ARCHIVE):
	$(CURL) -o $@ https://github.com/coreos/etcd/releases/download/v$(MANAGEMENT_ETCD_VERSION)/etcd-v$(MANAGEMENT_ETCD_VERSION)-linux-amd64.tar.gz

$(VAULT_ARCHIVE):
	$(CURL) -o $@ https://releases.hashicorp.com/vault/$(VAULT_VERSION)/$(VAULT_ARCHIVE)

$(COREOS_IMAGE):
	$(CURL) -o $@.bz2 https://stable.release.core-os.net/amd64-usr/current/$@.bz2
	bunzip2 -q $@.bz2

$(OUTPUT)/etcd $(OUTPUT)/etcdctl: $(MANAGEMENT_ETCD_ARCHIVE)
	mkdir -p $(OUTPUT)
	tar -C $(OUTPUT) -x -z --wildcards --strip-components 1 -f $< '*/$(notdir $@)' --warning=no-unknown-keyword
	touch $@

$(OUTPUT)/vault: $(VAULT_ARCHIVE)
	mkdir -p $(OUTPUT)
	unzip $< -d $(OUTPUT)
	touch $@

$(OUTPUT)/cke $(OUTPUT)/ckecli: FORCE
	mkdir -p $(OUTPUT)
	cd ..; gofail enable op/etcd && \
		if GOBIN=$(realpath $(OUTPUT)) go install -mod=vendor ./pkg/$(notdir $@); then \
			gofail disable op/etcd; \
		else \
			gofail disable op/etcd; \
			exit 1; \
		fi

$(OUTPUT)/cke.img: $(OUTPUT)/cke $(OUTPUT)/ckecli
	cp ../LICENSE $(shell pwd)/../docker
	cp $(OUTPUT)/cke ../docker/
	cp $(OUTPUT)/ckecli ../docker/
	sudo docker build --no-cache --rm=false -t $(CKE_IMAGE_URL) ../docker/
	mkdir -p $(OUTPUT)
	rm -f $@
	sudo docker save -o $@ $(CKE_IMAGE_URL)
	sudo chmod 644 $@

FORCE:

$(OUTPUT)/kubectl:
	mkdir -p $(OUTPUT)
	curl -o $@ -Lf $(KUBECTL_URL)
	chmod a+x $@

$(OUTPUT)/setup-cke.sh: setup-cke.sh
	mkdir -p $(OUTPUT)
	cp $< $@
	chmod a+x $@

$(OUTPUT)/mtest_key: $(SSH_PRIVKEY)
	mkdir -p $(OUTPUT)
	cp $< $@

$(OUTPUT)/cluster.yml: cluster.yml
	mkdir -p $(OUTPUT)
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__COREOS_IMAGE__/$(COREOS_IMAGE)/' $< > $@

$(OUTPUT)/host1.ign: host.ign
	mkdir -p $(OUTPUT)
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/host1/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__HOST1__/$(HOST1)/' \
		-e 's/__NODE_ADDRESS__/$(HOST1)/' $< > $@

$(OUTPUT)/host2.ign: host.ign
	mkdir -p $(OUTPUT)
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/host2/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__HOST1__/$(HOST1)/' \
		-e 's/__NODE_ADDRESS__/$(HOST2)/' $< > $@

$(CT):
	$(CURL) -o ct https://github.com/coreos/container-linux-config-transpiler/releases/download/v$(CT_VERSION)/ct-v$(CT_VERSION)-x86_64-unknown-linux-gnu
	chmod +x ct
	$(SUDO) mv ct $@

$(OUTPUT)/node.ign: node-ign.yml $(CT)
	mkdir -p $(OUTPUT)
	$(CT) -in-file=$< -pretty -out-file=$@

$(OUTPUT)/node1.ign: $(OUTPUT)/node.ign
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/node1/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__NODE_ADDRESS__/$(NODE1)/' $< > $@

$(OUTPUT)/node2.ign: $(OUTPUT)/node.ign
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/node2/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__NODE_ADDRESS__/$(NODE2)/' $< > $@

$(OUTPUT)/node3.ign: $(OUTPUT)/node.ign
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/node3/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__NODE_ADDRESS__/$(NODE3)/' $< > $@

$(OUTPUT)/node4.ign: $(OUTPUT)/node.ign
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/node4/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__NODE_ADDRESS__/$(NODE4)/' $< > $@

$(OUTPUT)/node5.ign: $(OUTPUT)/node.ign
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/node5/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__NODE_ADDRESS__/$(NODE5)/' $< > $@

$(OUTPUT)/node6.ign: $(OUTPUT)/node.ign
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/node6/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__NODE_ADDRESS__/$(NODE6)/' $< > $@

$(OUTPUT)/cke-cluster.yml: $(CLUSTER)
	mkdir -p $(OUTPUT)
	sed -e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__RUNTIME__/$(CONTAINER_RUNTIME)/' \
		-e 's|__RUNTIME_ENDPOINT__|$(RUNTIME_ENDPOINT)|' $< > $@
ifeq ($(notdir $(SUITE_PACKAGE)),robustness)
	sed -i -e '/address: $(NODE5)/a\    user: cybozu\n  - address: 10.0.0.107' $@
endif

$(OUTPUT)/cke.yml: cke.yml
	mkdir -p $(OUTPUT)
	sed -e 's/__HOST1__/$(HOST1)/' $< > $@

unpack-containerd:
	mkdir -p $(OUTPUT)
	$(CURL) https://github.com/containerd/containerd/releases/download/v$(CONTAINERD_VERSION)/containerd-$(CONTAINERD_VERSION)-linux-amd64.tar.gz | \
	tar xzf - --strip-components=1 -C $(OUTPUT) bin/containerd bin/containerd-shim bin/containerd-shim-runc-v1 bin/containerd-shim-runc-v2 bin/ctr --warning=no-unknown-keyword

$(CONTAINERD_FILES): unpack-containerd

$(OUTPUT)/crictl:
	mkdir -p $(OUTPUT)
	$(CURL) -o crictl.tar.gz https://github.com/kubernetes-sigs/cri-tools/releases/download/v$(CRITOOLS_VERSION)/crictl-v$(CRITOOLS_VERSION)-linux-amd64.tar.gz
	tar zxvf crictl.tar.gz -C $(OUTPUT) --warning=no-unknown-keyword

# Targets
test: $(TEST_DEPS)
	./test.sh "$(TARGET)"

bootstrap: $(TEST_DEPS)
	SUITE_PACKAGE=./functions ./test.sh "can run Pods"

placemat: $(PLACEMAT_DEPS)
	sudo rm -rf $(PLACEMAT_DATADIR)
	sudo rm -rf $(VM_SHARED_DIR)
	sudo systemd-run --unit=placemat.service $(PLACEMAT) -enable-virtfs $(abspath output/cluster.yml)
	@echo 'Placemat is running.  To kill it, do "make stop".'

stop:
	sudo systemctl stop placemat.service
	sudo systemctl disable placemat.service
	sudo rm /run/systemd/transient/placemat.service
	sudo systemctl daemon-reload
	sudo systemctl reset-failed placemat.service

clean:
	rm -rf $(OUTPUT)

setup:
	go install -mod=vendor github.com/onsi/ginkgo/ginkgo github.com/etcd-io/gofail
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends $(PACKAGES)

.PHONY: all test placemat stop clean setup unpack-containerd
