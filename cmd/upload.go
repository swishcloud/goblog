package cmd

import (
	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "upload file to cloud",
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}
