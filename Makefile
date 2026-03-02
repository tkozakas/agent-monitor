BINARY := agent-monitor

.PHONY: install build

build: $(BINARY)

install:
	go install .

GO_SRC := $(shell find . -name '*.go')
$(BINARY): $(GO_SRC) go.mod go.sum
	go build -o $(BINARY) .
