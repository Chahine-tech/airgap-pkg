package image

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

// Pull pulls a single image from src and writes it as a Docker-compatible tarball to destDir.
// Returns the path to the written file.
func Pull(src, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("creating images dir: %w", err)
	}

	img, err := crane.Pull(src)
	if err != nil {
		return "", fmt.Errorf("pulling %s: %w", src, err)
	}

	ref, err := name.ParseReference(src)
	if err != nil {
		return "", fmt.Errorf("parsing reference %s: %w", src, err)
	}

	outPath := filepath.Join(destDir, RefToFilename(src))

	if err := v1tarball.WriteToFile(outPath, ref, img); err != nil {
		return "", fmt.Errorf("writing tarball %s: %w", outPath, err)
	}

	return outPath, nil
}
