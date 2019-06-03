package main

import (
	"database/sql"
	"fmt"
	"github.com/github-123456/goweb"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"net/http"
	"time"
)

func BindHandlers(mux  *http.ServeMux)  {
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		fmt.Fprintf(w, "Welcome to the home page!")
	})

	mux.HandleFunc("/bloglist",BlogList)
	mux.HandleFunc("/blogedit",BlogEdit)
	mux.HandleFunc("/blogsave",BlogSave)
}

func BlogList(w http.ResponseWriter, req *http.Request) {
	db, err := sql.Open("mysql", config.SqlDataSourceName)
	if(err!=nil){
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	rows,err:=db.Query("select title,content,insertTime from blog")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	for rows.Next() {
			var(title   string
			content string
			insertTime string
			)
		if err := rows.Scan(&title, &content,&insertTime); err != nil {
			panic(err)
		}
		fmt.Fprintf(w,"title:%s\r\ncontent:%s\r\n",title,content)
	}



}

func BlogEdit(w http.ResponseWriter, req *http.Request) {
	tmpl, err := template.ParseFiles("view/layout.html", "view/blogedit.html")
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}
}

func BlogSave(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	title:=req.PostForm.Get("title")
	content:=req.PostForm.Get("content")
	r,err:=db.Exec(`insert into blog (title,content,author,insertTime,updateTime,isDeleted,isBanned)values(
	?,?,?,?,?,?,?
	)`,title,content,`xxxx`,time.Now(),time.Now(),0,0)
	if err != nil {
		goweb.HandlerResult{Error:err.Error()}.Write(w)
	}else{
		id,err:=r.LastInsertId()
		if err != nil {
			panic(err)
		}
		goweb.HandlerResult{Data:id}.Write(w)
	}
}