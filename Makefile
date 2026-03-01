BINARY  := airgap-pkg
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build install test tidy clean run-pull run-verify run-push run-push-ssh run-status run-sbom run-sbom-cyclonedx run-diff run-bundle run-unbundle run-update

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

run-push-ssh:
	go run . push --config examples/lumen-packages.yaml --output ./artifacts --via-ssh node-1

run-status:
	go run . status --config examples/lumen-packages.yaml

run-sbom:
	go run . sbom --config examples/lumen-packages.yaml --output ./artifacts

run-sbom-cyclonedx:
	go run . sbom --config examples/lumen-packages.yaml --output ./artifacts --format cyclonedx --out sbom.cdx.json

run-diff:
	go run . diff examples/lumen-packages.yaml examples/lumen-packages.yaml

run-bundle:
	go run . bundle --config examples/lumen-packages.yaml --output ./artifacts --out airgap-bundle.tar.gz

run-unbundle:
	go run . unbundle airgap-bundle.tar.gz --registry localhost:5001

run-update:
	go run . update --config examples/lumen-packages.yaml
