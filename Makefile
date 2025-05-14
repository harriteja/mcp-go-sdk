.PHONY: all test lint clean build examples cli install

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=mcp-go-sdk
CLI_NAME=mcp

all: test build cli

build:
	$(GOBUILD) -o bin/$(BINARY_NAME) -v ./...

cli:
	$(GOBUILD) -o bin/$(CLI_NAME) ./cmd/mcp

install: cli
	cp bin/$(CLI_NAME) $(GOPATH)/bin/

test:
	$(GOCMD) list ./...   | xargs -n1 -I{} sh -c 'echo "=== Testing {} ==="; $(GOTEST) -v {}'

lint:
	golangci-lint run

clean:
	$(GOCLEAN)
	rm -f bin/$(BINARY_NAME)
	rm -f bin/$(CLI_NAME)
	rm -f bin/simple-tool
	rm -f bin/simple-calculator
	rm -f bin/simple-stdio
	rm -f bin/simple-stdio-client

deps:
	$(GOMOD) download
	$(GOMOD) tidy

examples: build
	$(GOBUILD) -o bin/simple-tool ./examples/servers/simple-tool
	$(GOBUILD) -o bin/simple-calculator ./examples/clients/simple-calculator
	$(GOBUILD) -o bin/simple-stdio ./examples/servers/simple-stdio
	$(GOBUILD) -o bin/simple-stdio-client ./examples/clients/simple-stdio-client

# Run examples
run-http-server:
	./bin/simple-tool

run-http-client:
	./bin/simple-calculator

run-stdio-server:
	./bin/simple-stdio

run-stdio-client:
	./bin/simple-stdio-client 