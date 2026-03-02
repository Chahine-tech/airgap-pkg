package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/Chahine-tech/airgap-pkg/internal/config"
	"github.com/Chahine-tech/airgap-pkg/internal/image"
	"github.com/Chahine-tech/airgap-pkg/internal/registry"
	"github.com/Chahine-tech/airgap-pkg/pkg/output"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show which images from packages.yaml are present in the registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if cfg.Registry == "" {
			return fmt.Errorf("no registry specified in packages.yaml")
		}

		p := output.New()
		var missing int

		for _, pkg := range cfg.Packages {
			p.Section(pkg.Name)

			for _, img := range pkg.Images {
				status := registry.Check(cfg.Registry, img.Dest)
				if status.Err != nil {
					p.Fail(fmt.Sprintf("probe error for %s: %v", img.Dest, status.Err))
					missing++
				} else if !status.Exists {
					p.Fail(fmt.Sprintf("MISSING  %s/%s", cfg.Registry, img.Dest))
					missing++
				} else {
					tarPath := filepath.Join(outputDir, "images", image.RefToFilename(img.Source))
					localDigest := localOCIDigest(tarPath)
					if localDigest != "" && localDigest != status.Digest {
						p.Warn(fmt.Sprintf("STALE    %s/%s  (registry: %s  local: %s)", cfg.Registry, img.Dest, status.Digest, localDigest))
					} else {
						p.OK(fmt.Sprintf("PRESENT  %s/%s  (%s)", cfg.Registry, img.Dest, status.Digest))
					}
				}
			}
		}

		if missing > 0 {
			return fmt.Errorf("%d image(s) missing from registry", missing)
		}
		return nil
	},
}

// localOCIDigest returns the OCI digest of a local tarball, or "" if unavailable.
func localOCIDigest(tarPath string) string {
	img, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return ""
	}
	d, err := img.Digest()
	if err != nil {
		return ""
	}
	return d.String()
}
