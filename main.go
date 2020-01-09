package main

import (
	"log"

	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/github"
	"github.com/swishcloud/goblog/cmd"
)

func main() {
	cmd.Execute()
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile | log.LUTC)
}
