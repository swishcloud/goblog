package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/github-123456/goweb"
	"net/http"
)

func main() {
	addr:= flag.String("addr", config.Host, "http service address")
	fmt.Print("listening on:",config.Host)
	g:=goweb.New()
	BindHandlers(&g.RouterGroup)

	err := http.ListenAndServe(*addr, g)
	if err != nil {
		panic(err)
	}
}

var db *sql.DB

func init()  {
	config=ReadConfig()
	db, _ = sql.Open("mysql", config.SqlDataSourceName)
}