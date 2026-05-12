SHELL := /bin/sh

OUTPUT_DIR := agent
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

GO_LDFLAGS := -w -s -X gaiasec-nodeagent/pkg/version.Version=$(VERSION)
GO_BUILD_FLAGS := -trimpath -ldflags "$(GO_LDFLAGS)"

CC_AMD64 ?= x86_64-linux-musl-gcc
CC_ARM64 ?= aarch64-linux-gnu-gcc
C_STATIC_FLAGS ?= -O2 -Wall -Wextra -static -s
JATTACH_CFLAGS ?= -O2 -static -s

.PHONY: all list pull commit build build-all build-nodeagent build-jattach build-mounter clean sync push push_image push_image_remote

all: list

list:
	@echo "Targets:"
	@$(MAKE) -qpRr | egrep -e '^[a-zA-Z0-9_-]+:$$' | sed -e 's~:~~g' | grep -v '^all$$' | sort

$(OUTPUT_DIR):
	mkdir -p $(OUTPUT_DIR)

build: build-all

build-all: build-nodeagent build-jattach build-mounter

build-nodeagent: $(OUTPUT_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(OUTPUT_DIR)/nodeagent-linux-amd64 ./cmd/nodeagent
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o $(OUTPUT_DIR)/nodeagent-linux-arm64 ./cmd/nodeagent

build-jattach: $(OUTPUT_DIR)
	$(MAKE) -C jattach clean
	$(MAKE) -C jattach CC='$(CC_AMD64)' CFLAGS='$(JATTACH_CFLAGS)' all
	cp jattach/build/jattach $(OUTPUT_DIR)/jattach-linux-amd64
	$(MAKE) -C jattach clean
	$(MAKE) -C jattach CC='$(CC_ARM64)' CFLAGS='$(JATTACH_CFLAGS)' all
	cp jattach/build/jattach $(OUTPUT_DIR)/jattach-linux-arm64
	$(MAKE) -C jattach clean

build-mounter: $(OUTPUT_DIR)
	$(CC_AMD64) $(C_STATIC_FLAGS) -o $(OUTPUT_DIR)/mounter-linux-amd64 ./mounter/src/main.c
	$(CC_ARM64) $(C_STATIC_FLAGS) -o $(OUTPUT_DIR)/mounter-linux-arm64 ./mounter/src/main.c

clean:
	rm -f $(OUTPUT_DIR)/nodeagent-linux-amd64 $(OUTPUT_DIR)/nodeagent-linux-arm64
	rm -f $(OUTPUT_DIR)/jattach-linux-amd64 $(OUTPUT_DIR)/jattach-linux-arm64
	rm -f $(OUTPUT_DIR)/mounter-linux-amd64 $(OUTPUT_DIR)/mounter-linux-arm64
	$(MAKE) -C jattach clean
	$(MAKE) -C mounter clean

sync:
	chmod a+rx ./sync.sh
	./sync.sh

pull:
	git checkout master
	git pull

commit:
	test -z "$$(git status --short)" || opencode run 'git commit it'

push:
	test -z "$$(git cherry -v)" || opencode run 'git push it'

push_image:
	echo "ok"

push_image_remote:
	bash push_image_remote.sh
