package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Chahine-tech/airgap-pkg/internal/config"
	"github.com/Chahine-tech/airgap-pkg/internal/sbom"
	"github.com/Chahine-tech/airgap-pkg/pkg/output"
	"github.com/spf13/cobra"
)

var (
	sbomFormat string
	sbomOut    string
)

var sbomCmd = &cobra.Command{
	Use:   "sbom",
	Short: "Generate a Software Bill of Materials from artifacts",
	Long: `Generate a Software Bill of Materials (SBOM) from the artifacts already
pulled into the output directory. Reads packages.yaml for the full component
list and computes SHA256 digests from the tarballs on disk.

Missing tarballs (pull not yet run) appear with sha256:"NOT_FOUND".`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		imagesDir := filepath.Join(outputDir, "images")
		chartsDir := filepath.Join(outputDir, "charts")

		s, err := sbom.Generate(cfg, imagesDir, chartsDir)
		if err != nil {
			return fmt.Errorf("generating SBOM: %w", err)
		}

		// Warnings go to stderr so stdout stays valid JSON when piping.
		warn := output.NewTo(os.Stderr)
		for _, c := range s.Components {
			if c.SHA256 == "NOT_FOUND" {
				warn.Fail(fmt.Sprintf("%s not found on disk (run pull first): %s", c.Type, c.Tarball))
			}
		}

		// Marshal to the requested format.
		var data []byte
		switch sbomFormat {
		case "cyclonedx":
			bom := sbom.ToCycloneDX(s)
			data, err = json.MarshalIndent(bom, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling CycloneDX SBOM: %w", err)
			}
		default: // "json"
			data, err = json.MarshalIndent(s, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling SBOM: %w", err)
			}
		}

		// Write to file or stdout.
		if sbomOut == "" {
			fmt.Println(string(data))
			return nil
		}

		if err := os.WriteFile(sbomOut, data, 0644); err != nil {
			return fmt.Errorf("writing SBOM to %s: %w", sbomOut, err)
		}
		output.New().OK(fmt.Sprintf("SBOM written → %s", sbomOut))
		return nil
	},
}

func init() {
	sbomCmd.Flags().StringVar(&sbomFormat, "format", "json", `output format: "json" or "cyclonedx"`)
	sbomCmd.Flags().StringVar(&sbomOut, "out", "", "write SBOM to file (default: stdout)")
}
