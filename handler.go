package main

import (
	"database/sql"
	"github.com/github-123456/goweb"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

func BindHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		matched, _ := regexp.MatchString(`^/blog_\d+\.html$`, req.URL.Path);
		if matched {
			Blog(w, req)
			return
		}
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		BlogList(w, req)
	})

	mux.HandleFunc("/bloglist", BlogList)
	mux.HandleFunc("/blogedit", BlogEdit)
	mux.HandleFunc("/blogsave", BlogSave)
}

type PageModel struct {
	WebSiteName string
	PageTitle   string
	Data        interface{}
}

func NewPageModel(pageTitle string, data interface{}) PageModel {
	return PageModel{WebSiteName: config.WebsiteName, PageTitle: pageTitle, Data: data}
}

func GetPageTitle(title string) string {
	return title + " - " + config.WebsiteName
}

type BlogListItemModel struct {
	Id         int
	Title      string
	Content    string
	InsertTime string
}

func BlogList(w http.ResponseWriter, req *http.Request) {
	db, err := sql.Open("mysql", config.SqlDataSourceName)
	if (err != nil) {
		panic(err)
	}
	defer db.Close()

	keys, _ := req.URL.Query()["key"]
	var key string
	if len(keys) > 0 {
		key = keys[0]
	} else {
		key = ""
	}

	rows, err := db.Query("select id,title,content,insertTime from blog where title like ?", "%"+key+"%")
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	var blogItems []BlogListItemModel
	for rows.Next() {
		var (
			id         int
			title      string
			content    string
			insertTime string
		)
		if err := rows.Scan(&id, &title, &content, &insertTime); err != nil {
			panic(err)
		}
		blogItems = append(blogItems, BlogListItemModel{Id: id, Title: title, Content: content, InsertTime: insertTime})
	}

	goweb.RenderPage(w,NewPageModel("GOBLOG", blogItems), "view/layout.html", "view/bloglist.html")

}

func BlogEdit(w http.ResponseWriter, req *http.Request) {
	goweb.RenderPage(w, NewPageModel(GetPageTitle("写文章"), nil), "view/layout.html", "view/blogedit.html")
}

func BlogSave(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	title := req.PostForm.Get("title")
	content := req.PostForm.Get("content")
	r, err := db.Exec(`insert into blog (title,content,author,insertTime,updateTime,isDeleted,isBanned)values(
	?,?,?,?,?,?,?
	)`, title, content, `xxxx`, time.Now(), time.Now(), 0, 0)
	if err != nil {
		goweb.HandlerResult{Error: err.Error()}.Write(w)
	} else {
		id, err := r.LastInsertId()
		if err != nil {
			panic(err)
		}
		goweb.HandlerResult{Data: id}.Write(w)
	}
}

type BlogModel struct {
	Id         int
	Title      string
	Content    string
	InsertTime string
}

func Blog(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile(`\d+`)
	id, _ := strconv.Atoi(re.FindString(req.URL.Path))

	rows, err := db.Query("select id,title,content,insertTime from blog where id=?", id)
	if err != nil {
		panic(err)
	}
	var (
		title      string
		content    string
		insertTime string
	)
	if !rows.Next() {
		http.NotFound(w, req)
		return
	}
	if err := rows.Scan(&id, &title, &content, &insertTime); err != nil {
		panic(err)
	}

	goweb.RenderPage(w, NewPageModel(GetPageTitle(title), BlogModel{Id: id, Title: title, Content: content, InsertTime: insertTime}), "view/layout.html", "view/blog.html")
}
