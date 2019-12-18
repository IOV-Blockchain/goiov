# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: giov android ios giov-cross swarm evm all test clean
.PHONY: giov-linux giov-linux-386 giov-linux-amd64 giov-linux-mips64 giov-linux-mips64le
.PHONY: giov-linux-arm giov-linux-arm-5 giov-linux-arm-6 giov-linux-arm-7 giov-linux-arm64
.PHONY: giov-darwin giov-darwin-386 giov-darwin-amd64
.PHONY: giov-windows giov-windows-386 giov-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

giov:

	build/env.sh go run build/ci.go install ./cmd/giov
	@echo "Done building."
	@echo "Run \"$(GOBIN)/giov\" to launch giov."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/giov.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Geth.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

lint: ## Run linters.
	build/env.sh go run build/ci.go lint

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

giov-cross: giov-linux giov-darwin giov-windows giov-android giov-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/giov-*

giov-linux: giov-linux-386 giov-linux-amd64 giov-linux-arm giov-linux-mips64 giov-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/giov-linux-*

giov-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/giov
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/giov-linux-* | grep 386

giov-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/giov
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/giov-linux-* | grep amd64

giov-linux-arm: giov-linux-arm-5 giov-linux-arm-6 giov-linux-arm-7 giov-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/giov-linux-* | grep arm

giov-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/giov
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/giov-linux-* | grep arm-5

giov-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/giov
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/giov-linux-* | grep arm-6

giov-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/giov
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/giov-linux-* | grep arm-7

giov-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/giov
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/giov-linux-* | grep arm64

giov-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/giov
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/giov-linux-* | grep mips

giov-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/giov
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/giov-linux-* | grep mipsle

giov-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/giov
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/giov-linux-* | grep mips64

giov-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/giov
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/giov-linux-* | grep mips64le

giov-darwin: giov-darwin-386 giov-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/giov-darwin-*

giov-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/giov
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/giov-darwin-* | grep 386

giov-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/giov
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/giov-darwin-* | grep amd64

giov-windows: giov-windows-386 giov-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/giov-windows-*

giov-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/giov
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/giov-windows-* | grep 386

giov-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/giov
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/giov-windows-* | grep amd64
