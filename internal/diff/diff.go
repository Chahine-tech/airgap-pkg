package diff

import (
	"sort"

	"github.com/Chahine-tech/airgap-pkg/internal/config"
)

// ChangeKind indicates what happened to a component between two configs.
type ChangeKind string

const (
	Added   ChangeKind = "ADD"
	Removed ChangeKind = "DEL"
	Updated ChangeKind = "UPD"
	Same    ChangeKind = "=  "
)

// ImageChange describes a change to a Docker image entry.
type ImageChange struct {
	Kind      ChangeKind
	Dest      string // keyed by dest (registry path)
	OldSource string // empty for Added
	NewSource string // empty for Removed
}

// ChartChange describes a change to a Helm chart entry.
type ChartChange struct {
	Kind       ChangeKind
	Name       string
	OldVersion string // empty for Added
	NewVersion string // empty for Removed
	Repo       string // repo of whichever side exists
}

// Result holds all changes between two configs.
type Result struct {
	Images []ImageChange
	Charts []ChartChange
}

// HasChanges reports whether any additions, removals, or updates exist.
func (r *Result) HasChanges() bool {
	for _, c := range r.Images {
		if c.Kind != Same {
			return true
		}
	}
	for _, c := range r.Charts {
		if c.Kind != Same {
			return true
		}
	}
	return false
}

// Compare returns the diff between config a (old) and config b (new).
// Images are keyed by Dest; charts are keyed by Name.
func Compare(a, b *config.Config) *Result {
	res := &Result{}

	// --- images ---
	oldImgs := imageMap(a)
	newImgs := imageMap(b)

	keys := mergeKeys(oldImgs, newImgs)
	for _, dest := range keys {
		old, inOld := oldImgs[dest]
		nw, inNew := newImgs[dest]
		switch {
		case inOld && !inNew:
			res.Images = append(res.Images, ImageChange{Kind: Removed, Dest: dest, OldSource: old})
		case !inOld && inNew:
			res.Images = append(res.Images, ImageChange{Kind: Added, Dest: dest, NewSource: nw})
		case old != nw:
			res.Images = append(res.Images, ImageChange{Kind: Updated, Dest: dest, OldSource: old, NewSource: nw})
		default:
			res.Images = append(res.Images, ImageChange{Kind: Same, Dest: dest, OldSource: old, NewSource: nw})
		}
	}

	// --- charts ---
	oldCharts := chartMap(a)
	newCharts := chartMap(b)

	ckeys := mergeKeys(oldCharts, newCharts)
	for _, name := range ckeys {
		old, inOld := oldCharts[name]
		nw, inNew := newCharts[name]
		switch {
		case inOld && !inNew:
			res.Charts = append(res.Charts, ChartChange{Kind: Removed, Name: name, OldVersion: old.version, Repo: old.repo})
		case !inOld && inNew:
			res.Charts = append(res.Charts, ChartChange{Kind: Added, Name: name, NewVersion: nw.version, Repo: nw.repo})
		case old.version != nw.version:
			res.Charts = append(res.Charts, ChartChange{Kind: Updated, Name: name, OldVersion: old.version, NewVersion: nw.version, Repo: nw.repo})
		default:
			res.Charts = append(res.Charts, ChartChange{Kind: Same, Name: name, OldVersion: old.version, NewVersion: nw.version, Repo: nw.repo})
		}
	}

	return res
}

// imageMap returns dest → source for all images across all packages.
func imageMap(cfg *config.Config) map[string]string {
	m := make(map[string]string)
	for _, pkg := range cfg.Packages {
		for _, img := range pkg.Images {
			m[img.Dest] = img.Source
		}
	}
	return m
}

type chartEntry struct{ version, repo string }

// chartMap returns name → {version, repo} for all charts across all packages.
func chartMap(cfg *config.Config) map[string]chartEntry {
	m := make(map[string]chartEntry)
	for _, pkg := range cfg.Packages {
		for _, ch := range pkg.Charts {
			m[ch.Name] = chartEntry{version: ch.Version, repo: ch.Repo}
		}
	}
	return m
}

// mergeKeys returns the sorted union of keys from two maps.
func mergeKeys[V any](a, b map[string]V) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
