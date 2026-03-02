package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Pull, verify, and push in one step",
	Long: `sync runs pull → verify → push in sequence.
It stops immediately if any step fails.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := pullCmd.RunE(cmd, args); err != nil {
			return fmt.Errorf("pull: %w", err)
		}
		if err := verifyCmd.RunE(cmd, args); err != nil {
			return fmt.Errorf("verify: %w", err)
		}
		if err := pushCmd.RunE(cmd, args); err != nil {
			return fmt.Errorf("push: %w", err)
		}
		return nil
	},
}

func init() {
	// Inherit push flags so --registry, --via-ssh etc. work with sync too.
	syncCmd.Flags().AddFlagSet(pushCmd.Flags())
}
