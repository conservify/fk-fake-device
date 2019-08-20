UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
GOARCH ?= amd64
GOOS ?= linux
endif

ifeq ($(UNAME_S),Darwin)
GOARCH ?= amd64
GOOS ?= darwin
endif

GO ?= env GOOS=$(GOOS) GOARCH=$(GOARCH) go
GOFLAGS ?=
BUILD ?= $(abspath build)
BUILDARCH ?= $(BUILD)/$(GOOS)-$(GOARCH)

build:
	GOOS=linux GOARCH=amd64 $(MAKE) binaries-all
	GOOS=linux GOARCH=arm $(MAKE) binaries-all
	GOOS=darwin GOARCH=amd64 $(MAKE) binaries-all

rebuild:
	go get github.com/fieldkit/app-protocol
	go get github.com/fieldkit/data-protocol
	GOOS=linux GOARCH=amd64 GOFLAGS=-a $(MAKE) binaries-all
	GOOS=linux GOARCH=arm GOFLAGS=-a $(MAKE) binaries-all
	GOOS=darwin GOARCH=amd64 GOFLAGS=-a $(MAKE) binaries-all

ci: rebuild

all: rebuild

binaries-all: $(BUILDARCH)/fake-device

$(BUILDARCH)/fake-device: *.go
	$(GO) build $(GOFLAGS) -o $(BUILDARCH)/fake-device *.go

clean:
	rm -rf $(BUILD)

run: build
	$(BUILDARCH)/fake-device

.PHONY: build
