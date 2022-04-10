package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/swishcloud/goblog/internal"
)

var rootCmd = &cobra.Command{
	Use: "goblog",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("welcome")
	},
}

func Execute() {
	defer func() {
		if err := recover(); err != nil {
			if internal.Logger == nil {
				panic(err)
				// the logger is not initialized, therefore unable to log this err
			} else {
				internal.Logger.Panic(err)
			}
		}
	}()
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
