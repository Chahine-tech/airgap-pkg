package update

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

// ImageResult holds the update check result for a Docker image.
type ImageResult struct {
	Source  string
	Current string // current tag from the source ref
	Latest  string // latest semver tag found in the registry, or "" if unknown
	HasUpdate bool
	Err     error
}

// ChartResult holds the update check result for a Helm chart.
type ChartResult struct {
	Name      string
	Repo      string
	Current   string
	Latest    string
	HasUpdate bool
	Err       error
}

// CheckImage lists tags for the image repository and returns the latest semver
// tag compared to the current one. Non-semver tags (e.g. "latest", SHAs) are
// ignored so the result is always meaningful.
func CheckImage(src string) ImageResult {
	ref, err := name.ParseReference(src)
	if err != nil {
		return ImageResult{Source: src, Err: fmt.Errorf("parsing reference: %w", err)}
	}

	current := ref.Identifier() // tag or digest
	repo := ref.Context().String() // registry/repo without tag

	tags, err := crane.ListTags(repo, crane.WithAuth(authn.Anonymous))
	if err != nil {
		return ImageResult{Source: src, Current: current, Err: fmt.Errorf("listing tags for %s: %w", repo, err)}
	}

	latest, err := latestSemver(tags)
	if err != nil {
		// No semver tags found — not an error, just unknown.
		return ImageResult{Source: src, Current: current, Latest: ""}
	}

	currentSV, err := semver.NewVersion(current)
	if err != nil {
		// Current tag is not semver (e.g. "latest") — can't compare.
		return ImageResult{Source: src, Current: current, Latest: latest.Original()}
	}

	latestSV, _ := semver.NewVersion(latest.Original())
	return ImageResult{
		Source:    src,
		Current:   current,
		Latest:    latest.Original(),
		HasUpdate: latestSV.GreaterThan(currentSV),
	}
}

// CheckChart fetches the latest available version of a Helm chart from its
// repository and compares it to the current version.
func CheckChart(repoURL, chartName, currentVersion string) ChartResult {
	settings := cli.New()
	providers := getter.All(settings)

	// FindChartInRepoURL with version="" returns the latest available version.
	chartPath, err := repo.FindChartInRepoURL(repoURL, chartName, "", "", "", "", providers)
	if err != nil {
		return ChartResult{
			Name:    chartName,
			Repo:    repoURL,
			Current: currentVersion,
			Err:     fmt.Errorf("querying latest version: %w", err),
		}
	}

	// chartPath looks like "/tmp/.../name-version.tgz" — extract the version.
	latest := extractChartVersion(chartPath, chartName)
	if latest == "" {
		return ChartResult{Name: chartName, Repo: repoURL, Current: currentVersion, Latest: ""}
	}

	currentSV, err := semver.NewVersion(currentVersion)
	if err != nil {
		return ChartResult{Name: chartName, Repo: repoURL, Current: currentVersion, Latest: latest}
	}
	latestSV, err := semver.NewVersion(latest)
	if err != nil {
		return ChartResult{Name: chartName, Repo: repoURL, Current: currentVersion, Latest: latest}
	}

	return ChartResult{
		Name:      chartName,
		Repo:      repoURL,
		Current:   currentVersion,
		Latest:    latest,
		HasUpdate: latestSV.GreaterThan(currentSV),
	}
}

// latestSemver returns the highest semver tag from a list.
func latestSemver(tags []string) (*semver.Version, error) {
	var best *semver.Version
	for _, t := range tags {
		v, err := semver.NewVersion(t)
		if err != nil {
			continue // skip non-semver tags
		}
		if best == nil || v.GreaterThan(best) {
			best = v
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no semver tags found")
	}
	return best, nil
}

// extractChartVersion parses the version out of a chart filename like
// "/tmp/chaos-mesh-2.7.3.tgz" given the chart name "chaos-mesh".
func extractChartVersion(chartPath, chartName string) string {
	base := chartPath
	// Strip directory and extension.
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	base = strings.TrimSuffix(base, ".tgz")
	prefix := chartName + "-"
	if strings.HasPrefix(base, prefix) {
		return base[len(prefix):]
	}
	return ""
}
