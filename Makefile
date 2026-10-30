APP_NAME := gh-c2
APP_VERSION := 0.0.10

.DEFAULT_GOAL := help
.PHONY: build build-mac build-linux clean

help:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ \
	{ printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## build binary for current platform
	CGO_ENABLED=0 go build \
	-ldflags "-s -w -X 'main.Version=$(APP_VERSION)'" \
	-o build/$(APP_NAME)

build-mac: ## build binary for macOS (arm64)
	CGO_ENABLED=0 go build \
	-ldflags "-s -w -X 'main.Version=$(APP_VERSION)'" \
	-o build/$(APP_NAME)_macos_arm64
	shasum -a 256 build/$(APP_NAME)_macos_arm64 > build/$(APP_NAME)_macos_arm64.sha256

build-linux: ## build binary for Linux (amd64)
	GOOS=linux GOARCH=amd64 go build \
	-ldflags "-s -w -X 'main.Version=$(APP_VERSION)'" \
	-o build/$(APP_NAME)_linux_amd64
	shasum -a 256 build/$(APP_NAME)_linux_amd64 > build/$(APP_NAME)_linux_amd64.sha256

clean: ## remove ALL build binaries
	rm -rf ./build/*
