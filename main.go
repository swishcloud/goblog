package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/github-123456/goweb"
	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/github"
	"github.com/xiaozemin/goblog/chat"
	"github.com/xiaozemin/goblog/common"
	"github.com/xiaozemin/goblog/dbservice"
	"net/http"
)

func main() {
	fmt.Println("listening on:", config.Host)
	g := goweb.Default()
	g.ErrorPageFunc = ErrorPage
	g.ConcurrenceNumSem = make(chan int, config.ConcurrenceNum)
	BindHandlers(&g.RouterGroup)
	server := http.Server{
		Addr:    config.Host,
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
	configPath := flag.String("config","config-development.json", "application configuration file")
	flag.Parse()
	config = ReadConfig(*configPath)
	dbservice.InitializeDb(config.SqlDataSourceName)
	go chat.GetHub().Run()
	chat.GetHub().FileLocation=config.FileLocation
	emailSender = common.EmailSender{UserName: config.SmtpUsername, Password: config.SmtpPassword, Addr: config.SmtpAddr, Name: config.WebsiteName}
}

type ErrorPageModel struct {
	Status int
	Desc   string
}

func ErrorPage(context *goweb.Context, status int, desc string) {
	goweb.RenderPage(context, NewPageModel(string(status), ErrorPageModel{Status: status, Desc: desc}), "view/layout.html", "view/error.html")

}
