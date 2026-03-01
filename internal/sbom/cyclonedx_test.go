package sbom

import (
	"strings"
	"testing"
)

func TestToCycloneDX_metadata(t *testing.T) {
	s := &SBOM{
		GeneratedAt: "2025-01-01T00:00:00Z",
		Registry:    "192.168.1.1:5000",
		Components:  []Component{},
	}
	bom := ToCycloneDX(s)

	if bom.BOMFormat != "CycloneDX" {
		t.Errorf("BOMFormat = %q", bom.BOMFormat)
	}
	if bom.SpecVersion != "1.6" {
		t.Errorf("SpecVersion = %q", bom.SpecVersion)
	}
	if bom.Metadata.Timestamp != "2025-01-01T00:00:00Z" {
		t.Errorf("Timestamp = %q", bom.Metadata.Timestamp)
	}
	if len(bom.Metadata.Tools) != 1 || bom.Metadata.Tools[0].Name != "airgap-pkg" {
		t.Errorf("unexpected tools: %+v", bom.Metadata.Tools)
	}
	if len(bom.Components) != 0 {
		t.Errorf("expected 0 components, got %d", len(bom.Components))
	}
}

func TestToCycloneDX_imageComponent(t *testing.T) {
	s := &SBOM{
		GeneratedAt: "2025-01-01T00:00:00Z",
		Registry:    "192.168.1.1:5000",
		Components: []Component{
			{
				Type:   "image",
				Dest:   "chaos-mesh/chaos-mesh:v2.7.2",
				SHA256: "abc123",
			},
		},
	}
	bom := ToCycloneDX(s)

	if len(bom.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(bom.Components))
	}
	c := bom.Components[0]
	if c.Type != "container" {
		t.Errorf("Type = %q, want container", c.Type)
	}
	if c.Name != "chaos-mesh/chaos-mesh" {
		t.Errorf("Name = %q", c.Name)
	}
	if c.Version != "v2.7.2" {
		t.Errorf("Version = %q", c.Version)
	}
	if !strings.HasPrefix(c.PURL, "pkg:oci/") {
		t.Errorf("PURL should start with pkg:oci/, got %q", c.PURL)
	}
	if !strings.Contains(c.PURL, "192.168.1.1:5000") {
		t.Errorf("PURL should contain registry, got %q", c.PURL)
	}
	if len(c.Hashes) != 1 || c.Hashes[0].Content != "abc123" {
		t.Errorf("unexpected hashes: %+v", c.Hashes)
	}
}

func TestToCycloneDX_imageNotFound_noHashes(t *testing.T) {
	s := &SBOM{
		GeneratedAt: "2025-01-01T00:00:00Z",
		Registry:    "reg:5000",
		Components: []Component{
			{Type: "image", Dest: "img:latest", SHA256: "NOT_FOUND"},
		},
	}
	bom := ToCycloneDX(s)

	if len(bom.Components) != 1 {
		t.Fatalf("expected 1 component")
	}
	if len(bom.Components[0].Hashes) != 0 {
		t.Errorf("NOT_FOUND component should have no hashes, got %+v", bom.Components[0].Hashes)
	}
}

func TestToCycloneDX_chartComponent(t *testing.T) {
	s := &SBOM{
		GeneratedAt: "2025-01-01T00:00:00Z",
		Registry:    "reg:5000",
		Components: []Component{
			{Type: "chart", Name: "chaos-mesh", Version: "2.7.2", SHA256: "def456"},
		},
	}
	bom := ToCycloneDX(s)

	if len(bom.Components) != 1 {
		t.Fatalf("expected 1 component")
	}
	c := bom.Components[0]
	if c.Type != "library" {
		t.Errorf("Type = %q, want library", c.Type)
	}
	if c.PURL != "pkg:helm/chaos-mesh@2.7.2" {
		t.Errorf("PURL = %q", c.PURL)
	}
	if len(c.Hashes) != 1 || c.Hashes[0].Alg != "SHA-256" {
		t.Errorf("unexpected hashes: %+v", c.Hashes)
	}
}

func TestToCycloneDX_unknownTypeSkipped(t *testing.T) {
	s := &SBOM{
		GeneratedAt: "2025-01-01T00:00:00Z",
		Registry:    "reg:5000",
		Components: []Component{
			{Type: "unknown", Name: "whatever"},
		},
	}
	bom := ToCycloneDX(s)
	if len(bom.Components) != 0 {
		t.Errorf("unknown type should be skipped, got %d components", len(bom.Components))
	}
}

func TestSplitRef(t *testing.T) {
	cases := []struct{ ref, name, version string }{
		{"chaos-mesh/chaos-mesh:v2.7.2", "chaos-mesh/chaos-mesh", "v2.7.2"},
		{"img", "img", "latest"},
		{"a/b/c:tag", "a/b/c", "tag"},
	}
	for _, tc := range cases {
		n, v := splitRef(tc.ref)
		if n != tc.name || v != tc.version {
			t.Errorf("splitRef(%q) = (%q, %q), want (%q, %q)", tc.ref, n, v, tc.name, tc.version)
		}
	}
}

func TestLastName(t *testing.T) {
	cases := []struct{ in, out string }{
		{"chaos-mesh/chaos-mesh", "chaos-mesh"},
		{"simple", "simple"},
		{"a/b/c", "c"},
	}
	for _, tc := range cases {
		if got := lastName(tc.in); got != tc.out {
			t.Errorf("lastName(%q) = %q, want %q", tc.in, got, tc.out)
		}
	}
}
