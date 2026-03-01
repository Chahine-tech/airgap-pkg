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

# Control concurrency (default: 4 workers)
airgap-pkg pull --config packages.yaml --workers 8
```

Pulls all images and charts concurrently. Produces `artifacts/images/*.tar` and `artifacts/charts/*.tgz`.
A single failed artifact is reported but does not abort the others — the command exits non-zero if any artifact failed.

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

#### SSH tunnel mode

When the machine running `airgap-pkg` can reach a transit host via SSH but has no direct network path to the registry:

```bash
# Uses transit config from packages.yaml
airgap-pkg push --config packages.yaml --via-ssh node-1

# Full flag override
airgap-pkg push --config packages.yaml \
  --via-ssh node-1 --ssh-key ~/.ssh/id_rsa \
  --ssh-user ubuntu --ssh-port 22 \
  --registry 192.168.2.2:5000
```

`airgap-pkg` opens a local TCP listener on a random port and forwards all traffic to the registry through the SSH connection. No binary is required on the transit host.

**Transit block in `packages.yaml`:**

```yaml
transit:
  host: node-1
  port: "22"           # optional, default 22
  user: ubuntu         # optional, default $USER
  ssh_key: ~/.ssh/id_rsa
```

Flag precedence: CLI flag > `packages.yaml` > default.

**Host key verification:** `~/.ssh/known_hosts` is used automatically. If the file does not exist, a `[WARN]` is printed and host key checking is disabled.

### `sbom` — generate a Software Bill of Materials

```bash
# JSON to stdout (pipe-friendly)
airgap-pkg sbom --config packages.yaml --output ./artifacts

# Write to file
airgap-pkg sbom --config packages.yaml --output ./artifacts --out sbom.json

# CycloneDX v1.6 (compatible with Dependency-Track, Grype...)
airgap-pkg sbom --config packages.yaml --format cyclonedx --out sbom.cdx.json
```

Reads `packages.yaml` and computes SHA256 digests from the tarballs in `artifacts/`. Components whose tarball is missing appear with `sha256: "NOT_FOUND"` — a warning is printed to stderr but the command succeeds.

Supported formats: `json` (default), `cyclonedx`.

### `status` — check what's already in the registry

```bash
airgap-pkg status --config packages.yaml
```

Returns a non-zero exit code if any image is missing — scriptable and CI-friendly.

### `diff` — compare two packages.yaml files

```bash
airgap-pkg diff packages-v1.yaml packages-v2.yaml
```

Shows additions (`ADD`), removals (`DEL`), and version changes (`UPD`) between two config files. Exits with code 1 when differences are found — useful in CI to gate deployments on unexpected changes.

```bash
# Show unchanged entries too
airgap-pkg diff packages-v1.yaml packages-v2.yaml --all
```

Example output:

```
=== Images ===
[ADD] chaos-mesh/chaos-daemon:v2.7.3  ← ghcr.io/chaos-mesh/chaos-daemon:v2.7.3
[DEL] chaos-mesh/chaos-daemon:v2.7.2  (was ghcr.io/chaos-mesh/chaos-daemon:v2.7.2)
[UPD] chaos-mesh/chaos-mesh:latest  ghcr.io/chaos-mesh/chaos-mesh:v2.7.2 → ghcr.io/chaos-mesh/chaos-mesh:v2.7.3

=== Charts ===
[UPD] chaos-mesh  2.7.2 → 2.7.3
```

### `bundle` — pack artifacts into a single archive

```bash
airgap-pkg bundle --config packages.yaml --output ./artifacts --out airgap-bundle.tar.gz
```

Packs all images and charts from `artifacts/` into a single `.tar.gz` archive with an embedded `manifest.json`.
The manifest records the registry and destination path of each image — `unbundle` can push without the original `packages.yaml`.

### `unbundle` — extract and push from a bundle

```bash
# Push using the registry from the bundle manifest
airgap-pkg unbundle airgap-bundle.tar.gz

# Override the registry
airgap-pkg unbundle airgap-bundle.tar.gz --registry 192.168.2.2:5000

# Extract first, push via SSH tunnel
airgap-pkg unbundle airgap-bundle.tar.gz --registry 192.168.2.2:5000 --via-ssh node-1

# Keep extracted files for inspection
airgap-pkg unbundle airgap-bundle.tar.gz --extract ./unpacked
```

## Typical Workflow

**Without SSH access to the registry (direct):**

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

**With SSH tunnel (push from connected zone through node-1):**

```bash
airgap-pkg pull   --config packages.yaml --output ./artifacts
airgap-pkg verify --output ./artifacts
airgap-pkg push   --config packages.yaml --via-ssh node-1
airgap-pkg status --config packages.yaml
```

**With bundle (single-file USB transfer, no packages.yaml needed on the other side):**

```bash
# Connected zone
airgap-pkg pull   --config packages.yaml --output ./artifacts
airgap-pkg verify --output ./artifacts
airgap-pkg bundle --config packages.yaml --output ./artifacts --out airgap-bundle.tar.gz

# Transfer one file
scp airgap-bundle.tar.gz node-1:~

# Air-gapped zone — no config file needed
airgap-pkg unbundle airgap-bundle.tar.gz --registry 192.168.2.2:5000
```

## Stack

- No Docker daemon required — images handled via [go-containerregistry](https://github.com/google/go-containerregistry)
- Helm charts via `helm.sh/helm/v3` (no shell calls)
- CLI powered by [cobra](https://github.com/spf13/cobra)

## Makefile

```bash
make build               # compile → bin/airgap-pkg
make test                # go test ./...
make run-pull            # pull with examples/lumen-packages.yaml
make run-verify          # verify artifacts/
make run-push-ssh        # push via SSH tunnel through node-1
make run-sbom            # generate SBOM (JSON) to stdout
make run-sbom-cyclonedx  # generate SBOM in CycloneDX format → sbom.cdx.json
make run-diff            # diff examples/lumen-packages.yaml against itself (no changes)
make run-bundle          # pack artifacts/ → airgap-bundle.tar.gz
make run-unbundle        # unbundle airgap-bundle.tar.gz → localhost:5001
make clean               # remove bin/ and artifacts/
```
