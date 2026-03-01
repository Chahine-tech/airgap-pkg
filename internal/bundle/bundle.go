package bundle

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Chahine-tech/airgap-pkg/internal/config"
	"github.com/Chahine-tech/airgap-pkg/internal/image"
)

const manifestName = "manifest.json"

// Manifest is embedded in the bundle and drives unbundle.
type Manifest struct {
	CreatedAt string          `json:"created_at"` // RFC3339 UTC
	Registry  string          `json:"registry"`
	Images    []ManifestImage `json:"images"`
	Charts    []ManifestChart `json:"charts"`
}

// ManifestImage maps the tarball filename to its destination path in the registry.
type ManifestImage struct {
	Tarball string `json:"tarball"` // relative path inside bundle, e.g. images/foo.tar
	Dest    string `json:"dest"`    // registry destination, e.g. chaos-mesh/chaos-mesh:v2.7.2
}

// ManifestChart records a chart tarball for reference.
type ManifestChart struct {
	Tarball string `json:"tarball"` // relative path inside bundle, e.g. charts/chaos-mesh-2.7.2.tgz
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Pack creates a .tar.gz bundle at destPath from the artifacts in artifactsDir.
// cfg is used to build the manifest (image dest paths, chart metadata).
func Pack(cfg *config.Config, artifactsDir, destPath string) error {
	imagesDir := filepath.Join(artifactsDir, "images")
	chartsDir := filepath.Join(artifactsDir, "charts")

	// Build manifest.
	manifest := Manifest{
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Registry:  cfg.Registry,
	}

	for _, pkg := range cfg.Packages {
		for _, img := range pkg.Images {
			tarball := filepath.Join("images", image.RefToFilename(img.Source))
			manifest.Images = append(manifest.Images, ManifestImage{
				Tarball: tarball,
				Dest:    img.Dest,
			})
		}
		for _, ch := range pkg.Charts {
			tarball := filepath.Join("charts", fmt.Sprintf("%s-%s.tgz", ch.Name, ch.Version))
			manifest.Charts = append(manifest.Charts, ManifestChart{
				Tarball: tarball,
				Name:    ch.Name,
				Version: ch.Version,
			})
		}
	}

	// Create output file.
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("creating bundle directory: %w", err)
	}
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating bundle file: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Write manifest.json.
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	if err := writeBytesToTar(tw, manifestName, manifestData); err != nil {
		return fmt.Errorf("writing manifest to bundle: %w", err)
	}

	// Write images.
	if err := addDirToTar(tw, imagesDir, "images"); err != nil {
		return fmt.Errorf("adding images to bundle: %w", err)
	}

	// Write charts.
	if err := addDirToTar(tw, chartsDir, "charts"); err != nil {
		return fmt.Errorf("adding charts to bundle: %w", err)
	}

	return nil
}

// Unpack extracts a bundle .tar.gz to destDir and returns the embedded Manifest.
func Unpack(bundlePath, destDir string) (*Manifest, error) {
	f, err := os.Open(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("opening bundle: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("reading gzip bundle: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("creating dest dir: %w", err)
	}

	var manifest *Manifest

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading bundle: %w", err)
		}

		// Security: prevent path traversal.
		target := filepath.Join(destDir, filepath.Clean("/"+hdr.Name))

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return nil, fmt.Errorf("creating dir %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return nil, fmt.Errorf("creating parent dir: %w", err)
			}
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("reading entry %s: %w", hdr.Name, err)
			}
			if err := os.WriteFile(target, data, 0644); err != nil {
				return nil, fmt.Errorf("writing %s: %w", target, err)
			}
			// Parse manifest when encountered.
			if hdr.Name == manifestName {
				var m Manifest
				if err := json.Unmarshal(data, &m); err != nil {
					return nil, fmt.Errorf("parsing manifest: %w", err)
				}
				manifest = &m
			}
		}
	}

	if manifest == nil {
		return nil, fmt.Errorf("bundle has no %s — may be corrupt", manifestName)
	}
	return manifest, nil
}

// addDirToTar walks srcDir and adds all files under the tarPrefix directory.
// Missing source directory is silently skipped (pull may not have produced charts).
func addDirToTar(tw *tar.Writer, srcDir, tarPrefix string) error {
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return nil
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(srcDir, e.Name())
		if err := addFileToTar(tw, path, filepath.Join(tarPrefix, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func addFileToTar(tw *tar.Writer, srcPath, tarName string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	hdr := &tar.Header{
		Name:    tarName,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}

func writeBytesToTar(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{
		Name:    name,
		Size:    int64(len(data)),
		Mode:    0644,
		ModTime: time.Now().UTC(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

