package main

import (
	"os"

	"github.com/Chahine-tech/airgap-pkg/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
