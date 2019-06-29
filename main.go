package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/github-123456/goblog/common"
	"github.com/github-123456/goblog/dbservice"
	"github.com/github-123456/goweb"
	"net/http"
)

func main() {
	addr := flag.String("addr", config.Host, "http service address")
	fmt.Println("listening on:", config.Host)
	g := goweb.Default()
	g.ErrorPageFunc = ErrorPage
	g.ConcurrenceNumSem = make(chan int, config.ConcurrenceNum)
	BindHandlers(&g.RouterGroup)

	server := http.Server{
		Addr:    *addr,
		Handler: g,
	}
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

var db *sql.DB
var emailSender common.EmailSender

func init() {
	config = ReadConfig()
	db, _ = sql.Open("mysql", config.SqlDataSourceName)
	dbservice.SetDb(db)

	emailSender = common.EmailSender{UserName: config.SmtpUsername, Password: config.SmtpPassword, Addr: config.SmtpAddr,Name:config.WebsiteName}
}

type ErrorPageModel struct {
	Status int
	Desc   string
}

func ErrorPage(context *goweb.Context, status int, desc string) {
	goweb.RenderPage(context, NewPageModel(string(status), ErrorPageModel{Status: status, Desc: desc}), "view/layout.html", "view/error.html")

}
