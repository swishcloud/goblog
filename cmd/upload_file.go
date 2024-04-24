package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/swishcloud/goblog/server"
)

var uploadFileCmd = &cobra.Command{
	Use:   "file",
	Short: "upload local files to cloud",
	Run: func(cmd *cobra.Command, args []string) {
		path := MustGetString(cmd, "config")
		skip_tls_verify, err := cmd.Flags().GetBool("skip-tls-verify")
		if err != nil {
			log.Fatal(err)
		}
		err = server.NewGoBlogServer(path, skip_tls_verify).UploadLocalFiles()
		if err != nil {
			log.Println(err.Error())
			return
		}
		log.Println("Finish")
	},
}

func init() {
	uploadCmd.AddCommand(uploadFileCmd)
	uploadFileCmd.Flags().StringP("config", "c", "config.yaml", "server config file")
	uploadFileCmd.Flags().Bool("skip-tls-verify", false, "skip tls verify")
	uploadFileCmd.MarkFlagRequired("config")
}
