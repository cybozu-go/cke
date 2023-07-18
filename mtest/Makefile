# Makefile for multi-host testing using ginkgo

# tool versions
MANAGEMENT_ETCD_VERSION = 3.5.9
VAULT_VERSION = 1.14.0
K8S_VERSION = 1.26.6
CONTAINERD_VERSION = 1.7.2
CRITOOLS_VERSION = 1.27.1
CT_VERSION = 0.9.3

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
RUNTIME_ENDPOINT = unix:///var/run/k8s-containerd.sock
PLACEMAT = /usr/bin/placemat2
GINKGO = $(GOPATH)/bin/ginkgo --fail-fast -v
CURL = curl -fsL
PLACEMAT_DATADIR = /var/scratch/placemat
CT = /usr/local/bin/ct
SUDO = sudo
PACKAGES = libseccomp-dev automake autoconf libtool
SUITE = functions
CLUSTER := cke-cluster.yml

export HOST1 HOST2 NODE1 NODE2 NODE3 NODE4 NODE5 NODE6 PLACEMAT GINKGO SUITE

# non-configuration variables
MANAGEMENT_ETCD_ARCHIVE := etcd-v$(MANAGEMENT_ETCD_VERSION)-linux-amd64.tar.gz
VAULT_ARCHIVE := vault_$(VAULT_VERSION)_linux_amd64.zip
KUBECTL_URL := https://storage.googleapis.com/kubernetes-release/release/v$(K8S_VERSION)/bin/linux/amd64/kubectl
SSH_PRIVKEY = $(realpath ./mtest_key)
OUTPUT := ./output
DATA_DIR := $(abspath $(OUTPUT))/data
COREOS_IMAGE := flatcar_production_qemu_image.img
CKE_IMAGE := $(abspath $(OUTPUT))/cke.img
CKE_IMAGE_URL := quay.io/cybozu/cke:dev
CKECLUSTER := $(DATA_DIR)/cluster.yml
CKECONFIG := $(abspath $(OUTPUT))/cke.yml
KUBECTL := $(abspath $(OUTPUT))/kubectl

export SSH_PRIVKEY CKE_IMAGE CKE_IMAGE_URL CKECLUSTER CKECONFIG KUBECTL

CONTAINERD_FILES = \
	$(DATA_DIR)/containerd \
	$(DATA_DIR)/containerd-shim \
	$(DATA_DIR)/containerd-shim-runc-v1 \
	$(DATA_DIR)/containerd-shim-runc-v2 \
	$(DATA_DIR)/ctr

TEST_DEPS = \
	$(OUTPUT)/cke.img \
	$(OUTPUT)/cke.yml

PLACEMAT_DEPS = \
	$(CONTAINERD_FILES) \
	$(COREOS_IMAGE) \
	$(OUTPUT)/cluster.yml \
	$(OUTPUT)/host1.ign \
	$(OUTPUT)/host2.ign \
	$(OUTPUT)/kubectl \
	$(OUTPUT)/node1.ign \
	$(OUTPUT)/node2.ign \
	$(OUTPUT)/node3.ign \
	$(OUTPUT)/node4.ign \
	$(OUTPUT)/node5.ign \
	$(OUTPUT)/node6.ign \
	$(DATA_DIR)/cluster.yml \
	$(DATA_DIR)/crictl \
	$(DATA_DIR)/etcd \
	$(DATA_DIR)/etcdctl \
	$(DATA_DIR)/mtest_key \
	$(DATA_DIR)/setup-cke.sh \
	$(DATA_DIR)/vault

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
	$(CURL) -o $@.bz2 https://stable.release.flatcar-linux.net/amd64-usr/current/$@.bz2
	bunzip2 -q $@.bz2

$(DATA_DIR)/etcd $(DATA_DIR)/etcdctl: $(MANAGEMENT_ETCD_ARCHIVE)
	mkdir -p $(DATA_DIR)
	tar -C $(DATA_DIR) -x -z --wildcards --strip-components 1 -f $< '*/$(notdir $@)' --warning=no-unknown-keyword
	touch $@

