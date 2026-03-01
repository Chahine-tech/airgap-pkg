package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Chahine-tech/airgap-pkg/internal/config"
	"github.com/Chahine-tech/airgap-pkg/internal/image"
	"github.com/Chahine-tech/airgap-pkg/pkg/output"
	"github.com/spf13/cobra"
)

var pushRegistry string

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push all images to the internal registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		registry := pushRegistry
		if registry == "" {
			registry = cfg.Registry
		}
		if registry == "" {
			return fmt.Errorf("no registry specified: set 'registry' in packages.yaml or use --registry")
		}

		p := output.New()
		imagesDir := filepath.Join(outputDir, "images")
		var failed int

		for _, pkg := range cfg.Packages {
			p.Section(pkg.Name)

			for _, img := range pkg.Images {
				filename := image.RefToFilename(img.Source)
				tarPath := filepath.Join(imagesDir, filename)

				if _, err := os.Stat(tarPath); os.IsNotExist(err) {
					p.Fail(fmt.Sprintf("tarball not found for %s (run pull first)", img.Source))
					failed++
					continue
				}

				p.Info(fmt.Sprintf("pushing %s → %s/%s", filename, registry, img.Dest))
				if err := image.Push(tarPath, registry, img.Dest); err != nil {
					p.Fail(fmt.Sprintf("%s: %v", img.Dest, err))
					failed++
					continue
				}
				p.OK(fmt.Sprintf("pushed → %s/%s", registry, img.Dest))
			}
		}

		if failed > 0 {
			return fmt.Errorf("%d image(s) failed to push", failed)
		}
		return nil
	},
}

func init() {
	pushCmd.Flags().StringVar(&pushRegistry, "registry", "", "override registry from packages.yaml (e.g. 192.168.2.2:5000)")
}
