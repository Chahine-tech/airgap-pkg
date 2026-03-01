package cmd

import (
	"fmt"
	"path/filepath"
	"sync/atomic"

	"github.com/Chahine-tech/airgap-pkg/internal/chart"
	"github.com/Chahine-tech/airgap-pkg/internal/config"
	"github.com/Chahine-tech/airgap-pkg/internal/image"
	"github.com/Chahine-tech/airgap-pkg/pkg/output"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var pullWorkers int

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull all images and charts defined in packages.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		p := output.New()
		imagesDir := filepath.Join(outputDir, "images")
		chartsDir := filepath.Join(outputDir, "charts")

		// Collect all work items upfront.
		type imageWork struct {
			pkg config.Package
			img config.Image
		}
		type chartWork struct {
			pkg config.Package
			ch  config.Chart
		}

		var imgs []imageWork
		var chts []chartWork
		for _, pkg := range cfg.Packages {
			for _, img := range pkg.Images {
				imgs = append(imgs, imageWork{pkg, img})
			}
			for _, ch := range pkg.Charts {
				chts = append(chts, chartWork{pkg, ch})
			}
		}

		total := len(imgs) + len(chts)
		p.Info(fmt.Sprintf("pulling %d image(s) and %d chart(s) — workers: %d", len(imgs), len(chts), pullWorkers))

		// Semaphore limits concurrency to pullWorkers goroutines.
		sem := make(chan struct{}, pullWorkers)
		var failed atomic.Int32

		g, _ := errgroup.WithContext(cmd.Context())

		for _, w := range imgs {
			w := w
			sem <- struct{}{}
			g.Go(func() error {
				defer func() { <-sem }()
				p.Info(fmt.Sprintf("[%s] pulling image %s", w.pkg.Name, w.img.Source))
				path, err := image.Pull(w.img.Source, imagesDir)
				if err != nil {
					p.Fail(fmt.Sprintf("[%s] image %s: %v", w.pkg.Name, w.img.Source, err))
					failed.Add(1)
					return nil // don't abort siblings on one failure
				}
				p.OK(fmt.Sprintf("[%s] image saved → %s", w.pkg.Name, path))
				return nil
			})
		}

		for _, w := range chts {
			w := w
			sem <- struct{}{}
			g.Go(func() error {
				defer func() { <-sem }()
				p.Info(fmt.Sprintf("[%s] pulling chart %s@%s", w.pkg.Name, w.ch.Name, w.ch.Version))
				path, err := chart.Pull(w.ch.Repo, w.ch.Name, w.ch.Version, chartsDir)
				if err != nil {
					p.Fail(fmt.Sprintf("[%s] chart %s@%s: %v", w.pkg.Name, w.ch.Name, w.ch.Version, err))
					failed.Add(1)
					return nil
				}
				p.OK(fmt.Sprintf("[%s] chart saved → %s", w.pkg.Name, path))
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}

		n := int(failed.Load())
		if n > 0 {
			return fmt.Errorf("%d/%d artifact(s) failed to pull", n, total)
		}
		return nil
	},
}

func init() {
	pullCmd.Flags().IntVar(&pullWorkers, "workers", 4, "number of concurrent pull workers")
}
