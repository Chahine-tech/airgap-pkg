BINARY  := airgap-pkg
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: help build install test tidy clean run-pull run-verify run-push run-push-ssh run-status run-sbom run-sbom-cyclonedx run-diff run-bundle run-unbundle run-update

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

build: ## Compile → bin/airgap-pkg
	go build -trimpath $(LDFLAGS) -o bin/$(BINARY) .

install: ## Install binary to $GOPATH/bin
	go install -trimpath $(LDFLAGS) .

test: ## Run tests with race detector
	go test ./... -v -race -timeout 60s

tidy: ## Tidy go.mod
	go mod tidy

clean: ## Remove bin/ and artifacts/
	rm -rf bin/ artifacts/

run-pull: ## Pull images and charts (examples/lumen-packages.yaml)
	go run . pull --config examples/lumen-packages.yaml --output ./artifacts

run-verify: ## Verify SHA256 of artifacts/
	go run . verify --output ./artifacts

run-push: ## Push images to local registry
	go run . push --config examples/lumen-packages.yaml --output ./artifacts

run-push-ssh: ## Push via SSH tunnel through node-1
	go run . push --config examples/lumen-packages.yaml --output ./artifacts --via-ssh node-1

run-status: ## Check which images are in the registry
	go run . status --config examples/lumen-packages.yaml

run-sbom: ## Generate SBOM (JSON) to stdout
	go run . sbom --config examples/lumen-packages.yaml --output ./artifacts

run-sbom-cyclonedx: ## Generate SBOM in CycloneDX format → sbom.cdx.json
	go run . sbom --config examples/lumen-packages.yaml --output ./artifacts --format cyclonedx --out sbom.cdx.json

run-diff: ## Diff examples/lumen-packages.yaml against itself (no changes)
	go run . diff examples/lumen-packages.yaml examples/lumen-packages.yaml

run-bundle: ## Pack artifacts/ → airgap-bundle.tar.gz
	go run . bundle --config examples/lumen-packages.yaml --output ./artifacts --out airgap-bundle.tar.gz

run-unbundle: ## Extract bundle and push to localhost:5001
	go run . unbundle airgap-bundle.tar.gz --registry localhost:5001

run-update: ## Check for newer versions of all images and charts
	go run . update --config examples/lumen-packages.yaml
