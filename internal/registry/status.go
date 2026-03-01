package registry

import (
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type ImageStatus struct {
	Ref    string
	Exists bool
	Digest string
	Err    error
}

// Check probes the registry for a single image and returns its status.
// registry example: "192.168.2.2:5000"
// dest example:     "chaos-mesh/chaos-mesh:v2.7.2"
func Check(registry, dest string) ImageStatus {
	fullRef := registry + "/" + dest

	ref, err := name.NewTag(fullRef, name.Insecure)
	if err != nil {
		return ImageStatus{Ref: fullRef, Err: err}
	}

	desc, err := remote.Head(ref, remote.WithAuth(authn.Anonymous))
	if err != nil {
		return ImageStatus{Ref: fullRef, Exists: false}
	}

	return ImageStatus{
		Ref:    fullRef,
		Exists: true,
		Digest: desc.Digest.String(),
	}
}
