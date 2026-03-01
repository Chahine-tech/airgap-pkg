package sbom

import (
	"fmt"
	"strings"
)

// CycloneDXBOM is a minimal CycloneDX v1.6 JSON document.
type CycloneDXBOM struct {
	BOMFormat   string               `json:"bomFormat"`
	SpecVersion string               `json:"specVersion"`
	Version     int                  `json:"version"`
	Metadata    CycloneDXMetadata    `json:"metadata"`
	Components  []CycloneDXComponent `json:"components"`
}

// CycloneDXMetadata carries document-level metadata.
type CycloneDXMetadata struct {
	Timestamp string          `json:"timestamp"`
	Tools     []CycloneDXTool `json:"tools"`
}

// CycloneDXTool identifies the tool that generated the BOM.
type CycloneDXTool struct {
	Vendor  string `json:"vendor"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// CycloneDXComponent is a single software component entry.
type CycloneDXComponent struct {
	Type    string          `json:"type"`            // "container" or "library"
	Name    string          `json:"name"`
	Version string          `json:"version"`
	PURL    string          `json:"purl,omitempty"`
	Hashes  []CycloneDXHash `json:"hashes,omitempty"`
}

// CycloneDXHash is a hash entry using CycloneDX algorithm identifiers.
type CycloneDXHash struct {
	Alg     string `json:"alg"`
	Content string `json:"content"`
}

// ToCycloneDX converts an SBOM to CycloneDX v1.6 format.
// The metadata timestamp is taken from s.GeneratedAt to ensure consistency.
// Components with sha256:"NOT_FOUND" are included without a hashes entry.
func ToCycloneDX(s *SBOM) *CycloneDXBOM {
	bom := &CycloneDXBOM{
		BOMFormat:   "CycloneDX",
		SpecVersion: "1.6",
		Version:     1,
		Metadata: CycloneDXMetadata{
			Timestamp: s.GeneratedAt,
			Tools: []CycloneDXTool{
				{Vendor: "Chahine-tech", Name: "airgap-pkg", Version: "dev"},
			},
		},
		Components: make([]CycloneDXComponent, 0),
	}

	for _, c := range s.Components {
		var comp CycloneDXComponent
		switch c.Type {
		case "image":
			comp = imageComponent(c, s.Registry)
		case "chart":
			comp = chartComponent(c)
		default:
			continue
		}
		bom.Components = append(bom.Components, comp)
	}

	return bom
}

// imageComponent converts an image Component to a CycloneDX container component.
// PURL: pkg:oci/<imageName>@<version>?repository_url=<registry>/<repoPath>
func imageComponent(c Component, registry string) CycloneDXComponent {
	name, version := splitRef(c.Dest)
	purl := fmt.Sprintf("pkg:oci/%s@%s?repository_url=%s/%s",
		lastName(name), version, registry, name)

	comp := CycloneDXComponent{
		Type:    "container",
		Name:    name,
		Version: version,
		PURL:    purl,
	}
	if c.SHA256 != "NOT_FOUND" && c.SHA256 != "" {
		comp.Hashes = []CycloneDXHash{{Alg: "SHA-256", Content: c.SHA256}}
	}
	return comp
}

// chartComponent converts a chart Component to a CycloneDX library component.
// PURL: pkg:helm/<name>@<version>
func chartComponent(c Component) CycloneDXComponent {
	comp := CycloneDXComponent{
		Type:    "library",
		Name:    c.Name,
		Version: c.Version,
		PURL:    fmt.Sprintf("pkg:helm/%s@%s", c.Name, c.Version),
	}
	if c.SHA256 != "NOT_FOUND" && c.SHA256 != "" {
		comp.Hashes = []CycloneDXHash{{Alg: "SHA-256", Content: c.SHA256}}
	}
	return comp
}

// splitRef splits "repo/name:tag" into ("repo/name", "tag").
func splitRef(ref string) (name, version string) {
	idx := strings.LastIndex(ref, ":")
	if idx < 0 {
		return ref, "latest"
	}
	return ref[:idx], ref[idx+1:]
}

// lastName returns the last path segment: "chaos-mesh/chaos-mesh" → "chaos-mesh".
func lastName(name string) string {
	idx := strings.LastIndex(name, "/")
	if idx < 0 {
		return name
	}
	return name[idx+1:]
}
