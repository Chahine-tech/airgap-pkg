package image

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestVerify_knownContent(t *testing.T) {
	content := []byte("hello airgap")
	expected := sha256hex(content)

	f, err := os.CreateTemp(t.TempDir(), "*.tar")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	got, err := Verify(f.Name())
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if got != expected {
		t.Errorf("Verify = %q, want %q", got, expected)
	}
}

func TestVerify_emptyFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.tar")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	// SHA256 of empty input is well-known.
	const emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	got, err := Verify(f.Name())
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if got != emptySHA256 {
		t.Errorf("Verify(empty) = %q, want %q", got, emptySHA256)
	}
}

func TestVerify_deterministicOutput(t *testing.T) {
	content := []byte("deterministic content")
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tar")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	first, err := Verify(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		got, err := Verify(path)
		if err != nil {
			t.Fatal(err)
		}
		if got != first {
			t.Errorf("non-deterministic: run %d got %q, first was %q", i, got, first)
		}
	}
}

func TestVerify_fileNotFound(t *testing.T) {
	_, err := Verify("/nonexistent/path/file.tar")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestVerify_hexLength(t *testing.T) {
	// SHA256 hex digest is always 64 characters.
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tar")
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := Verify(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 64 {
		t.Errorf("expected 64-char hex digest, got %d chars: %q", len(got), got)
	}
}

func sha256hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
