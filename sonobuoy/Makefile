### Configurable variables
SONOBUOY_VERSION = 0.57.1
KUBECTL_VERSION = 1.27.10
DOCKER_COMPOSE_VERSION = 2.23.3
CT_VERSION = 0.9.3

### Unconfigurable
CURL = curl -fsL
ifeq ($(shell id -u),0)
SUDO =
else
SUDO = sudo
endif

SONOBUOY_URL = https://github.com/vmware-tanzu/sonobuoy/releases/download/v$(SONOBUOY_VERSION)/sonobuoy_$(SONOBUOY_VERSION)_linux_amd64.tar.gz
KUBECTL_URL = https://storage.googleapis.com/kubernetes-release/release/v$(KUBECTL_VERSION)/bin/linux/amd64/kubectl
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

.PHONY: run
run: bin/cke bin/ckecli bin/kubectl
	mkdir -p etcd-data
	/usr/local/bin/docker-compose up -d
	./wait-up.sh
	$(CKECLI) vault ssh-privkey /home/cybozu/gcp_rsa
	$(CKECLI) constraints set minimum-workers 2
	$(CKECLI) constraints set control-plane-count 1
	$(CKECLI) cluster set /home/cybozu/cke-cluster.yml
	$(CKECLI) kubernetes issue > .kubeconfig
	for i in $$(seq 300); do sleep 1; if [ "$$(./bin/kubectl get -n kube-system ds/node-dns -o json | jq .status.numberReady)" = 3 ]; then break; fi; done
	./bin/kubectl apply -k ../example
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
	-/usr/local/bin/docker-compose down
	$(SUDO) rm -rf bin etcd-data

.PHONY: setup
setup:
	$(SUDO) $(CURL) -o /usr/local/bin/docker-compose $(DOCKER_COMPOSE_URL)
	$(SUDO) chmod +x /usr/local/bin/docker-compose

$(CT):
	mkdir -p bin
	$(CURL) -o ct https://github.com/flatcar/container-linux-config-transpiler/releases/download/v$(CT_VERSION)/ct-v$(CT_VERSION)-x86_64-unknown-linux-gnu
	chmod +x ct
	$(SUDO) mv ct $@

worker.ign: worker-ign.yml $(CT)
	$(CT) -in-file=$< -pretty -out-file=$@
