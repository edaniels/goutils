BIN_ARCH = $(shell uname -s | tr '[:upper:]' '[:lower:]')-$(shell uname -m | tr '[:upper:]' '[:lower:]')
TOOL_BINS = bin/tools/$(BIN_ARCH)

.phony: lint

$(TOOL_BINS)/gofumpt:
	GOBIN=`pwd`/$(TOOL_BINS) go install mvdan.cc/gofumpt@latest

$(TOOL_BINS)/elinters:
	GOBIN=`pwd`/$(TOOL_BINS) go install github.com/edaniels/golinters/cmd/deferfor@HEAD
	GOBIN=`pwd`/$(TOOL_BINS) go install github.com/edaniels/golinters/cmd/mustcheck@HEAD
	GOBIN=`pwd`/$(TOOL_BINS) go install github.com/edaniels/golinters/cmd/printf@HEAD
	GOBIN=`pwd`/$(TOOL_BINS) go install github.com/edaniels/golinters/cmd/println@HEAD
	GOBIN=`pwd`/$(TOOL_BINS) go install github.com/edaniels/golinters/cmd/uselessf@HEAD

$(TOOL_BINS)/golangci-lint:
	GOBIN=`pwd`/$(TOOL_BINS) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.1

lint: $(TOOL_BINS)/gofumpt $(TOOL_BINS)/elinters $(TOOL_BINS)/golangci-lint
	$(TOOL_BINS)/gofumpt -l -w .
	go vet -vettool=$(TOOL_BINS)/deferfor ./...
	go vet -vettool=$(TOOL_BINS)/mustcheck ./...
	go vet -vettool=$(TOOL_BINS)/printf ./...
	go vet -vettool=$(TOOL_BINS)/println ./...
	go vet -vettool=$(TOOL_BINS)/uselessf ./...
	$(TOOL_BINS)/golangci-lint run -v --fix --config=./etc/.golangci.yaml ./...
