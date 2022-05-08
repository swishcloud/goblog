package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/swishcloud/goblog/server"
)

var serveCmd = &cobra.Command{
	Use: "serve",
	Run: func(cmd *cobra.Command, args []string) {
		path := MustGetString(cmd, "config")
		skip_tls_verify, err := cmd.Flags().GetBool("skip-tls-verify")
		if err != nil {
			log.Fatal(err)
		}
		server.NewGoBlogServer(path, skip_tls_verify).Serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("config", "c", "config.yaml", "server config file")
	serveCmd.Flags().Bool("skip-tls-verify", false, "skip tls verify")
	serveCmd.MarkFlagRequired("config")
}
