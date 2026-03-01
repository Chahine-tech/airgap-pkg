package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/Chahine-tech/airgap-pkg/internal/bundle"
	"github.com/Chahine-tech/airgap-pkg/internal/image"
	"github.com/Chahine-tech/airgap-pkg/internal/transport"
	"github.com/Chahine-tech/airgap-pkg/pkg/output"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	unbundleRegistry string
	unbundleViaSSH   string
	unbundleSSHKey   string
	unbundleSSHUser  string
	unbundleSSHPort  string
	unbundleWorkers  int
	unbundleExtract  string
)

var unbundleCmd = &cobra.Command{
	Use:   "unbundle <bundle.tar.gz>",
	Short: "Extract a bundle and push images to the internal registry",
	Long: `Extract a bundle produced by 'bundle' and push all images to the registry
declared in the embedded manifest (or overridden with --registry).

The bundle is extracted to a temporary directory (or --extract if specified),
then each image tarball is pushed concurrently.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bundlePath := args[0]
		p := output.New()

		// Determine extraction directory.
		extractDir := unbundleExtract
		if extractDir == "" {
			tmp, err := os.MkdirTemp("", "airgap-unbundle-*")
			if err != nil {
				return fmt.Errorf("creating temp dir: %w", err)
			}
			defer os.RemoveAll(tmp)
			extractDir = tmp
		}

		p.Info(fmt.Sprintf("extracting %s → %s", bundlePath, extractDir))
		manifest, err := bundle.Unpack(bundlePath, extractDir)
		if err != nil {
			return fmt.Errorf("unpacking bundle: %w", err)
		}
		p.OK(fmt.Sprintf("extracted %d image(s) and %d chart(s)", len(manifest.Images), len(manifest.Charts)))

		// Resolve registry: flag > manifest.
		registry := unbundleRegistry
		if registry == "" {
			registry = manifest.Registry
		}
		if registry == "" {
			return fmt.Errorf("no registry: set 'registry' in the bundle manifest or use --registry")
		}

		// SSH tunnel — same flag set as push.
		if unbundleViaSSH != "" {
			sshKey := unbundleSSHKey
			if sshKey == "" {
				sshKey = "~/.ssh/id_rsa"
			}
			sshUser := unbundleSSHUser
			if sshUser == "" {
				sshUser = os.Getenv("USER")
			}
			sshPort := unbundleSSHPort
			if sshPort == "" {
				sshPort = "22"
			}

			sshCfg := transport.SSHConfig{
				Host:    unbundleViaSSH,
				Port:    sshPort,
				User:    sshUser,
				KeyPath: sshKey,
			}

			p.Info(fmt.Sprintf("opening SSH tunnel via %s@%s:%s → %s", sshUser, unbundleViaSSH, sshPort, registry))
			localAddr, closeTunnel, err := transport.Tunnel(sshCfg, registry)
			if err != nil {
				return fmt.Errorf("establishing SSH tunnel: %w", err)
			}
			defer closeTunnel()
			p.OK(fmt.Sprintf("tunnel ready: local %s → remote %s", localAddr, registry))
			registry = localAddr
		}

		p.Info(fmt.Sprintf("pushing %d image(s) → %s — workers: %d", len(manifest.Images), registry, unbundleWorkers))

		sem := make(chan struct{}, unbundleWorkers)
		var failed atomic.Int32

		g, _ := errgroup.WithContext(cmd.Context())

		for _, img := range manifest.Images {
			img := img
			sem <- struct{}{}
			g.Go(func() error {
				defer func() { <-sem }()
				tarPath := filepath.Join(extractDir, filepath.Clean(img.Tarball))
				p.Info(fmt.Sprintf("pushing %s → %s/%s", filepath.Base(tarPath), registry, img.Dest))
				if err := image.Push(tarPath, registry, img.Dest); err != nil {
					p.Fail(fmt.Sprintf("%s: %v", img.Dest, err))
					failed.Add(1)
					return nil
				}
				p.OK(fmt.Sprintf("pushed → %s/%s", registry, img.Dest))
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}

		n := int(failed.Load())
		if n > 0 {
			return fmt.Errorf("%d/%d image(s) failed to push", n, len(manifest.Images))
		}
		return nil
	},
}

func init() {
	unbundleCmd.Flags().StringVar(&unbundleRegistry, "registry", "", "override registry from bundle manifest")
	unbundleCmd.Flags().StringVar(&unbundleViaSSH, "via-ssh", "", "SSH host to tunnel through")
	unbundleCmd.Flags().StringVar(&unbundleSSHKey, "ssh-key", "", "path to SSH private key (default: ~/.ssh/id_rsa)")
	unbundleCmd.Flags().StringVar(&unbundleSSHUser, "ssh-user", "", "SSH user (default: $USER)")
	unbundleCmd.Flags().StringVar(&unbundleSSHPort, "ssh-port", "", "SSH port (default: 22)")
	unbundleCmd.Flags().IntVar(&unbundleWorkers, "workers", 2, "number of concurrent push workers")
	unbundleCmd.Flags().StringVar(&unbundleExtract, "extract", "", "extract to this directory instead of a temp dir (kept after exit)")
}
