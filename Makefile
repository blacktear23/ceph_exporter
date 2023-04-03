CURDIR     := $(shell pwd)
BUILD_PATH := $(CURDIR)/build
LDFLAGS    := -s -w
BUILD_ARGS := -trimpath -ldflags '$(LDFLAGS)'

.PHONY: all build-linux

all: build-linux

prepare-path:
	@mkdir -p $(BUILD_PATH)/linux

build-linux: prepare-path
	GOARCH=amd64 GOOS=linux go build $(BUILD_ARGS) -o $(BUILD_PATH)/linux/ceph_exporter
	cp scripts/install.sh $(BUILD_PATH)/linux/
	cp scripts/ceph_exporter.service $(BUILD_PATH)/linux/
