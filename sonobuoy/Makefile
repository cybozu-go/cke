### Configurable variables
### Make sure the binary is included in the GitHub release. The release of ct v0.9.4 does not include binary files.
SONOBUOY_VERSION = 0.57.3
KUBECTL_VERSION = 1.33.7
DOCKER_COMPOSE_VERSION = 5.0.1
CT_VERSION = 0.9.3
CILIUM_VERSION = 1.18.6

### Unconfigurable
CURL = curl -fsL
ifeq ($(shell id -u),0)
SUDO =
else
SUDO = sudo
endif

CILIUM_CLI_VERSION := $(shell curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
CILIUM_CLI_ARCH := $(shell if [ "$$(uname -m)" = "aarch64" ]; then echo arm64; else echo amd64; fi)
CILIUM_TARBALL = cilium-linux-$(CILIUM_CLI_ARCH).tar.gz
CILIUM_CHECKSUM = $(CILIUM_TARBALL).sha256sum

SONOBUOY_URL = https://github.com/vmware-tanzu/sonobuoy/releases/download/v$(SONOBUOY_VERSION)/sonobuoy_$(SONOBUOY_VERSION)_linux_amd64.tar.gz
KUBECTL_URL = https://dl.k8s.io/v$(KUBECTL_VERSION)/bin/linux/amd64/kubectl
DOCKER_COMPOSE_URL = https://github.com/docker/compose/releases/download/v$(DOCKER_COMPOSE_VERSION)/docker-compose-Linux-x86_64
CKECLI = ./bin/ckecli --config=./cke.config
KUBECONFIG := $(shell pwd)/.kubeconfig
CT = $(shell pwd)/bin/ct
export KUBECONFIG

.PHONY: all
all:
	@echo "Available targets:"
	@echo "    run              Run CKE and install Kubernetes into GCP Instances."
	@echo "    sonobuoy         Run sonobuoy."
	@echo "    clean            Stop VMs and clean files."
	@echo "    setup            Onetime setup.  Need root privilege."

bin/cke bin/ckecli:
	mkdir -p bin
	GOBIN=$(shell pwd)/bin go install ../pkg/$(notdir $@)

bin/kubectl:
	mkdir -p bin
	$(CURL) -o bin/kubectl $(KUBECTL_URL)
	chmod +x ./bin/kubectl

bin/sonobuoy:
	mkdir -p bin
	$(CURL) $(SONOBUOY_URL) | tar xzf - -C bin sonobuoy

bin/cilium:
	mkdir -p bin
	curl -L --fail --remote-name-all https://github.com/cilium/cilium-cli/releases/download/$(CILIUM_CLI_VERSION)/$(CILIUM_TARBALL){,.sha256sum}
	sha256sum --check $(CILIUM_CHECKSUM)
	sudo tar xzvfC $(CILIUM_TARBALL) ./bin
	rm $(CILIUM_TARBALL) $(CILIUM_CHECKSUM)

.PHONY: run
run: bin/cke bin/ckecli bin/kubectl bin/cilium
	mkdir -p etcd-data
	docker compose up -d
	./wait-up.sh
	$(CKECLI) vault ssh-privkey /home/cybozu/gcp_rsa
	$(CKECLI) constraints set control-plane-count 1
	$(CKECLI) cluster set /home/cybozu/cke-cluster.yml
	$(CKECLI) kubernetes issue > .kubeconfig
	for i in $$(seq 300); do sleep 1; if [ "$$(./bin/kubectl get -n kube-system ds/node-dns -o json | jq .status.numberReady)" = 3 ]; then break; fi; done
	./bin/cilium install --version $(CILIUM_VERSION) --wait
	for i in $$(seq 300); do sleep 1; if [ "$$(./bin/kubectl get nodes --no-headers | grep Ready | wc -l)" = 3 ]; then break; fi; done
	for i in $$(seq 300); do sleep 1; if [ "$$(./bin/kubectl get -n kube-system deployments/cluster-dns -o json | jq .status.readyReplicas)" = 2 ]; then break; fi; done

.PHONY: sonobuoy
sonobuoy: bin/sonobuoy
	$(CKECLI) kubernetes issue --ttl=10h > .kubeconfig
	time ./bin/sonobuoy run --mode=certified-conformance --timeout=14400 --wait
	outfile=$$(./bin/sonobuoy retrieve) && mv $$outfile sonobuoy.tar.gz
	./bin/sonobuoy run --rerun-failed sonobuoy.tar.gz 2>&1 | tee e2e-check.log
	./bin/sonobuoy delete

.PHONY: clean
clean:
	-docker compose down
	$(SUDO) rm -rf bin etcd-data

.PHONY: setup
setup:
	$(SUDO) mkdir -p /usr/local/lib/docker/cli-plugins
	$(SUDO) $(CURL) -o /usr/local/lib/docker/cli-plugins/docker-compose $(DOCKER_COMPOSE_URL)
	$(SUDO) chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

$(CT):
	mkdir -p bin
	$(CURL) -o ct https://github.com/flatcar/container-linux-config-transpiler/releases/download/v$(CT_VERSION)/ct-v$(CT_VERSION)-x86_64-unknown-linux-gnu
	chmod +x ct
	$(SUDO) mv ct $@

worker.ign: worker-ign.yml $(CT)
	$(CT) -in-file=$< -pretty -out-file=$@
