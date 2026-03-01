package cmd

import (
	"fmt"

	"github.com/Chahine-tech/airgap-pkg/internal/bundle"
	"github.com/Chahine-tech/airgap-pkg/internal/config"
	"github.com/Chahine-tech/airgap-pkg/pkg/output"
	"github.com/spf13/cobra"
)

var bundleOut string

var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Pack artifacts into a single transportable archive",
	Long: `Pack all pulled images and charts from the output directory into a single
.tar.gz archive suitable for USB/SCP transfer to an air-gapped environment.

The bundle embeds a manifest.json so that 'unbundle' can push images to the
correct registry paths without needing the original packages.yaml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		p := output.New()
		p.Info(fmt.Sprintf("packing artifacts from %s → %s", outputDir, bundleOut))

		if err := bundle.Pack(cfg, outputDir, bundleOut); err != nil {
			return fmt.Errorf("packing bundle: %w", err)
		}

		p.OK(fmt.Sprintf("bundle written → %s", bundleOut))
		return nil
	},
}

func init() {
	bundleCmd.Flags().StringVar(&bundleOut, "out", "airgap-bundle.tar.gz", "output bundle file path")
}
