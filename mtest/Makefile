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
PLACEMAT = $(GOPATH)/bin/placemat
GINKGO = $(GOPATH)/bin/ginkgo
MANAGEMENT_ETCD_VERSION = 3.3.9
PLACEMAT_DATADIR = /var/scratch/placemat

export BRIDGE_ADDRESS HOST1 HOST2 NODE1 NODE2 NODE3 NODE4 NODE5 NODE6 PLACEMAT GINKGO

# non-configuration variables
MANAGEMENT_ETCD_ARCHIVE := etcd-v$(MANAGEMENT_ETCD_VERSION)-linux-amd64.tar.gz
SSH_PRIVKEY := ./mtest_key
OUTPUT := ./output
UBUNTU_IMAGE := ubuntu-18.04-server-cloudimg-amd64.img
COREOS_IMAGE := coreos_production_qemu_image.img
CKECLI := $(OUTPUT)/ckecli
CKECLUSTER := $(OUTPUT)/cke-cluster.yml
CKECONFIG := $(OUTPUT)/cke.yml

export SSH_PRIVKEY CKECLI CKECLUSTER CKECONFIG


GENERATED_FILES = $(OUTPUT)/etcd  $(OUTPUT)/etcdctl \
	$(OUTPUT)/cke $(OUTPUT)/ckecli \
	$(OUTPUT)/cluster.yml \
	$(OUTPUT)/user-data.yml \
	$(OUTPUT)/network1.yml \
	$(OUTPUT)/network2.yml \
	$(OUTPUT)/node1.ign \
	$(OUTPUT)/node2.ign \
	$(OUTPUT)/node3.ign \
	$(OUTPUT)/node4.ign \
	$(OUTPUT)/node5.ign \
	$(OUTPUT)/node6.ign \
	$(OUTPUT)/cke-cluster.yml \
	$(OUTPUT)/cke.yml \
	$(UBUNTU_IMAGE) \
	$(COREOS_IMAGE)

all: test

$(MANAGEMENT_ETCD_ARCHIVE):
	curl -sSLf -o $@ https://github.com/coreos/etcd/releases/download/v$(MANAGEMENT_ETCD_VERSION)/etcd-v$(MANAGEMENT_ETCD_VERSION)-linux-amd64.tar.gz

$(UBUNTU_IMAGE):
	curl -sSLf -o $@ https://cloud-images.ubuntu.com/releases/18.04/release/$@

$(COREOS_IMAGE):
	curl -sSLf -o $@.bz2 https://stable.release.core-os.net/amd64-usr/current/$@.bz2
	bunzip2 -q $@.bz2

$(OUTPUT)/etcd $(OUTPUT)/etcdctl: $(MANAGEMENT_ETCD_ARCHIVE)
	mkdir -p $(OUTPUT)
	tar -C $(OUTPUT) -x -z --wildcards --strip-components 1 -f $< '*/$(notdir $@)'
	touch $@

$(OUTPUT)/cke $(OUTPUT)/ckecli:
	mkdir -p $(OUTPUT)
	cd ..; gofail enable; GOBIN=$(realpath $(OUTPUT)) go install ./cmd/$(notdir $@) || gofail disable; gofail disable

$(OUTPUT)/cluster.yml: cluster.yml
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@UBUNTU_IMAGE@/$(UBUNTU_IMAGE)/' \
		-e 's/@COREOS_IMAGE@/$(COREOS_IMAGE)/' $< > $@

$(OUTPUT)/user-data.yml: user-data.yml
	mkdir -p $(OUTPUT)
	sed -e 's/@HOST1@/$(HOST1)/' $< > $@

$(OUTPUT)/network1.yml: network.yml
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@HOST@/$(HOST1)/' $< > $@

$(OUTPUT)/network2.yml: network.yml
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@HOST@/$(HOST2)/' $< > $@

$(OUTPUT)/node1.ign: node.ign
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@NODE_NAME@/node1/' \
		-e 's/@NODE_ADDRESS@/$(NODE1)/' $< > $@

$(OUTPUT)/node2.ign: node.ign
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@NODE_NAME@/node2/' \
		-e 's/@NODE_ADDRESS@/$(NODE2)/' $< > $@

$(OUTPUT)/node3.ign: node.ign
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@NODE_NAME@/node3/' \
		-e 's/@NODE_ADDRESS@/$(NODE3)/' $< > $@

$(OUTPUT)/node4.ign: node.ign
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@NODE_NAME@/node4/' \
		-e 's/@NODE_ADDRESS@/$(NODE4)/' $< > $@

$(OUTPUT)/node5.ign: node.ign
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@NODE_NAME@/node5/' \
		-e 's/@NODE_ADDRESS@/$(NODE5)/' $< > $@

$(OUTPUT)/node6.ign: node.ign
	mkdir -p $(OUTPUT)
	sed -e 's/@BRIDGE_ADDRESS@/$(BRIDGE_ADDRESS)/' \
		-e 's/@NODE_NAME@/node6/' \
		-e 's/@NODE_ADDRESS@/$(NODE6)/' $< > $@

$(OUTPUT)/cke-cluster.yml: cke-cluster.yml
	mkdir -p $(OUTPUT)
	sed -e 's/@NODE1@/$(NODE1)/' \
		-e 's/@NODE2@/$(NODE2)/' \
		-e 's/@NODE3@/$(NODE3)/' \
		-e 's/@NODE4@/$(NODE4)/' \
		-e 's/@NODE5@/$(NODE5)/' \
		-e 's/@NODE6@/$(NODE6)/' $< > $@

$(OUTPUT)/cke.yml: cke.yml
	mkdir -p $(OUTPUT)
	sed -e 's/@HOST1@/$(HOST1)/' $< > $@

test: $(GENERATED_FILES)
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/host1
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/host2
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node1
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node2
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node3
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node4
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node5
	sudo rm -rf $(PLACEMAT_DATADIR)/volumes/node6
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
	sudo -b $(PLACEMAT) output/cluster.yml
	@echo 'Placemat is running.  To kill it, do "sudo pkill placemat".'

test-light:
	$(GINKGO)

clean:
	rm -rf $(OUTPUT)

setup:
	go get -u github.com/onsi/ginkgo/ginkgo
	go get -u github.com/onsi/gomega/...
	go get -u github.com/cybozu-go/placemat
	go get -u github.com/coreos/gofail/...

.PHONY:	all clean setup test
