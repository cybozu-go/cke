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
RUNTIME = remote
ifeq ($(RUNTIME),remote)
RUNTIME_ENDPOINT = unix:///var/run/k8s-containerd.sock
endif
PLACEMAT = /usr/bin/placemat
GINKGO = env GOFLAGS=-mod=vendor $(GOPATH)/bin/ginkgo
MANAGEMENT_ETCD_VERSION = 3.3.11
VAULT_VERSION = 1.0.2
K8S_VERSION = 1.13.2
PLACEMAT_DATADIR = /var/scratch/placemat
CONTAINERD_VERSION = 1.2.4
PROTOC_VERSION = 3.5.0
CRITOOLS_VERSION = 1.13.0
SUDO = sudo
PACKAGES = libseccomp-dev
CT = /usr/local/bin/ct

export HOST1 HOST2 NODE1 NODE2 NODE3 NODE4 NODE5 NODE6 PLACEMAT GINKGO

# non-configuration variables
MANAGEMENT_ETCD_ARCHIVE := etcd-v$(MANAGEMENT_ETCD_VERSION)-linux-amd64.tar.gz
VAULT_ARCHIVE := vault_$(VAULT_VERSION)_linux_amd64.zip
KUBECTL_URL := https://storage.googleapis.com/kubernetes-release/release/v$(K8S_VERSION)/bin/linux/amd64/kubectl
SSH_PRIVKEY := ./mtest_key
OUTPUT := ./output
COREOS_IMAGE := coreos_production_qemu_image.img
CKECLI := $(OUTPUT)/ckecli
KUBECTL := $(OUTPUT)/kubectl
CKECLUSTER := $(OUTPUT)/cke-cluster.yml
CKECONFIG := $(OUTPUT)/cke.yml
DOWNLOADER := download-disk-if-exists

export SSH_PRIVKEY CKECLI CKECLUSTER CKECONFIG KUBECTL

GENERATED_FILES = $(OUTPUT)/etcd  $(OUTPUT)/etcdctl \
	$(OUTPUT)/vault \
	$(OUTPUT)/cke $(OUTPUT)/ckecli \
	$(OUTPUT)/kubectl \
	$(OUTPUT)/cluster.yml \
	$(OUTPUT)/host1.ign \
	$(OUTPUT)/host2.ign \
	$(OUTPUT)/node1.ign \
	$(OUTPUT)/node2.ign \
	$(OUTPUT)/node3.ign \
	$(OUTPUT)/node4.ign \
	$(OUTPUT)/node5.ign \
	$(OUTPUT)/node6.ign \
	$(OUTPUT)/cke-cluster.yml \
	$(OUTPUT)/cke.yml \
	$(OUTPUT)/setup-cke.sh \
	$(OUTPUT)/mtest_key \
	$(ETCD_BACKUP) \
	$(OUTPUT)/setup-local-pv.sh \
	$(COREOS_IMAGE) \
	$(GOPATH)/bin/$(DOWNLOADER) \
	$(CONTAINERD_FILES) \
	$(OUTPUT)/crictl

CONTAINERD_FILES = \
	$(OUTPUT)/containerd \
	$(OUTPUT)/containerd-shim \
	$(OUTPUT)/containerd-shim-runc-v1 \
	$(OUTPUT)/ctr

all: test

$(MANAGEMENT_ETCD_ARCHIVE):
	curl -sSLf -o $@ https://github.com/coreos/etcd/releases/download/v$(MANAGEMENT_ETCD_VERSION)/etcd-v$(MANAGEMENT_ETCD_VERSION)-linux-amd64.tar.gz

$(VAULT_ARCHIVE):
	curl -sSLf -o $@ https://releases.hashicorp.com/vault/$(VAULT_VERSION)/$(VAULT_ARCHIVE)

$(COREOS_IMAGE):
	curl -sSLf -o $@.bz2 https://stable.release.core-os.net/amd64-usr/current/$@.bz2
	bunzip2 -q $@.bz2

$(GOPATH)/bin/$(DOWNLOADER): ../images.go
	go install -mod=vendor ./$(DOWNLOADER)/...

$(OUTPUT)/etcd $(OUTPUT)/etcdctl: $(MANAGEMENT_ETCD_ARCHIVE)
	mkdir -p $(OUTPUT)
	tar -C $(OUTPUT) -x -z --wildcards --strip-components 1 -f $< '*/$(notdir $@)'
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

$(OUTPUT)/setup-local-pv.sh: setup-local-pv.sh
	mkdir -p $(OUTPUT)
	cp $< $@
	chmod a+x $@

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

node.ign: node-ign.yml
	$(CT) -in-file=$< -pretty -out-file=$@

$(OUTPUT)/node1.ign: node.ign
	mkdir -p $(OUTPUT)
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/node1/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__NODE_ADDRESS__/$(NODE1)/' $< > $@

$(OUTPUT)/node2.ign: node.ign
	mkdir -p $(OUTPUT)
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/node2/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__NODE_ADDRESS__/$(NODE2)/' $< > $@

