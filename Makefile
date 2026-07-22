VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build build-team test lint install clean

build:
	go build $(LDFLAGS) -o bbgo .

# Team build with embedded OAuth client credentials (zero-config login):
#   make build-team CLIENT_ID=... CLIENT_SECRET=...
# Only embed credentials of an OAuth client whose Client credentials grant
# is DISABLED — see docs/oauth-setup.md.
build-team:
ifndef CLIENT_ID
	$(error CLIENT_ID is required: make build-team CLIENT_ID=... CLIENT_SECRET=...)
endif
ifndef CLIENT_SECRET
	$(error CLIENT_SECRET is required: make build-team CLIENT_ID=... CLIENT_SECRET=...)
endif
	go build -ldflags "-X main.version=$(VERSION) \
		-X github.com/josejibin/bbgo/cmd.DefaultOAuthClientID=$(CLIENT_ID) \
		-X github.com/josejibin/bbgo/cmd.DefaultOAuthClientSecret=$(CLIENT_SECRET)" -o bbgo .

test:
	go test -race ./...

lint:
	golangci-lint run ./...

install:
	go install $(LDFLAGS) .

clean:
	rm -f bbgo
