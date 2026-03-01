package cmd

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Chahine-tech/airgap-pkg/internal/config"
	"github.com/Chahine-tech/airgap-pkg/internal/update"
	"github.com/Chahine-tech/airgap-pkg/pkg/output"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var updateWorkers int

var updateCmd = &cobra.Command{
	Use:          "update",
	Short:        "Check for newer versions of images and charts",
	SilenceUsage: true,
	Long: `Query upstream registries and Helm repositories to check whether newer
versions are available for each image and chart declared in packages.yaml.

Exits with code 1 when at least one update is available — useful in CI to
detect stale configurations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		p := output.New()

		// Collect work items.
		type imgWork struct {
			pkg config.Package
			img config.Image
		}
		type chartWork struct {
			pkg config.Package
			ch  config.Chart
		}
		var imgs []imgWork
		var chts []chartWork
		for _, pkg := range cfg.Packages {
			for _, img := range pkg.Images {
				imgs = append(imgs, imgWork{pkg, img})
			}
			for _, ch := range pkg.Charts {
				chts = append(chts, chartWork{pkg, ch})
			}
		}

		p.Info(fmt.Sprintf("checking %d image(s) and %d chart(s) — workers: %d",
			len(imgs), len(chts), updateWorkers))

		sem := make(chan struct{}, updateWorkers)
		var hasUpdate atomic.Bool
		var mu sync.Mutex // protects p (already thread-safe) — used for section grouping

		g, _ := errgroup.WithContext(cmd.Context())

		for _, w := range imgs {
			w := w
			sem <- struct{}{}
			g.Go(func() error {
				defer func() { <-sem }()
				res := update.CheckImage(w.img.Source)
				mu.Lock()
				defer mu.Unlock()
				if res.Err != nil {
					p.Warn(fmt.Sprintf("[%s] image %s: %v", w.pkg.Name, w.img.Source, res.Err))
					return nil
				}
				if res.Latest == "" {
					p.Skip(fmt.Sprintf("[%s] %s (no semver tags found)", w.pkg.Name, w.img.Source))
					return nil
				}
				if res.HasUpdate {
					hasUpdate.Store(true)
					p.Custom("UPD", fmt.Sprintf("[%s] image %s  %s → %s",
						w.pkg.Name, w.img.Dest, res.Current, res.Latest))
				} else {
					p.OK(fmt.Sprintf("[%s] image %s (%s up-to-date)",
						w.pkg.Name, w.img.Dest, res.Current))
				}
				return nil
			})
		}

		for _, w := range chts {
			w := w
			sem <- struct{}{}
			g.Go(func() error {
				defer func() { <-sem }()
				res := update.CheckChart(w.ch.Repo, w.ch.Name, w.ch.Version)
				mu.Lock()
				defer mu.Unlock()
				if res.Err != nil {
					p.Warn(fmt.Sprintf("[%s] chart %s: %v", w.pkg.Name, w.ch.Name, res.Err))
					return nil
				}
				if res.Latest == "" {
					p.Skip(fmt.Sprintf("[%s] chart %s (version unknown)", w.pkg.Name, w.ch.Name))
					return nil
				}
				if res.HasUpdate {
					hasUpdate.Store(true)
					p.Custom("UPD", fmt.Sprintf("[%s] chart %s  %s → %s",
						w.pkg.Name, w.ch.Name, res.Current, res.Latest))
				} else {
					p.OK(fmt.Sprintf("[%s] chart %s (%s up-to-date)",
						w.pkg.Name, w.ch.Name, res.Current))
				}
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}

		if hasUpdate.Load() {
			return errors.New("updates available")
		}
		return nil
	},
}

func init() {
	updateCmd.Flags().IntVar(&updateWorkers, "workers", 4, "number of concurrent check workers")
}
