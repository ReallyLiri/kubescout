GOCMD=go
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
BINARY_NAME=kubescout

.PHONY: all test build vendor

all: vet lint test clean build verify-binaries compress-bin

build: build-osx build-linux

build-osx:
ifneq ($(shell uname -s),Darwin)
	$(error this makefile assumes you're building from mac env)
endif
	GO111MODULE=on CGO_ENABLED=0 $(GOCMD) build -o bin/$(BINARY_NAME)-$(shell $(GOCMD) run . --version | cut -d" " -f 3)-osx .

build-linux:
	docker run --rm -v $(shell pwd):/app -w /app golang:alpine /bin/sh -c "GO111MODULE=on CGO_ENABLED=0 $(GOCMD) build -o bin/$(BINARY_NAME)-$(shell $(GOCMD) run . --version | cut -d" " -f 3)-linux ."

clean:
	rm -rf ./bin

vet:
	go vet

lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:latest-alpine golangci-lint run --deadline=65s

test:
	$(GOTEST) -v ./...

verify-binaries:
	$(info running binaries verification on osx, alpine, debiand, centos)
	./bin/$(BINARY_NAME)-$(shell $(GOCMD) run . --version | cut -d" " -f 3)-osx --version
	docker run --rm -v $(shell pwd)/bin:/app -w /app alpine /app/$(BINARY_NAME)-$(shell $(GOCMD) run . --version | cut -d" " -f 3)-linux --version
	docker run --rm -v $(shell pwd)/bin:/app -w /app debian:buster /app/$(BINARY_NAME)-$(shell $(GOCMD) run . --version | cut -d" " -f 3)-linux --version
	docker run --rm -v $(shell pwd)/bin:/app -w /app centos:8 /app/$(BINARY_NAME)-$(shell $(GOCMD) run . --version | cut -d" " -f 3)-linux --version

compress-bin:
	find bin -type f -print -exec zip -j '{}'.zip '{}' \;

integration:
	go test -tags integration ./...
