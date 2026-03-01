package image

import (
	"strings"
	"testing"
)

func TestRefToFilename(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{
			src:  "ghcr.io/chaos-mesh/chaos-mesh:v2.7.2",
			want: "chaos-mesh+chaos-mesh+v2.7.2.tar",
		},
		{
			src:  "ghcr.io/chaos-mesh/chaos-daemon:v2.7.2",
			want: "chaos-mesh+chaos-daemon+v2.7.2.tar",
		},
		{
			src:  "docker.io/library/nginx:latest",
			want: "library+nginx+latest.tar",
		},
		{
			src:  "falcosecurity/falco-no-driver:0.43.0",
			want: "falcosecurity+falco-no-driver+0.43.0.tar",
		},
	}

	for _, tc := range cases {
		got := RefToFilename(tc.src)
		if got != tc.want {
			t.Errorf("RefToFilename(%q) = %q, want %q", tc.src, got, tc.want)
		}
	}
}

func TestRefToFilename_noSlashOrColon(t *testing.T) {
	// Result must never contain "/" or ":" — those are unsafe on most filesystems.
	refs := []string{
		"ghcr.io/chaos-mesh/chaos-mesh:v2.7.2",
		"docker.io/library/nginx:1.25",
		"registry.k8s.io/pause:3.9",
		"quay.io/prometheus/node-exporter:v1.7.0",
	}
	for _, ref := range refs {
		got := RefToFilename(ref)
		if strings.Contains(got, "/") {
			t.Errorf("RefToFilename(%q) contains '/': %q", ref, got)
		}
		if strings.Contains(got, ":") {
			t.Errorf("RefToFilename(%q) contains ':': %q", ref, got)
		}
	}
}

func TestRefToFilename_alwaysEndsTar(t *testing.T) {
	refs := []string{
		"ghcr.io/foo/bar:v1",
		"docker.io/library/alpine:3.18",
	}
	for _, ref := range refs {
		got := RefToFilename(ref)
		if !strings.HasSuffix(got, ".tar") {
			t.Errorf("RefToFilename(%q) = %q, want .tar suffix", ref, got)
		}
	}
}

func TestRefToFilename_deterministic(t *testing.T) {
	// Same input must always produce the same output.
	ref := "ghcr.io/chaos-mesh/chaos-mesh:v2.7.2"
	first := RefToFilename(ref)
	for i := 0; i < 10; i++ {
		if got := RefToFilename(ref); got != first {
			t.Errorf("non-deterministic: run %d got %q, first was %q", i, got, first)
		}
	}
}
