package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chahine-tech/airgap-pkg/internal/image"
	"github.com/Chahine-tech/airgap-pkg/pkg/output"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify [artifacts-dir]",
	Short: "Verify SHA256 digests of all artifacts in artifacts/",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := outputDir
		if len(args) > 0 {
			dir = args[0]
		}

		p := output.New()
		var failed int

		for _, subDir := range []string{"images", "charts"} {
			p.Section(subDir)
			entries, err := os.ReadDir(filepath.Join(dir, subDir))
			if err != nil {
				p.Fail(fmt.Sprintf("reading %s dir: %v", subDir, err))
				failed++
				continue
			}

			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				n := e.Name()
				if !strings.HasSuffix(n, ".tar") && !strings.HasSuffix(n, ".tgz") {
					continue
				}

				fullPath := filepath.Join(dir, subDir, n)
				digest, err := image.Verify(fullPath)
				if err != nil {
					p.Fail(fmt.Sprintf("%s: %v", n, err))
					failed++
					continue
				}
				p.OK(fmt.Sprintf("sha256:%s  %s", digest, n))
			}
		}

		if failed > 0 {
			return fmt.Errorf("%d artifact(s) failed verification", failed)
		}
		return nil
	},
}
