package diff

import (
	"testing"

	"github.com/Chahine-tech/airgap-pkg/internal/config"
)

// cfg builds a minimal Config from inline images and charts.
func cfg(images []config.Image, charts []config.Chart) *config.Config {
	return &config.Config{
		Registry: "192.168.1.1:5000",
		Packages: []config.Package{
			{Name: "test", Images: images, Charts: charts},
		},
	}
}

func imgs(pairs ...string) []config.Image {
	var out []config.Image
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, config.Image{Source: pairs[i], Dest: pairs[i+1]})
	}
	return out
}

func chts(triples ...string) []config.Chart {
	var out []config.Chart
	for i := 0; i+2 < len(triples); i += 3 {
		out = append(out, config.Chart{Name: triples[i], Version: triples[i+1], Repo: triples[i+2]})
	}
	return out
}

func TestCompare_identical(t *testing.T) {
	a := cfg(imgs("src/img:v1", "img:v1"), chts("mychart", "1.0.0", "https://charts.example.com"))
	result := Compare(a, a)

	if result.HasChanges() {
		t.Fatal("expected no changes for identical configs")
	}
	if len(result.Images) != 1 || result.Images[0].Kind != Same {
		t.Errorf("expected 1 Same image, got %+v", result.Images)
	}
	if len(result.Charts) != 1 || result.Charts[0].Kind != Same {
		t.Errorf("expected 1 Same chart, got %+v", result.Charts)
	}
}

func TestCompare_empty(t *testing.T) {
	a := cfg(nil, nil)
	result := Compare(a, a)
	if result.HasChanges() {
		t.Fatal("empty configs should have no changes")
	}
}

func TestCompare_imageAdded(t *testing.T) {
	a := cfg(nil, nil)
	b := cfg(imgs("src/img:v1", "img:v1"), nil)
	result := Compare(a, b)

	if !result.HasChanges() {
		t.Fatal("expected changes")
	}
	if len(result.Images) != 1 {
		t.Fatalf("expected 1 image change, got %d", len(result.Images))
	}
	c := result.Images[0]
	if c.Kind != Added {
		t.Errorf("expected Added, got %s", c.Kind)
	}
	if c.Dest != "img:v1" {
		t.Errorf("unexpected dest %q", c.Dest)
	}
	if c.NewSource != "src/img:v1" {
		t.Errorf("unexpected NewSource %q", c.NewSource)
	}
}

func TestCompare_imageRemoved(t *testing.T) {
	a := cfg(imgs("src/img:v1", "img:v1"), nil)
	b := cfg(nil, nil)
	result := Compare(a, b)

	if len(result.Images) != 1 || result.Images[0].Kind != Removed {
		t.Errorf("expected Removed, got %+v", result.Images)
	}
	if result.Images[0].OldSource != "src/img:v1" {
		t.Errorf("unexpected OldSource %q", result.Images[0].OldSource)
	}
}

func TestCompare_imageUpdated(t *testing.T) {
	a := cfg(imgs("src/img:v1", "img:latest"), nil)
	b := cfg(imgs("src/img:v2", "img:latest"), nil) // same dest, different source
	result := Compare(a, b)

	if len(result.Images) != 1 || result.Images[0].Kind != Updated {
		t.Errorf("expected Updated, got %+v", result.Images)
	}
	c := result.Images[0]
	if c.OldSource != "src/img:v1" || c.NewSource != "src/img:v2" {
		t.Errorf("unexpected sources: old=%q new=%q", c.OldSource, c.NewSource)
	}
}

func TestCompare_chartUpdated(t *testing.T) {
	a := cfg(nil, chts("mychart", "1.0.0", "https://charts.example.com"))
	b := cfg(nil, chts("mychart", "2.0.0", "https://charts.example.com"))
	result := Compare(a, b)

	if len(result.Charts) != 1 || result.Charts[0].Kind != Updated {
		t.Errorf("expected Updated chart, got %+v", result.Charts)
	}
	c := result.Charts[0]
	if c.OldVersion != "1.0.0" || c.NewVersion != "2.0.0" {
		t.Errorf("unexpected versions: old=%q new=%q", c.OldVersion, c.NewVersion)
	}
}

func TestCompare_chartAddedAndRemoved(t *testing.T) {
	a := cfg(nil, chts("old-chart", "1.0.0", "https://charts.example.com"))
	b := cfg(nil, chts("new-chart", "1.0.0", "https://charts.example.com"))
	result := Compare(a, b)

	if len(result.Charts) != 2 {
		t.Fatalf("expected 2 chart changes, got %d", len(result.Charts))
	}
	// Results are sorted by name: "new-chart" < "old-chart"
	if result.Charts[0].Kind != Added || result.Charts[0].Name != "new-chart" {
		t.Errorf("expected new-chart Added, got %+v", result.Charts[0])
	}
	if result.Charts[1].Kind != Removed || result.Charts[1].Name != "old-chart" {
		t.Errorf("expected old-chart Removed, got %+v", result.Charts[1])
	}
}

func TestCompare_deterministicOrder(t *testing.T) {
	// Multiple images — result must be sorted by dest key.
	a := cfg(imgs(
		"src/z:v1", "z/img:v1",
		"src/a:v1", "a/img:v1",
		"src/m:v1", "m/img:v1",
	), nil)
	result := Compare(a, a)

	if len(result.Images) != 3 {
		t.Fatalf("expected 3, got %d", len(result.Images))
	}
	dests := []string{result.Images[0].Dest, result.Images[1].Dest, result.Images[2].Dest}
	expected := []string{"a/img:v1", "m/img:v1", "z/img:v1"}
	for i, d := range dests {
		if d != expected[i] {
			t.Errorf("position %d: got %q, want %q", i, d, expected[i])
		}
	}
}

func TestHasChanges_onlyUnchanged(t *testing.T) {
	r := &Result{
		Images: []ImageChange{{Kind: Same}},
		Charts: []ChartChange{{Kind: Same}},
	}
	if r.HasChanges() {
		t.Error("HasChanges should be false when all entries are Same")
	}
}

func TestHasChanges_mixedWithSame(t *testing.T) {
	r := &Result{
		Images: []ImageChange{{Kind: Same}, {Kind: Added}},
	}
	if !r.HasChanges() {
		t.Error("HasChanges should be true when at least one Added entry exists")
	}
}
