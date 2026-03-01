package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoad_full(t *testing.T) {
	yaml := `
registry: 192.168.2.2:5000

transit:
  host: node-1
  port: "22"
  user: ubuntu
  ssh_key: ~/.ssh/id_rsa

packages:
  - name: chaos-mesh
    images:
      - source: ghcr.io/chaos-mesh/chaos-mesh:v2.7.2
        dest: chaos-mesh/chaos-mesh:v2.7.2
    charts:
      - repo: https://charts.chaos-mesh.org
        name: chaos-mesh
        version: "2.7.2"
`
	cfg, err := Load(writeYAML(t, yaml))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Registry != "192.168.2.2:5000" {
		t.Errorf("Registry = %q", cfg.Registry)
	}
	if cfg.Transit.Host != "node-1" {
		t.Errorf("Transit.Host = %q", cfg.Transit.Host)
	}
	if cfg.Transit.Port != "22" {
		t.Errorf("Transit.Port = %q", cfg.Transit.Port)
	}
	if cfg.Transit.User != "ubuntu" {
		t.Errorf("Transit.User = %q", cfg.Transit.User)
	}
	if len(cfg.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(cfg.Packages))
	}

	pkg := cfg.Packages[0]
	if pkg.Name != "chaos-mesh" {
		t.Errorf("Package.Name = %q", pkg.Name)
	}
	if len(pkg.Images) != 1 || pkg.Images[0].Source != "ghcr.io/chaos-mesh/chaos-mesh:v2.7.2" {
		t.Errorf("Images = %+v", pkg.Images)
	}
	if len(pkg.Charts) != 1 || pkg.Charts[0].Version != "2.7.2" {
		t.Errorf("Charts = %+v", pkg.Charts)
	}
}

func TestLoad_multiplePackages(t *testing.T) {
	yaml := `
registry: reg:5000
packages:
  - name: pkg-a
    images:
      - source: img-a:v1
        dest: img-a:v1
  - name: pkg-b
    charts:
      - repo: https://charts.example.com
        name: chart-b
        version: "1.0.0"
`
	cfg, err := Load(writeYAML(t, yaml))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(cfg.Packages))
	}
	if cfg.Packages[0].Name != "pkg-a" || cfg.Packages[1].Name != "pkg-b" {
		t.Errorf("unexpected package names: %q %q",
			cfg.Packages[0].Name, cfg.Packages[1].Name)
	}
}

func TestLoad_emptyPackages(t *testing.T) {
	yaml := `registry: reg:5000`
	cfg, err := Load(writeYAML(t, yaml))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.Packages) != 0 {
		t.Errorf("expected 0 packages, got %d", len(cfg.Packages))
	}
}

func TestLoad_fileNotFound(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoad_invalidYAML(t *testing.T) {
	_, err := Load(writeYAML(t, "{ invalid yaml: ["))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoad_transitOptionalFields(t *testing.T) {
	// Port, User, SSHKey are optional — should be empty strings when absent.
	yaml := `
registry: reg:5000
transit:
  host: node-1
packages: []
`
	cfg, err := Load(writeYAML(t, yaml))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Transit.Port != "" {
		t.Errorf("Port should be empty when not set, got %q", cfg.Transit.Port)
	}
	if cfg.Transit.User != "" {
		t.Errorf("User should be empty when not set, got %q", cfg.Transit.User)
	}
}
