package sbom

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/Chahine-tech/airgap-pkg/internal/config"
	"github.com/Chahine-tech/airgap-pkg/internal/image"
)

// Component represents a single artifact — either a container image or a Helm chart.
type Component struct {
	Type    string `json:"type"`              // "image" or "chart"
	Package string `json:"package"`           // package name from config
	Source  string `json:"source,omitempty"`  // original image source ref (images only)
	Dest    string `json:"dest,omitempty"`    // dest ref inside registry (images only)
	Name    string `json:"name,omitempty"`    // chart name (charts only)
	Version string `json:"version,omitempty"` // chart version (charts only)
	Tarball string `json:"tarball"`           // base filename of the artifact on disk
	SHA256  string `json:"sha256"`            // hex SHA256, or "NOT_FOUND" if missing
}

// SBOM is the top-level document.
type SBOM struct {
	GeneratedAt string      `json:"generated_at"` // RFC3339 UTC
	Registry    string      `json:"registry"`
	Components  []Component `json:"components"`
}

// Generate builds an SBOM by iterating over config packages and computing
// SHA256 digests of artifact tarballs present on disk.
// imagesDir is typically <outputDir>/images, chartsDir is <outputDir>/charts.
// Missing tarballs are recorded with sha256:"NOT_FOUND" — no error is returned
// for missing files; only genuine I/O errors propagate.
func Generate(cfg *config.Config, imagesDir, chartsDir string) (*SBOM, error) {
	s := &SBOM{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Registry:    cfg.Registry,
		Components:  make([]Component, 0),
	}

	for _, pkg := range cfg.Packages {
		for _, img := range pkg.Images {
			tarball := image.RefToFilename(img.Source)
			digest, err := image.Verify(filepath.Join(imagesDir, tarball))
			if err != nil {
				digest = "NOT_FOUND"
			}
			s.Components = append(s.Components, Component{
				Type:    "image",
				Package: pkg.Name,
				Source:  img.Source,
				Dest:    img.Dest,
				Tarball: tarball,
				SHA256:  digest,
			})
		}

		for _, ch := range pkg.Charts {
			// Helm names chart downloads as <name>-<version>.tgz.
			tarball := fmt.Sprintf("%s-%s.tgz", ch.Name, ch.Version)
			digest, err := image.Verify(filepath.Join(chartsDir, tarball))
			if err != nil {
				digest = "NOT_FOUND"
			}
			s.Components = append(s.Components, Component{
				Type:    "chart",
				Package: pkg.Name,
				Name:    ch.Name,
				Version: ch.Version,
				Tarball: tarball,
				SHA256:  digest,
			})
		}
	}

	return s, nil
}
