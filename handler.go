package main

import (
	"encoding/json"
	"github.com/github-123456/goblog/common"
	"github.com/github-123456/goblog/dbservice"
	"github.com/github-123456/gostudy/aesencryption"
	"github.com/github-123456/gostudy/superdb"
	"github.com/github-123456/goweb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"html/template"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

const (
	PATH_ARTICLELIST    = "/articlelist"
	PATH_ARTICLEEDIT    = "/articleedit"
	PATH_ARTICLESAVE    = "/articlesave"
	PATH_ARTICLEDELETE  = "/articledelete"
	PATH_ARTICLELOCK    = "/articlelock"
	PATH_LOGIN          = "/login"
	PATH_REGISTER       = "/register"
	PATH_LOGOUT         = "/logout"
	PATH_CATEGORYLIST   = "/categories"
	PATH_CATEGORYEDIT   = "/categoryedit"
	PATH_CATEGORYSAVE   = "/categorysave"
	PATH_CATEGORYDELETE = "/categorydelete"
	PATH_SETLEVELTWOPWD = "/setlevel2pwd"
	PATH_PROFILE        = "/profile"
	PATH_UPLOAD         = "/upload"
)

func BindHandlers(group *goweb.RouterGroup) {
	group.GET("/", ArticleList)
	group.RegexMatch(regexp.MustCompile(`^/u/\d+/post/\d+$`), Article)
	group.RegexMatch(regexp.MustCompile(`^/user/\d+/article$`), UserArticle)
	group.RegexMatch(regexp.MustCompile(`/static/.+`), func(context *goweb.Context) {
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))).ServeHTTP(context.Writer, context.Request)
	})
	group.RegexMatch(regexp.MustCompile(`/src/.+`), func(context *goweb.Context) {
		http.StripPrefix("/src/", http.FileServer(http.Dir(config.FileLocation))).ServeHTTP(context.Writer, context.Request)
	})
	group.GET(PATH_ARTICLELIST, ArticleList)
	group.GET(PATH_ARTICLEEDIT, ArticleEdit)
	group.POST(PATH_ARTICLESAVE, ArticleSave)
	group.POST(PATH_ARTICLEDELETE, ArticleDelete)
	group.GET(PATH_ARTICLELOCK, ArticleLock)
	group.POST(PATH_ARTICLELOCK, ArticleLockPost)
	group.GET(PATH_LOGIN, Login)
	group.POST(PATH_LOGIN, LoginPost)
	group.POST(PATH_LOGOUT, LogoutPost)
	group.GET(PATH_REGISTER, Register)
	group.POST(PATH_REGISTER, RegisterPost)
	group.GET(PATH_CATEGORYLIST, CategoryList)
	group.GET(PATH_CATEGORYEDIT, CategoryEdit)
	group.POST(PATH_CATEGORYSAVE, CategorySave)
	group.POST(PATH_CATEGORYDELETE, CategoryDelete)
	group.GET(PATH_SETLEVELTWOPWD, SetLevelTwoPwd)
	group.POST(PATH_SETLEVELTWOPWD, SetLevelTwoPwdPost)
	group.GET(PATH_PROFILE, Profile)
	group.POST(PATH_UPLOAD, Upload)
}

type PageModel struct {
	User           User
	Path           string
	RequestUri     string
	Request        *http.Request
	WebSiteName    string
	PageTitle      string
	Data           interface{}
	Duration       int
	LastUpdateTime time.Time
}

func NewPageModel(pageTitle string, data interface{}) *PageModel {
	return &PageModel{WebSiteName: config.WebsiteName, PageTitle: pageTitle, Data: data, LastUpdateTime: config.LastUpdateTime}
}

