package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"

	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/github"
	"github.com/swishcloud/goblog/chat"
	"github.com/swishcloud/goblog/common"
	"github.com/swishcloud/goblog/storage"
	"github.com/swishcloud/goweb"
)

func main() {
	fmt.Println("listening on:", config.Host)
	log.Println("accepting tcp connections on http://" + config.Host)
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
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile | log.LUTC)
	configPath := flag.String("config", "config-development.json", "application configuration file")
	flag.Parse()
	config = ReadConfig(*configPath)
	storage.InitializeDb(config.SqlDataSourceName)
	hub := chat.GetHub(config.SqlDataSourceName)
	go hub.Run()
	hub.FileLocation = config.FileLocation
	emailSender = common.EmailSender{UserName: config.SmtpUsername, Password: config.SmtpPassword, Addr: config.SmtpAddr, Name: config.WebsiteName}
}

type ErrorPageModel struct {
	Status int
	Desc   string
}

func ErrorPage(context *goweb.Context, status int, desc string) {
	context.RenderPage(NewPageModel(context, string(status), ErrorPageModel{Status: status, Desc: desc}), "view/layout.html", "view/error.html")

}
