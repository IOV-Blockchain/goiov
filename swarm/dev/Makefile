.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/CarLiveChainCo/goiov/cmd/swarm
	go build -o bin/geth     github.com/CarLiveChainCo/goiov/cmd/geth
	go build -o bin/bootnode github.com/CarLiveChainCo/goiov/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/CarLiveChainCo/goiov/swarm/...
