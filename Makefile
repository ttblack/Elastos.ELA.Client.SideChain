BUILD=go build
VERSION := $(shell git describe --abbrev=4 --dirty --always --tags)
Minversion := $(shell date)
BUILD_ELA_CLI = -ldflags "-X main.Version=$(VERSION)"

all:
	$(BUILD) $(BUILD_ELA_CLI) ela-cli.go

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(BUILD) $(BUILD_ELA_CLI) ela-cli.go