$(OUTPUT)/node3.ign: node.ign
	mkdir -p $(OUTPUT)
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/node3/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__NODE_ADDRESS__/$(NODE3)/' $< > $@

$(OUTPUT)/node4.ign: node.ign
	mkdir -p $(OUTPUT)
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/node4/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__NODE_ADDRESS__/$(NODE4)/' $< > $@

$(OUTPUT)/node5.ign: node.ign
	mkdir -p $(OUTPUT)
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/node5/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__NODE_ADDRESS__/$(NODE5)/' $< > $@

$(OUTPUT)/node6.ign: node.ign
	mkdir -p $(OUTPUT)
	sed -e 's/__BRIDGE_ADDRESS__/$(BRIDGE_ADDRESS)/' \
		-e 's/__NODE_NAME__/node6/' \
		-e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__NODE_ADDRESS__/$(NODE6)/' $< > $@

$(OUTPUT)/cke-cluster.yml: cke-cluster.yml
	mkdir -p $(OUTPUT)
	sed -e 's/__NODE1__/$(NODE1)/' \
		-e 's/__NODE2__/$(NODE2)/' \
		-e 's/__NODE3__/$(NODE3)/' \
		-e 's/__NODE4__/$(NODE4)/' \
		-e 's/__NODE5__/$(NODE5)/' \
		-e 's/__NODE6__/$(NODE6)/' \
		-e 's/__RUNTIME__/$(RUNTIME)/' \
		-e 's|__RUNTIME_ENDPOINT__|$(RUNTIME_ENDPOINT)|' $< > $@

$(OUTPUT)/cke.yml: cke.yml
	mkdir -p $(OUTPUT)
	sed -e 's/__HOST1__/$(HOST1)/' $< > $@

build-containerd:
	rm -rf $(GOPATH)/src/github.com/containerd/containerd
	mkdir -p $(GOPATH)/src/github.com/containerd/containerd
	curl -fsSL -o containerd.tar.gz https://github.com/containerd/containerd/archive/v${CONTAINERD_VERSION}.tar.gz
	tar -x -z --strip-components 1 -f containerd.tar.gz -C $(GOPATH)/src/github.com/containerd/containerd
	rm -f containerd.tar.gz
	(cd $(GOPATH)/src/github.com/containerd/containerd; make VERSION=$(CONTAINERD_VERSION))

$(OUTPUT)/containerd: build-containerd
	mkdir -p $(OUTPUT)
	cp $(GOPATH)/src/github.com/containerd/containerd/bin/containerd $@

$(OUTPUT)/containerd-shim: build-containerd
	mkdir -p $(OUTPUT)
	cp $(GOPATH)/src/github.com/containerd/containerd/bin/containerd-shim $@

$(OUTPUT)/containerd-shim-runc-v1: build-containerd
	mkdir -p $(OUTPUT)
	cp $(GOPATH)/src/github.com/containerd/containerd/bin/containerd-shim-runc-v1 $@

$(OUTPUT)/ctr: build-containerd
	mkdir -p $(OUTPUT)
	cp $(GOPATH)/src/github.com/containerd/containerd/bin/ctr $@

$(OUTPUT)/crictl:
	mkdir -p $(OUTPUT)
	curl -fsSL -o crictl.tar.gz https://github.com/kubernetes-sigs/cri-tools/releases/download/v$(CRITOOLS_VERSION)/crictl-v$(CRITOOLS_VERSION)-linux-amd64.tar.gz
	tar zxvf crictl.tar.gz -C $(OUTPUT)

test: $(GENERATED_FILES)
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/host1
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/host2
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node1
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node2
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node3
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node4
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node5
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node6
	$(DOWNLOADER)
	./test.sh

placemat: $(GENERATED_FILES)
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/host1
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/host2
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node1
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node2
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node3
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node4
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node5
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node6
	$(DOWNLOADER)
	sudo -b $(PLACEMAT) output/cluster.yml
	@echo 'Placemat is running.  To kill it, do "sudo pkill placemat".'

test-light: $(OUTPUT)/cke
	for h in host-1 host-2; do \
		./mssh $$h sudo systemctl reset-failed cke.service; \
		./mssh $$h sudo systemctl stop cke.service; \
		cat $(OUTPUT)/cke | ./mssh $$h sudo dd of=/opt/bin/cke status=none; \
		./mssh $$h sudo chmod 755 /opt/bin/cke; \
	done
	$(GINKGO) -v -focus="${TARGET}"

clean:
	rm -rf $(OUTPUT) node.ign

setup:
	GO111MODULE=off go get -u \
		github.com/onsi/ginkgo/ginkgo \
		github.com/coreos/gofail
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends $(PACKAGES)
	curl -fsSL -o protoc.zip https://github.com/google/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-linux-x86_64.zip
	$(SUDO) unzip -o protoc.zip -d /usr/local
	curl -fsSL -o ct https://github.com/coreos/container-linux-config-transpiler/releases/download/v0.9.0/ct-v0.9.0-x86_64-unknown-linux-gnu
	chmod +x ct
	$(SUDO) mv ct /usr/local/bin

.PHONY:	all clean setup test
