package main

import (
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:        "doctor",
	Short:      "Deprecated: use 'devkit status' instead",
	Deprecated: "use 'devkit status' instead",
	RunE:       runStatus,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
