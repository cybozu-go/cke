CNI_PLUGIN_VERSION = 0.9.0
TAG = quay.io/cybozu/cke-tools:dev
GOBUILD = CGO_ENABLED=0 go build -ldflags="-w -s"

.PHONY: all
all: bin/empty-dir bin/etcdbackup bin/install-cni bin/make_directories bin/rivers bin/write_files plugins

.PHONY: image
image: all
	docker build --no-cache -t $(TAG) .

bin/empty-dir:
	mkdir -p bin
	$(GOBUILD) -o $@ ./empty-dir

bin/etcdbackup:
	mkdir -p bin
	$(GOBUILD) -o $@ ./etcdbackup

bin/install-cni:
	mkdir -p bin
	$(GOBUILD) -o $@ ./install-cni

bin/make_directories:
	mkdir -p bin
	$(GOBUILD) -o $@ ./make_directories

bin/rivers:
	mkdir -p bin
	$(GOBUILD) -o $@ ./rivers

bin/write_files:
	mkdir -p bin
	$(GOBUILD) -o $@ ./write_files

.PHONY: plugins
plugins:
	rm -rf plugins
	git clone --depth 1 -b v$(CNI_PLUGIN_VERSION) https://github.com/containernetworking/plugins
	cd plugins; CGO_ENABLED=0 ./build_linux.sh

.PHONY: clean
clean:
	rm -rf bin plugins