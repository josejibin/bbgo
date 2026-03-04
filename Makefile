VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint install clean

build:
	go build $(LDFLAGS) -o bbgo .

test:
	go test ./...

lint:
	golangci-lint run ./...

install:
	go install $(LDFLAGS) .

clean:
	rm -f bbgo
