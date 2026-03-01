# airgap-pkg

A Go CLI to automate Docker image and Helm chart packaging for air-gapped deployments.

## The Problem

Every deployment to an isolated environment involves the same repetitive workflow:

```
docker pull → docker save → USB/SCP transfer → docker load → docker tag → docker push → verify
```

`airgap-pkg` automates all of this from a single declarative `packages.yaml` file.

## Install

```bash
go install github.com/Chahine-tech/airgap-pkg@latest
# or
make build && ./bin/airgap-pkg
```

## Configuration

```yaml
# packages.yaml
registry: 192.168.2.2:5000

packages:
  - name: chaos-mesh
    images:
      - source: ghcr.io/chaos-mesh/chaos-mesh:v2.7.2
        dest: chaos-mesh/chaos-mesh:v2.7.2
      - source: ghcr.io/chaos-mesh/chaos-daemon:v2.7.2
        dest: chaos-mesh/chaos-daemon:v2.7.2
    charts:
      - repo: https://charts.chaos-mesh.org
        name: chaos-mesh
        version: "2.7.2"
```

## Commands

### `pull` — download images and charts

```bash
airgap-pkg pull --config packages.yaml --output ./artifacts
```

Produces `artifacts/images/*.tar` and `artifacts/charts/*.tgz`.

### `verify` — check SHA256 integrity

```bash
airgap-pkg verify --output ./artifacts
```

### `push` — push to the internal registry

```bash
airgap-pkg push --config packages.yaml --output ./artifacts
# Override the registry from the config file:
airgap-pkg push --config packages.yaml --registry localhost:5001
```

Works over plain HTTP (insecure registries like `192.168.2.2:5000`).

### `status` — check what's already in the registry

```bash
airgap-pkg status --config packages.yaml
```

Returns a non-zero exit code if any image is missing — scriptable and CI-friendly.

## Typical Workflow

```bash
# 1. Connected zone: pull everything
airgap-pkg pull --config packages.yaml --output ./artifacts

# 2. Verify before transfer
airgap-pkg verify --output ./artifacts

# 3. Transfer to the transit zone (USB, SCP...)
scp -r ./artifacts node-1:~/artifacts

# 4. From node-1: push to the internal registry
airgap-pkg push --config packages.yaml --output ~/artifacts --registry 192.168.2.2:5000

# 5. Verify
airgap-pkg status --config packages.yaml
```

## Stack

- No Docker daemon required — images handled via [go-containerregistry](https://github.com/google/go-containerregistry)
- Helm charts via `helm.sh/helm/v3` (no shell calls)
- CLI powered by [cobra](https://github.com/spf13/cobra)

## Makefile

```bash
make build       # compile → bin/airgap-pkg
make test        # go test ./...
make run-pull    # pull with examples/lumen-packages.yaml
make run-verify  # verify artifacts/
make clean       # remove bin/ and artifacts/
```
