package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/Chahine-tech/airgap-pkg/internal/config"
	"github.com/Chahine-tech/airgap-pkg/internal/hooks"
	"github.com/Chahine-tech/airgap-pkg/internal/image"
	"github.com/Chahine-tech/airgap-pkg/internal/transport"
	"github.com/Chahine-tech/airgap-pkg/pkg/output"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	pushRegistry string
	pushViaSSH   string
	pushSSHKey   string
	pushSSHUser  string
	pushSSHPort  string
	pushWorkers  int
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push all images to the internal registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		registry := pushRegistry
		if registry == "" {
			registry = cfg.Registry
		}
		if registry == "" {
			return fmt.Errorf("no registry specified: set 'registry' in packages.yaml or use --registry")
		}

		p := output.New()

		// SSH tunnel setup — flag > config > default precedence
		sshHost := pushViaSSH
		if sshHost == "" {
			sshHost = cfg.Transit.Host
		}

		if sshHost != "" {
			sshKey := pushSSHKey
			if sshKey == "" {
				sshKey = cfg.Transit.SSHKey
			}
			if sshKey == "" {
				sshKey = "~/.ssh/id_rsa"
			}

			sshUser := pushSSHUser
			if sshUser == "" {
				sshUser = cfg.Transit.User
			}
			if sshUser == "" {
				sshUser = os.Getenv("USER")
			}

			sshPort := pushSSHPort
			if sshPort == "" {
				sshPort = cfg.Transit.Port
			}
			if sshPort == "" {
				sshPort = "22"
			}

			sshCfg := transport.SSHConfig{
				Host:    sshHost,
				Port:    sshPort,
				User:    sshUser,
				KeyPath: sshKey,
			}

			p.Info(fmt.Sprintf("opening SSH tunnel via %s@%s:%s → %s", sshUser, sshHost, sshPort, registry))

			localAddr, closeTunnel, err := transport.Tunnel(sshCfg, registry)
			if err != nil {
				return fmt.Errorf("establishing SSH tunnel: %w", err)
			}
			defer closeTunnel()

			p.OK(fmt.Sprintf("tunnel ready: local %s → remote %s", localAddr, registry))
			registry = localAddr
		}

		imagesDir := filepath.Join(outputDir, "images")

		// Collect all work items.
		type pushWork struct {
			pkg config.Package
			img config.Image
		}
		var work []pushWork
		for _, pkg := range cfg.Packages {
			for _, img := range pkg.Images {
				work = append(work, pushWork{pkg, img})
			}
		}

		p.Info(fmt.Sprintf("pushing %d image(s) → %s — workers: %d", len(work), registry, pushWorkers))

		sem := make(chan struct{}, pushWorkers)
		var failed atomic.Int32

		g, _ := errgroup.WithContext(cmd.Context())

		for _, w := range work {
			w := w
			sem <- struct{}{}
			g.Go(func() error {
				defer func() { <-sem }()

				filename := image.RefToFilename(w.img.Source)
				tarPath := filepath.Join(imagesDir, filename)

				if _, err := os.Stat(tarPath); os.IsNotExist(err) {
					p.Fail(fmt.Sprintf("[%s] tarball not found for %s (run pull first)", w.pkg.Name, w.img.Source))
					failed.Add(1)
					return nil
				}

				if err := hooks.Run(cfg.Hooks.PrePush, map[string]string{
					"Source":   w.img.Source,
					"Path":     tarPath,
					"Dest":     w.img.Dest,
					"Registry": registry,
				}); err != nil {
					p.Warn(fmt.Sprintf("[%s] pre_push hook failed: %v", w.pkg.Name, err))
				}
				p.Info(fmt.Sprintf("[%s] pushing %s → %s/%s", w.pkg.Name, filename, registry, w.img.Dest))
				if err := image.Push(tarPath, registry, w.img.Dest); err != nil {
					p.Fail(fmt.Sprintf("[%s] %s: %v", w.pkg.Name, w.img.Dest, err))
					failed.Add(1)
					return nil
				}
				p.OK(fmt.Sprintf("[%s] pushed → %s/%s", w.pkg.Name, registry, w.img.Dest))
				if err := hooks.Run(cfg.Hooks.PostPush, map[string]string{
					"Source":   w.img.Source,
					"Path":     tarPath,
					"Dest":     w.img.Dest,
					"Registry": registry,
				}); err != nil {
					p.Warn(fmt.Sprintf("[%s] post_push hook failed: %v", w.pkg.Name, err))
				}
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}

		n := int(failed.Load())
		if n > 0 {
			return fmt.Errorf("%d/%d image(s) failed to push", n, len(work))
		}
		return nil
	},
}

func init() {
	pushCmd.Flags().StringVar(&pushRegistry, "registry", "", "override registry from packages.yaml (e.g. 192.168.2.2:5000)")
	pushCmd.Flags().StringVar(&pushViaSSH, "via-ssh", "", "SSH host to tunnel through (e.g. node-1)")
	pushCmd.Flags().StringVar(&pushSSHKey, "ssh-key", "", "path to SSH private key (default: transit.ssh_key from config, then ~/.ssh/id_rsa)")
	pushCmd.Flags().StringVar(&pushSSHUser, "ssh-user", "", "SSH user (default: transit.user from config, then $USER)")
	pushCmd.Flags().StringVar(&pushSSHPort, "ssh-port", "", "SSH port (default: transit.port from config, then 22)")
	pushCmd.Flags().IntVar(&pushWorkers, "workers", 2, "number of concurrent push workers")
}
