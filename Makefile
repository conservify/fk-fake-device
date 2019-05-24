GOARCH ?= amd64
GOOS ?= linux
GO ?= env GOOS=$(GOOS) GOARCH=$(GOARCH) go
BUILD ?= $(abspath build)
BUILDARCH ?= $(BUILD)/$(GOOS)-$(GOARCH)

all:
	GOOS=linux GOARCH=amd64 $(MAKE) binaries-all
	GOOS=linux GOARCH=arm $(MAKE) binaries-all
	GOOS=darwin GOARCH=amd64 $(MAKE) binaries-all

binaries-all: $(BUILDARCH)/fake-device

$(BUILDARCH)/fake-device: *.go
	$(GO) build -o $(BUILDARCH)/fake-device *.go

clean:
	rm -rf $(BUILD)
