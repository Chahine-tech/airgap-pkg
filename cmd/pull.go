package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/Chahine-tech/airgap-pkg/internal/chart"
	"github.com/Chahine-tech/airgap-pkg/internal/config"
	"github.com/Chahine-tech/airgap-pkg/internal/image"
	"github.com/Chahine-tech/airgap-pkg/pkg/output"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull all images and charts defined in packages.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		p := output.New()
		imagesDir := filepath.Join(outputDir, "images")
		chartsDir := filepath.Join(outputDir, "charts")
		var failed int

		for _, pkg := range cfg.Packages {
			p.Section(pkg.Name)

			for _, img := range pkg.Images {
				p.Info(fmt.Sprintf("pulling image %s", img.Source))
				path, err := image.Pull(img.Source, imagesDir)
				if err != nil {
					p.Fail(fmt.Sprintf("image %s: %v", img.Source, err))
					failed++
					continue
				}
				p.OK(fmt.Sprintf("image saved → %s", path))
			}

			for _, ch := range pkg.Charts {
				p.Info(fmt.Sprintf("pulling chart %s/%s@%s", ch.Repo, ch.Name, ch.Version))
				path, err := chart.Pull(ch.Repo, ch.Name, ch.Version, chartsDir)
				if err != nil {
					p.Fail(fmt.Sprintf("chart %s@%s: %v", ch.Name, ch.Version, err))
					failed++
					continue
				}
				p.OK(fmt.Sprintf("chart saved → %s", path))
			}
		}

		if failed > 0 {
			return fmt.Errorf("%d artifact(s) failed to pull", failed)
		}
		return nil
	},
}
