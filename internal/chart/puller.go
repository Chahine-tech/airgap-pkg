package chart

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

// Pull downloads a Helm chart tarball to destDir.
// Returns the path to the downloaded .tgz file.
func Pull(repoURL, chartName, version, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("creating charts dir: %w", err)
	}

	settings := cli.New()
	providers := getter.All(settings)

	chartPath, err := repo.FindChartInRepoURL(
		repoURL, chartName, version,
		"", "", "", // certFile, keyFile, caFile — no TLS client auth
		providers,
	)
	if err != nil {
		return "", fmt.Errorf("finding chart %s/%s@%s: %w", repoURL, chartName, version, err)
	}

	destFile := filepath.Join(destDir, filepath.Base(chartPath))
	if err := copyFile(chartPath, destFile); err != nil {
		return "", fmt.Errorf("copying chart to artifacts: %w", err)
	}

	// Clean up the temp file helm wrote
	_ = os.Remove(chartPath)

	return destFile, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
