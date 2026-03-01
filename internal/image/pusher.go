package image

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
)

// Push loads a tarball from tarPath and pushes it to registry/dest over plain HTTP.
// registry example: "192.168.2.2:5000"
// dest example:     "chaos-mesh/chaos-mesh:v2.7.2"
func Push(tarPath, registry, dest string) error {
	destRef := registry + "/" + dest

	ref, err := name.NewTag(destRef, name.Insecure)
	if err != nil {
		return fmt.Errorf("parsing dest ref %s: %w", destRef, err)
	}

	img, err := v1tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return fmt.Errorf("loading tarball %s: %w", tarPath, err)
	}

	if err := crane.Push(img, ref.String(), crane.Insecure); err != nil {
		return fmt.Errorf("pushing %s: %w", ref.String(), err)
	}

	return nil
}
