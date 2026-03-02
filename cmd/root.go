package cmd

import (
	"github.com/spf13/cobra"
)

var (
	configFile string
	outputDir  string
)

var rootCmd = &cobra.Command{
	Use:   "airgap-pkg",
	Short: "Package Docker images and Helm charts for airgap deployments",
	Long: `airgap-pkg automates the workflow of pulling Docker images and Helm charts
from internet-connected environments, verifying them, and pushing them to
internal registries in air-gapped environments.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "packages.yaml", "path to packages.yaml")
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "./artifacts", "output directory for artifacts")
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(sbomCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(bundleCmd)
	rootCmd.AddCommand(unbundleCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(syncCmd)
}
