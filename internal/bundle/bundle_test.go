package bundle

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/Chahine-tech/airgap-pkg/internal/config"
)

func testCfg() *config.Config {
	return &config.Config{
		Registry: "192.168.1.1:5000",
		Packages: []config.Package{
			{
				Name: "test-pkg",
				Images: []config.Image{
					{Source: "ghcr.io/chaos-mesh/chaos-mesh:v2.7.2", Dest: "chaos-mesh/chaos-mesh:v2.7.2"},
				},
				Charts: []config.Chart{
					{Name: "chaos-mesh", Version: "2.7.2", Repo: "https://charts.chaos-mesh.org"},
				},
			},
		},
	}
}

// setupArtifacts creates fake image and chart tarballs so Pack() has real files.
func setupArtifacts(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	imagesDir := filepath.Join(dir, "images")
	chartsDir := filepath.Join(dir, "charts")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(chartsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Filename must match image.RefToFilename("ghcr.io/chaos-mesh/chaos-mesh:v2.7.2")
	imgFile := filepath.Join(imagesDir, "chaos-mesh+chaos-mesh+v2.7.2.tar")
	if err := os.WriteFile(imgFile, []byte("fake image tar content"), 0644); err != nil {
		t.Fatal(err)
	}

	chartFile := filepath.Join(chartsDir, "chaos-mesh-2.7.2.tgz")
	if err := os.WriteFile(chartFile, []byte("fake chart tgz content"), 0644); err != nil {
		t.Fatal(err)
	}

	return dir
}

// writeFakeTarGz writes a minimal .tar.gz bundle with the given file map.
func writeFakeTarGz(t *testing.T, path string, files map[string][]byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	for name, data := range files {
		hdr := &tar.Header{Name: name, Size: int64(len(data)), Mode: 0644}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPackUnpack_roundtrip(t *testing.T) {
	cfg := testCfg()
	artifactsDir := setupArtifacts(t)
	bundlePath := filepath.Join(t.TempDir(), "test.tar.gz")

	if err := Pack(cfg, artifactsDir, bundlePath); err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	fi, err := os.Stat(bundlePath)
	if err != nil {
		t.Fatalf("bundle not created: %v", err)
	}
	if fi.Size() == 0 {
		t.Fatal("bundle file is empty")
	}

	extractDir := t.TempDir()
	manifest, err := Unpack(bundlePath, extractDir)
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}

	if manifest.Registry != "192.168.1.1:5000" {
		t.Errorf("Registry = %q", manifest.Registry)
	}
	if manifest.CreatedAt == "" {
		t.Error("CreatedAt should not be empty")
	}
	if len(manifest.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(manifest.Images))
	}
	if manifest.Images[0].Dest != "chaos-mesh/chaos-mesh:v2.7.2" {
		t.Errorf("image Dest = %q", manifest.Images[0].Dest)
	}
	if len(manifest.Charts) != 1 {
		t.Fatalf("expected 1 chart, got %d", len(manifest.Charts))
	}
	if manifest.Charts[0].Name != "chaos-mesh" || manifest.Charts[0].Version != "2.7.2" {
		t.Errorf("chart = %+v", manifest.Charts[0])
	}

	extractedImg := filepath.Join(extractDir, "images", "chaos-mesh+chaos-mesh+v2.7.2.tar")
	if _, err := os.Stat(extractedImg); err != nil {
		t.Errorf("extracted image not found: %v", err)
	}
	extractedChart := filepath.Join(extractDir, "charts", "chaos-mesh-2.7.2.tgz")
	if _, err := os.Stat(extractedChart); err != nil {
		t.Errorf("extracted chart not found: %v", err)
	}
}

func TestPack_missingArtifactDirs_ok(t *testing.T) {
	// Pack should succeed even when images/ and charts/ don't exist.
	cfg := testCfg()
	emptyDir := t.TempDir()
	bundlePath := filepath.Join(t.TempDir(), "empty.tar.gz")

	if err := Pack(cfg, emptyDir, bundlePath); err != nil {
		t.Fatalf("Pack with missing artifact dirs should not fail: %v", err)
	}

	manifest, err := Unpack(bundlePath, t.TempDir())
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}
	if manifest.Registry != "192.168.1.1:5000" {
		t.Errorf("Registry = %q", manifest.Registry)
	}
}

func TestUnpack_missingManifest(t *testing.T) {
	bundlePath := filepath.Join(t.TempDir(), "bad.tar.gz")
	writeFakeTarGz(t, bundlePath, map[string][]byte{
		"images/dummy.tar": []byte("data"),
	})

	_, err := Unpack(bundlePath, t.TempDir())
	if err == nil {
		t.Fatal("expected error for bundle without manifest.json")
	}
}

func TestUnpack_pathTraversal(t *testing.T) {
	bundlePath := filepath.Join(t.TempDir(), "evil.tar.gz")
	writeFakeTarGz(t, bundlePath, map[string][]byte{
		manifestName:        []byte(`{"registry":"r","images":[],"charts":[],"created_at":"2025-01-01T00:00:00Z"}`),
		"../../../evil.txt": []byte("pwned"),
	})

	extractDir := t.TempDir()
	if _, err := Unpack(bundlePath, extractDir); err != nil {
		// Erroring on traversal is also acceptable.
		return
	}

	// The evil file must NOT have escaped extractDir.
	evilPath := filepath.Join(filepath.Dir(extractDir), "evil.txt")
	if _, err := os.Stat(evilPath); err == nil {
		t.Fatal("path traversal succeeded — security bug!")
	}
}