$(DATA_DIR)/vault: $(VAULT_ARCHIVE)
	mkdir -p $(DATA_DIR)
	unzip $< -d $(DATA_DIR)
	touch $@

$(OUTPUT)/cke $(OUTPUT)/ckecli $(OUTPUT)/cke-localproxy: FORCE
	mkdir -p $(OUTPUT)
	cd ..; gofail enable op/etcd && \
		if CGO_ENABLED=0 GOBIN=$(realpath $(OUTPUT)) go install ./pkg/$(notdir $@); then \
			gofail disable op/etcd; \
		else \
			gofail disable op/etcd; \
			exit 1; \
		fi

$(OUTPUT)/cke.img: $(OUTPUT)/cke $(OUTPUT)/ckecli $(OUTPUT)/cke-localproxy
	cp ../LICENSE $(shell pwd)/../docker
	cp $(OUTPUT)/cke ../docker/
	cp $(OUTPUT)/ckecli ../docker/
	cp $(OUTPUT)/cke-localproxy ../docker/
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

$(DATA_DIR)/setup-cke.sh: setup-cke.sh
	mkdir -p $(DATA_DIR)
	cp $< $@
	chmod a+x $@

$(DATA_DIR)/mtest_key: $(SSH_PRIVKEY)
	mkdir -p $(DATA_DIR)
	cp $< $@
	chmod 644 $@

$(OUTPUT)/cluster.yml: cluster.yml
	mkdir -p $(OUTPUT)
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's#__DATA_DIR__#$(DATA_DIR)#' \
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
	$(CURL) -o ct https://github.com/flatcar/container-linux-config-transpiler/releases/download/v$(CT_VERSION)/ct-v$(CT_VERSION)-x86_64-unknown-linux-gnu
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

$(DATA_DIR)/cluster.yml: $(CLUSTER)
	mkdir -p $(DATA_DIR)
	sed -e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's|__RUNTIME_ENDPOINT__|$(RUNTIME_ENDPOINT)|' $< > $@
ifeq ($(SUITE),robustness)
	sed -i -e '/address: $(NODE5)/a\    user: cybozu\n  - address: 10.0.0.107' $@
endif

$(OUTPUT)/cke.yml: cke.yml
	mkdir -p $(OUTPUT)
	sed -e 's/__HOST1__/$(HOST1)/' $< > $@

unpack-containerd:
	mkdir -p $(DATA_DIR)
	$(CURL) https://github.com/containerd/containerd/releases/download/v$(CONTAINERD_VERSION)/containerd-$(CONTAINERD_VERSION)-linux-amd64.tar.gz | \
	tar xzf - --strip-components=1 -C $(DATA_DIR) bin/containerd bin/containerd-shim bin/containerd-shim-runc-v1 bin/containerd-shim-runc-v2 bin/ctr --warning=no-unknown-keyword

$(CONTAINERD_FILES): unpack-containerd

$(DATA_DIR)/crictl:
	mkdir -p $(DATA_DIR)
	$(CURL) -o crictl.tar.gz https://github.com/kubernetes-sigs/cri-tools/releases/download/v$(CRITOOLS_VERSION)/crictl-v$(CRITOOLS_VERSION)-linux-amd64.tar.gz
	tar zxvf crictl.tar.gz -C $(DATA_DIR) --warning=no-unknown-keyword

# Targets
test: $(TEST_DEPS)
	./test.sh

bootstrap: $(TEST_DEPS)
	./test.sh "can run Pods"

placemat: $(PLACEMAT_DEPS)
	sudo rm -rf $(PLACEMAT_DATADIR)
	sudo rm -rf $(VM_SHARED_DIR)
	sudo systemd-run --unit=placemat.service $(PLACEMAT) $(abspath output/cluster.yml)
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
	go install github.com/onsi/ginkgo/v2/ginkgo go.etcd.io/gofail
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends $(PACKAGES)

.PHONY: all test placemat stop clean setup unpack-containerd
