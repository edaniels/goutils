TOOL_BIN = bin/gotools/$(shell uname -s)-$(shell uname -m)
PATH_WITH_TOOLS="`pwd`/$(TOOL_BIN)"

tool-install:
	GOBIN=`pwd`/$(TOOL_BIN)  go install \
		`go list -e -f '{{ range $$import := .Imports }} {{ $$import }} {{ end }}' ./tools/tools.go`

lint: tool-install
	go vet -vettool=$(TOOL_BIN)/combined ./...
	$(TOOL_BIN)/golangci-lint run -v --fix --config=./etc/.golangci.yaml ./...
