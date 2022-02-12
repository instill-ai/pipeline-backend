.ONESHELL:
.PHONY:

all: build

##### Variables ######
ifndef GOPATH
GOPATH := $(shell go env GOPATH)
endif

GOBIN := $(if $(shell go env GOBIN),$(shell go env GOBIN),$(GOPATH)/bin)
PATH := $(GOBIN):$(PATH)

COLOR := "\e[1;36m%s\e[0m\n"

##### Build #####
build: build-server

build-server:
	go mod tidy
	go build -o pipeline-backend ./cmd/