func (p *PageModel) Prepare(c *goweb.Context) interface{} {
	p.Path = c.Request.URL.Path
	p.RequestUri = c.Request.RequestURI
	p.Request = c.Request

	u, err := GetLoginUser(c)
	if err == nil {
		p.User = u
	}

	n := time.Now()
	p.Duration = int(n.Sub(c.CT) / time.Millisecond)

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

type UserArticleModel struct {
	Articles []*dbservice.ArticleDto
}

func UserArticle(context *goweb.Context) {
	re := regexp.MustCompile(`\d+`)
	id, _ := strconv.Atoi(re.FindString(context.Request.URL.Path))
	user := dbservice.GetUser(id)
	loginUser := MustGetLoginUser(context)
	var queryArticleType int
	if loginUser.Id == user.Id {
		queryArticleType = 0
	} else {
		queryArticleType = 1
	}
	articles := dbservice.GetArticles(queryArticleType, user.Id, "", false)
	model := UserArticleModel{Articles: articles}
	goweb.RenderPage(context, NewPageModel(GetPageTitle(user.UserName), model), "view/layout.html", "view/userLayout.html", "view/userArticle.html")
}

type ArticleListItemModel struct {
	Id         int
	Title      string
	Content    string
	InsertTime string
}

func ArticleList(context *goweb.Context) {
	keys, _ := context.Request.URL.Query()["key"]
	var key string
	if len(keys) > 0 {
		key = keys[0]
	} else {
		key = ""
	}
	data := dbservice.GetArticles(1, 0, key, false)
	goweb.RenderPage(context, NewPageModel("GOBLOG", data), "view/layout.html", "view/articlelist.html")

}

type ArticleEditModel struct {
	CategoryList []dbservice.CategoryDto
	Article      dbservice.ArticleDto
}

func ArticleEdit(context *goweb.Context) {
	if !IsLogin(context) {
		http.Redirect(context.Writer, context.Request, PATH_LOGIN+"?redirectUri="+context.Request.RequestURI, 302)
		return
	}
	categoryList := dbservice.GetCategories(MustGetLoginUser(context).Id)
	model := ArticleEditModel{CategoryList: categoryList}
	if article, ok := context.Data["article"].(*dbservice.ArticleDto); ok {
		model.Article = *article
	} else {
		id := context.Request.URL.Query().Get("id")
		if id != "" {
			intId, err := strconv.Atoi(id)
			if err != nil {
				panic(err)
			}
			article := dbservice.GetArticle(intId)
			if article.ArticleType == 3 {
				http.Redirect(context.Writer, context.Request, PATH_ARTICLELOCK+"?id="+strconv.Itoa(article.Id)+"&t=1", 302)
				return
			}
			model.Article = *article
		}
	}
	goweb.RenderPage(context, NewPageModel(GetPageTitle("写文章"), model), "view/layout.html", "view/articleedit.html")
}

func ArticleSave(context *goweb.Context) {
	context.Request.ParseForm()
	id := context.Request.PostForm.Get("id")
	title := context.Request.PostForm.Get("title")
	content := context.Request.PostForm.Get("content")
	categoryId := context.Request.PostForm.Get("categoryId")
	articleType := context.Request.PostForm.Get("type")
	lev2pwd := context.Request.PostForm.Get("lev2pwd")
	html := context.Request.PostForm.Get("html")
	summary := context.Request.PostForm.Get("summary")
	if len(summary) > 100 {
		summary = summary[:100]
	}

	intArticleType, err := strconv.Atoi(articleType)
	if err != nil {
		panic(err)
	}
	intCategoryId, err := strconv.Atoi(categoryId)
	if err != nil {
		panic(err)
	}

	intId, err := strconv.Atoi(id)
	if err != nil {
		panic(err)
	}
	if intId == 0 {
		superdb.ExecuteTransaction(db, dbservice.NewArticle(title, summary, html, content, MustGetLoginUser(context).Id, intArticleType, intCategoryId, lev2pwd))
	} else {
		rawArticle := dbservice.GetArticle(intId)
		superdb.ExecuteTransaction(db, dbservice.NewArticle(rawArticle.Title, rawArticle.Summary, rawArticle.Html, rawArticle.Content, rawArticle.UserId, 4, rawArticle.CategoryId, lev2pwd), dbservice.UpdateArticle(intId, title, summary, html, content, intArticleType, categoryId, lev2pwd, MustGetLoginUser(context).Id))
	}
	context.Success(nil)
}

type ArticleModel struct {
	Article  *dbservice.ArticleDto
	Html     template.HTML
	Readonly bool
}

func Article(context *goweb.Context) {
	re := regexp.MustCompile(`\d+$`)
	id, _ := strconv.Atoi(re.FindString(context.Request.URL.Path))
	article := dbservice.GetArticle(id)
	if article == nil {
		context.ShowErrorPage(http.StatusNotFound, "page not found")
		return
	}
	model := ArticleModel{Article: article, Readonly: true}
	if !IsLogin(context) {
		if article.ArticleType != 1 {
			http.Redirect(context.Writer, context.Request, PATH_LOGIN, 302)
			return
		}
	} else {
		loginUserId := MustGetLoginUser(context).Id
		if article.ArticleType != 1 {
			if article.UserId != loginUserId {
				context.ShowErrorPage(http.StatusUnauthorized, "")
				return
			}
		}
		if article.ArticleType == 3 {
			http.Redirect(context.Writer, context.Request, PATH_ARTICLELOCK+"?id="+strconv.Itoa(article.Id)+"&t=2", 302)
			return
		}
		if article.UserId == loginUserId {
			model.Readonly = false
		}
	}
	model.Html = template.HTML(model.Article.Html)
	goweb.RenderPage(context, NewPageModel(GetPageTitle(article.Title), model), "view/layout.html", "view/article.html")
}

type ArticleLockModel struct {
	Id    string
	Error string
	Type  string
}

func ArticleLock(context *goweb.Context) {
	id := context.Request.URL.Query().Get("id")
	t := context.Request.URL.Query().Get("t")
	goweb.RenderPage(context, NewPageModel(GetPageTitle("lock"), ArticleLockModel{Id: id, Type: t}), "view/layout.html", "view/articlelock.html")
}
func ArticleLockPost(context *goweb.Context) {
	id := context.Request.PostForm.Get("id")
	pwd := context.Request.PostForm.Get("pwd")
	t, _ := strconv.Atoi(context.Request.PostForm.Get("type"))
	intId, _ := strconv.Atoi(id)
	article := dbservice.GetArticle(intId)
	if article.UserId != MustGetLoginUser(context).Id {
		context.ShowErrorPage(http.StatusUnauthorized, "")
		return
	}
	c, err := aesencryption.Decrypt(pwd, article.Content)
	if err != nil || !common.Lev2PwdCheck(*dbservice.GetUser(article.UserId).Level2pwd, pwd) {
		goweb.RenderPage(context, NewPageModel(GetPageTitle("lock"), ArticleLockModel{Id: id, Type: strconv.Itoa(t), Error: "二级密码错误"}), "view/layout.html", "view/articlelock.html")
		return
	}
	article.Content = c;
	if t == 1 {
		context.Data["article"] = article
		ArticleEdit(context)
	} else if t == 2 {
		model := ArticleModel{Article: article, Readonly: false}
		html, _ := aesencryption.Decrypt(pwd, article.Html)
		model.Html = template.HTML(html)
		goweb.RenderPage(context, NewPageModel(GetPageTitle(article.Title), model), "view/layout.html", "view/article.html")
	}
}

func Login(context *goweb.Context) {
	redirectUri := context.Request.URL.Query().Get("redirectUri")
	goweb.RenderPage(context, NewPageModel(GetPageTitle("登录"), redirectUri), "view/layout.html", "view/login.html")
}

func LoginPost(context *goweb.Context) {
	account := context.Request.PostForm.Get("account")
	password := context.Request.PostForm.Get("password")

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

	if !common.PwdCheck(r_password, password) {
		context.Failed("账号或密码错误")
		return
	}
	jsonB, err := json.Marshal(User{Id: r_id, UserName: r_userName})
	if err != nil {
		context.Failed(err.Error())
		return
	}
	userJsonText := string(jsonB)

	cookie := http.Cookie{Name: SessionName, Value: aesencryption.Encrypt(config.Key, userJsonText), Path: "/"}
	http.SetCookie(context.Writer, &cookie)

	context.Success(nil)
}

func LogoutPost(context *goweb.Context) {
	redirectUri := context.Request.URL.Query().Get("redirectUri")

	expire := time.Now().Add(-7 * 24 * time.Hour)
	cookie := http.Cookie{
		Name:    SessionName,
		Value:   "",
		Expires: expire,
	}
	http.SetCookie(context.Writer, &cookie)

	http.Redirect(context.Writer, context.Request, redirectUri, 302)
}

func Register(context *goweb.Context) {
	goweb.RenderPage(context, NewPageModel(GetPageTitle("注册"), nil), "view/layout.html", "view/register.html")
}
func RegisterPost(context *goweb.Context) {
	account := context.Request.PostForm.Get("account")
	password := context.Request.PostForm.Get("password")
	superdb.ExecuteTransaction(db, dbservice.NewUser(account, password))
	context.Success(nil)
}
func CategoryList(context *goweb.Context) {
	categoryList := dbservice.GetCategories(MustGetLoginUser(context).Id)
	goweb.RenderPage(context, NewPageModel(GetPageTitle("我的分类"), categoryList), "view/layout.html", "view/categorylist.html")
}

type CategoryEditModel struct {
	Id   string
	Name string
}

func CategoryEdit(context *goweb.Context) {
	id := context.Request.URL.Query().Get("id")
	var model CategoryEditModel
	if id != "" {
		r := db.QueryRow(`select name from category where isdeleted=0 and id=?`, id)
		var name string
		err := r.Scan(&name)
		if err != nil {
			context.ShowErrorPage(http.StatusNotFound, err.Error())
			return
		}
		model = CategoryEditModel{Id: id, Name: name}
	}
	title := "编辑分类"
	if id == "" {
		title = "新增分类"
	}
	goweb.RenderPage(context, NewPageModel(GetPageTitle(title), model), "view/layout.html", "view/categoryedit.html")
}
func CategorySave(context *goweb.Context) {
	if !IsLogin(context) {
		context.Failed("登录失效")
		return
	}
	name := context.Request.PostForm.Get("name")
	id := context.Request.PostForm.Get("id")
	if id == "" {
		superdb.ExecuteTransaction(db, dbservice.NewCategory(name, MustGetLoginUser(context).Id))
	} else {
		intId, _ := strconv.Atoi(id)
		superdb.ExecuteTransaction(db, dbservice.UpdateCategory(name, intId, MustGetLoginUser(context).Id))
	}
	context.Success(nil)
}
func CategoryDelete(context *goweb.Context) {
	//todo check that articles exists
	id := context.Request.FormValue("id")
	_, err := db.Exec(`delete from category where id=?`, id)
	if err != nil {
		context.Failed(err.Error())
		return
	}
	context.Success(nil)
}

func ArticleDelete(context *goweb.Context) {
	id := context.Request.FormValue("id")
	intId, _ := strconv.Atoi(id)
	superdb.ExecuteTransaction(db, dbservice.ArticleDelete(intId, MustGetLoginUser(context).Id))
	context.Success(nil)
}

type SettingsItemModel struct {
	Path   string
	Name   string
	Active bool
}

func GetSettingsModel(activePath string) []*SettingsItemModel {
	model := []*SettingsItemModel{
		&SettingsItemModel{Path: PATH_PROFILE, Name: "个人资料"},
		&SettingsItemModel{Path: PATH_SETLEVELTWOPWD, Name: "二级密码"},
	}
	for _, v := range model {
		if v.Path == activePath {
			v.Active = true
			break
		}
	}
	return model
}

type SetLevelTwoPwdModel struct {
	Settings       []*SettingsItemModel
	ExistLevel2Pwd bool
}

func SetLevelTwoPwd(context *goweb.Context) {
	user := dbservice.GetUser(MustGetLoginUser(context).Id)
	existLevel2Pwd := user.Level2pwd != nil
	model := SetLevelTwoPwdModel{Settings: GetSettingsModel(context.Request.URL.Path)}
	model.ExistLevel2Pwd = existLevel2Pwd
	goweb.RenderPage(context, NewPageModel(GetPageTitle("设置二级密码"), model), "view/layout.html", "view/settingsLeftBar.html", "view/setleveltwopwd.html")
}
func SetLevelTwoPwdPost(context *goweb.Context) {
	oldPwd := context.Request.PostForm.Get("oldPwd")
	newPwd := context.Request.PostForm.Get("newPwd")
	user := dbservice.GetUser(MustGetLoginUser(context).Id)
	if user.Level2pwd != nil {
		if !common.Lev2PwdCheck(*user.Level2pwd, oldPwd) {
			context.Failed("旧密码有误")
			return
		}
	}
	superdb.ExecuteTransaction(db, dbservice.SetLevelTwoPwd(oldPwd, newPwd, user.Id))
	context.Success(nil)
}

type ProfileModel struct {
	Settings []*SettingsItemModel
}

func Profile(context *goweb.Context) {
	goweb.RenderPage(context, NewPageModel(GetPageTitle("个人资料"), ProfileModel{Settings: GetSettingsModel(context.Request.URL.Path)}), "view/layout.html", "view/settingsLeftBar.html", "view/profile.html")
}
func Upload(context *goweb.Context) {
	file, fileHeader, err := context.Request.FormFile("image")
	if err != nil {
		panic(err)
	}
	uuid:=uuid.New().String()+".png"
	path := config.FileLocation + `/image/` +uuid
	url:="/src/image/"+uuid
	out, err := os.Create(path)
	defer out.Close()
	if err != nil {
		panic(err)
	}
	io.Copy(out, file)
	data := struct {
		DownloadUrl string `json:"downloadUrl"`
		Filename    string `json:"filename"`
	}{DownloadUrl: url, Filename: fileHeader.Filename}
	json, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	context.Writer.Header().Add("Content-Type", "application/json")
	context.Writer.Write(json)
}
