package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/http"
)

func main() {
	mux:=http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	BindHandlers(mux)
	addr:= flag.String("addr", config.Host, "http service address")
	fmt.Print("listening on:",config.Host)
	err := http.ListenAndServe(*addr, mux)
	if err != nil {
		panic(err)
	}
}

var db *sql.DB

func init()  {
	config=ReadConfig()
	db, _ = sql.Open("mysql", config.SqlDataSourceName)
}