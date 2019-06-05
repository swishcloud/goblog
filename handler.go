package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"github.com/github-123456/gostudy/aesencryption"
	"github.com/github-123456/goweb"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

const (
	PATH_BLOGLIST = "/bloglist"
	PATH_BLOGEDIT = "/blogedit"
	PATH_BLOGSAVE = "/blogsave"
	PATH_LOGIN    = "/login"
	PATH_REGISTER = "/register"
	PATH_LOGOUT="/logout"
)

func BindHandlers(group *goweb.RouterGroup) {
	group.GET("/", func(context *goweb.Context) {
		if context.Request.URL.Path != "/" {
			http.NotFound(context.Writer, context.Request)
			return
		}
		BlogList(context)
	})
	group.RegexMatch(regexp.MustCompile(`^/blog_\d+\.html$`), Blog)
	group.RegexMatch(regexp.MustCompile(`/static/.+`), func(context *goweb.Context) {
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))).ServeHTTP(context.Writer, context.Request)
	})
	group.GET(PATH_BLOGLIST, BlogList)
	group.GET(PATH_BLOGEDIT, BlogEdit)
	group.POST(PATH_BLOGSAVE, BlogSave)
	group.GET(PATH_LOGIN, Login)
	group.POST(PATH_LOGIN, LoginPost)
	group.POST(PATH_LOGOUT, LogoutPost)
	group.GET(PATH_REGISTER, Register)
	group.POST(PATH_REGISTER, RegisterPost)
}

type PageModel struct {
	User User
	Path string
	WebSiteName string
	PageTitle   string
	Data        interface{}
	Duration int
}

func NewPageModel(pageTitle string, data interface{}) *PageModel {
	return &PageModel{WebSiteName: config.WebsiteName, PageTitle: pageTitle, Data: data}
}

func (p *PageModel) Prepare(c *goweb.Context) interface{} {
	p.Path=c.Request.URL.Path

	u,err:=GetLoginUser(c)
	if err==nil{
		p.User=u
	}

	n:=time.Now()
	p.Duration=int(n.Sub(c.CT)/time.Millisecond)

	return p
}

func GetPageTitle(title string) string {
	return title + " - " + config.WebsiteName
}

const SessionName = "session"

func Authorize(w http.ResponseWriter, req *http.Request) bool {
	cookie, err := req.Cookie(SessionName)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	_ = cookie
	return true
}

type BlogListItemModel struct {
	Id         int
	Title      string
	Content    string
	InsertTime string
}

func BlogList(context *goweb.Context) {
	db, err := sql.Open("mysql", config.SqlDataSourceName)
	if (err != nil) {
		panic(err)
	}
	defer db.Close()

	keys, _ := context.Request.URL.Query()["key"]
	var key string
	if len(keys) > 0 {
		key = keys[0]
	} else {
		key = ""
	}

	rows, err := db.Query("select id,title,content,insertTime from blog where title like ? order by updateTime desc", "%"+key+"%")
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

	goweb.RenderPage(context, NewPageModel("GOBLOG", blogItems), "view/layout.html", "view/bloglist.html")

}

func BlogEdit(context *goweb.Context) {
	if !IsLogin(context) {
		http.Redirect(context.Writer, context.Request, PATH_LOGIN, 302)
	}
	goweb.RenderPage(context, NewPageModel(GetPageTitle("写文章"), nil), "view/layout.html", "view/blogedit.html")
}

func BlogSave(context *goweb.Context) {
	context.Request.ParseForm()
	title := context.Request.PostForm.Get("title")
	content := context.Request.PostForm.Get("content")
	r, err := db.Exec(`insert into blog (title,content,author,insertTime,updateTime,isDeleted,isBanned)values(
	?,?,?,?,?,?,?
	)`, title, content, `xxxx`, time.Now(), time.Now(), 0, 0)
	if err != nil {
		goweb.HandlerResult{Error: err.Error()}.Write(context.Writer)
	} else {
		id, err := r.LastInsertId()
		if err != nil {
			panic(err)
		}
		goweb.HandlerResult{Data: id}.Write(context.Writer)
	}
}

type BlogModel struct {
	Id         int
	Title      string
	Content    string
	InsertTime string
}

func Blog(context *goweb.Context) {
	re := regexp.MustCompile(`\d+`)
	id, _ := strconv.Atoi(re.FindString(context.Request.URL.Path))

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
		http.NotFound(context.Writer, context.Request)
		return
	}
	if err := rows.Scan(&id, &title, &content, &insertTime); err != nil {
		panic(err)
	}

	goweb.RenderPage(context, NewPageModel(GetPageTitle(title), BlogModel{Id: id, Title: title, Content: content, InsertTime: insertTime}), "view/layout.html", "view/blog.html")
}

func Login(context *goweb.Context) {
	goweb.RenderPage(context,NewPageModel(GetPageTitle("登录"), nil), "view/layout.html", "view/login.html")
}

func LoginPost(context *goweb.Context) {
	account := context.Request.PostForm.Get("account")
	password := context.Request.PostForm.Get("password")

	b := md5.Sum([]byte(password))
	hashedPassword := hex.EncodeToString(b[:])

	var (
		r_id       int
		r_password string
		r_userName string
	)
	r := db.QueryRow("select id,userName, password from user where userName=?", account)
	err := r.Scan(&r_id, &r_userName, &r_password)
	if err != nil {
		context.Failed("账号或密码错误")
		return
	}

	if hashedPassword != r_password {
		context.Failed("账号或密码错误")
		return
	}
	jsonB, err := json.Marshal(User{Id: r_id, UserName: r_userName})
	if err != nil {
		context.Failed(err.Error())
		return
	}
	userJsonText := string(jsonB)

	cookie := http.Cookie{Name: SessionName, Value: aesencryption.Encrypt([]byte(config.Key),userJsonText), Path: "/"}
	http.SetCookie(context.Writer, &cookie)

	context.Success(nil)
}

func  LogoutPost(context *goweb.Context)  {
	redirectUri:=context.Request.URL.Query().Get("redirectUri")

	expire := time.Now().Add(-7 * 24 * time.Hour)
	cookie := http.Cookie{
		Name:    SessionName,
		Value:   "",
		Expires: expire,
	}
	http.SetCookie(context.Writer, &cookie)

	http.Redirect(context.Writer,context.Request,redirectUri,302)
}

func Register(context *goweb.Context) {
	goweb.RenderPage(context, NewPageModel(GetPageTitle("注册"), nil), "view/layout.html", "view/register.html")
}
func RegisterPost(context *goweb.Context) {
	account := context.Request.PostForm.Get("account")
	password := context.Request.PostForm.Get("password")
	passwordBytes := []byte(password)
	b := md5.Sum(passwordBytes)
	hashedPassword := hex.EncodeToString(b[:])
	r, err := db.Exec(`insert into user (userName,password,insertTime,isDeleted,isBanned)values(
	?,?,?,?,?
	)`, account, hashedPassword, time.Now(), 0, 0)
	if err != nil {
		goweb.HandlerResult{Error: err.Error()}.Write(context.Writer)
	} else {
		id, err := r.LastInsertId()
		if err != nil {
			panic(err)
		}
		goweb.HandlerResult{Data: id}.Write(context.Writer)
	}
}
