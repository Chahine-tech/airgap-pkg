package cmd

import (
	"errors"
	"fmt"

	"github.com/Chahine-tech/airgap-pkg/internal/config"
	"github.com/Chahine-tech/airgap-pkg/internal/diff"
	"github.com/Chahine-tech/airgap-pkg/pkg/output"
	"github.com/spf13/cobra"
)

var diffShowAll bool

var diffCmd = &cobra.Command{
	Use:          "diff <old-config> <new-config>",
	Short:        "Show changes between two packages.yaml files",
	Args:         cobra.ExactArgs(2),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgA, err := config.Load(args[0])
		if err != nil {
			return fmt.Errorf("loading %s: %w", args[0], err)
		}
		cfgB, err := config.Load(args[1])
		if err != nil {
			return fmt.Errorf("loading %s: %w", args[1], err)
		}

		result := diff.Compare(cfgA, cfgB)
		p := output.New()

		p.Section("Images")
		for _, c := range result.Images {
			if c.Kind == diff.Same && !diffShowAll {
				continue
			}
			switch c.Kind {
			case diff.Added:
				p.Custom("ADD", fmt.Sprintf("%s  ← %s", c.Dest, c.NewSource))
			case diff.Removed:
				p.Custom("DEL", fmt.Sprintf("%s  (was %s)", c.Dest, c.OldSource))
			case diff.Updated:
				p.Custom("UPD", fmt.Sprintf("%s  %s → %s", c.Dest, c.OldSource, c.NewSource))
			case diff.Same:
				p.Custom("=  ", fmt.Sprintf("%s", c.Dest))
			}
		}

		p.Section("Charts")
		for _, c := range result.Charts {
			if c.Kind == diff.Same && !diffShowAll {
				continue
			}
			switch c.Kind {
			case diff.Added:
				p.Custom("ADD", fmt.Sprintf("%s@%s", c.Name, c.NewVersion))
			case diff.Removed:
				p.Custom("DEL", fmt.Sprintf("%s@%s", c.Name, c.OldVersion))
			case diff.Updated:
				p.Custom("UPD", fmt.Sprintf("%s  %s → %s", c.Name, c.OldVersion, c.NewVersion))
			case diff.Same:
				p.Custom("=  ", fmt.Sprintf("%s@%s", c.Name, c.OldVersion))
			}
		}

		if result.HasChanges() {
			return errors.New("differences found")
		}
		return nil
	},
}

func init() {
	diffCmd.Flags().BoolVar(&diffShowAll, "all", false, "also show unchanged entries")
}
