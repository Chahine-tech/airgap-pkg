package image

import (
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// RefToFilename converts an image reference into a filesystem-safe filename.
// Example: "ghcr.io/chaos-mesh/chaos-mesh:v2.7.2" → "chaos-mesh+chaos-mesh+v2.7.2.tar"
// The registry prefix is stripped, then "/" and ":" are replaced by "+".
func RefToFilename(src string) string {
	ref, err := name.ParseReference(src)
	if err != nil {
		// Fallback: simple sanitization
		safe := strings.NewReplacer("/", "+", ":", "+", ".", "-").Replace(src)
		return safe + ".tar"
	}
	repo := ref.Context().RepositoryStr() // e.g. "chaos-mesh/chaos-mesh"
	tag := ref.Identifier()               // e.g. "v2.7.2" or sha256:...
	safe := strings.ReplaceAll(repo, "/", "+")
	return safe + "+" + tag + ".tar"
}
