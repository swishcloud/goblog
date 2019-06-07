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
	g.ErrorPageFunc=ErrorPage
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

type ErrorPageModel struct{
	Status int
}
func ErrorPage(context *goweb.Context, status int) {
	goweb.RenderPage(context, NewPageModel(string(status), ErrorPageModel{Status:status}), "view/layout.html", "view/error.html")

}