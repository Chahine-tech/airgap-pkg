BINARY  := airgap-pkg
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build install test tidy clean run-pull run-verify run-push run-status

build:
	go build -trimpath $(LDFLAGS) -o bin/$(BINARY) .

install:
	go install -trimpath $(LDFLAGS) .

test:
	go test ./... -v -race -timeout 60s

tidy:
	go mod tidy

clean:
	rm -rf bin/ artifacts/

run-pull:
	go run . pull --config examples/lumen-packages.yaml --output ./artifacts

run-verify:
	go run . verify --output ./artifacts

run-push:
	go run . push --config examples/lumen-packages.yaml --output ./artifacts

run-status:
	go run . status --config examples/lumen-packages.yaml
