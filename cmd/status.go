package cmd

import (
	"fmt"

	"github.com/Chahine-tech/airgap-pkg/internal/config"
	"github.com/Chahine-tech/airgap-pkg/internal/registry"
	"github.com/Chahine-tech/airgap-pkg/pkg/output"
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
					p.OK(fmt.Sprintf("PRESENT  %s/%s  (%s)", cfg.Registry, img.Dest, status.Digest))
				}
			}
		}

		if missing > 0 {
			return fmt.Errorf("%d image(s) missing from registry", missing)
		}
		return nil
	},
}
