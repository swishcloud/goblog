package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/github-123456/gostudy/aesencryption"
	"github.com/github-123456/gostudy/superdb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/swishcloud/goblog/dbservice"
	"github.com/swishcloud/goblog/storage/models"
	"github.com/swishcloud/goweb"
	"github.com/swishcloud/goweb/auth"
	"github.com/xiaozemin/goblog/chat"
	"github.com/xiaozemin/goblog/common"
	"golang.org/x/oauth2"
)

const session_user_key = "session_user"
const (
	PATH_ARTICLELIST       = "/articlelist"
	PATH_ARTICLEEDIT       = "/articleedit"
	PATH_ARTICLESAVE       = "/articlesave"
	PATH_ARTICLEDELETE     = "/articledelete"
	PATH_ARTICLELOCK       = "/articlelock"
	PATH_LOGIN             = "/login"
	PATH_LOGIN_CALLBACK    = "/login-callback"
	PATH_LOGOUT            = "/logout"
	PATH_CATEGORYLIST      = "/categories"
	PATH_CATEGORYEDIT      = "/categoryedit"
	PATH_CATEGORYSAVE      = "/categorysave"
	PATH_CATEGORYDELETE    = "/categorydelete"
	PATH_SETLEVELTWOPWD    = "/setlevel2pwd"
	PATH_PROFILE           = "/profile"
	PATH_UPLOAD            = "/upload"
	PATH_EMAILVALIDATE     = "/emailValidate"
	PATH_EMAILVALIDATESEND = "/emailValidateSend"
	PATH_WEBSOCKET         = "/ws"
	PATH_CHAT              = "/chat"
)

func BindHandlers(group *goweb.RouterGroup) {
	auth := group.Group()
	auth.Use(AuthMiddleware())

	group.GET("/", ArticleList)
	group.RegexMatch(regexp.MustCompile(`^/u/\d+/post/\d+$`), Article)
	group.RegexMatch(regexp.MustCompile(`^/user/\d+/article$`), UserArticle)
	group.RegexMatch(regexp.MustCompile(`/static/.+`), func(context *goweb.Context) {
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))).ServeHTTP(context.Writer, context.Request)
	})
	group.RegexMatch(regexp.MustCompile(`/src/.+`), func(context *goweb.Context) {
		http.StripPrefix("/src/", http.FileServer(http.Dir(config.FileLocation))).ServeHTTP(context.Writer, context.Request)
	})
	group.GET(PATH_WEBSOCKET, chat.WebSocket)
	group.GET(PATH_ARTICLELIST, ArticleList)
	auth.GET(PATH_ARTICLEEDIT, ArticleEdit)
	auth.POST(PATH_ARTICLESAVE, ArticleSave)
	auth.POST(PATH_ARTICLEDELETE, ArticleDelete)
	auth.GET(PATH_ARTICLELOCK, ArticleLock)
	auth.POST(PATH_ARTICLELOCK, ArticleLockPost)
	group.GET(PATH_LOGIN, Login)
	group.GET(PATH_LOGIN_CALLBACK, LoginCallback)
	group.POST(PATH_LOGOUT, LogoutPost)
	auth.GET(PATH_CATEGORYLIST, CategoryList)
	auth.GET(PATH_CATEGORYEDIT, CategoryEdit)
	auth.POST(PATH_CATEGORYSAVE, CategorySave)
	auth.POST(PATH_CATEGORYDELETE, CategoryDelete)
	auth.GET(PATH_SETLEVELTWOPWD, SetLevelTwoPwd)
	auth.POST(PATH_SETLEVELTWOPWD, SetLevelTwoPwdPost)
	auth.GET(PATH_PROFILE, Profile)
	auth.POST(PATH_UPLOAD, Upload)
	group.GET(PATH_EMAILVALIDATE, EmailValidate)
	group.POST(PATH_EMAILVALIDATESEND, EmailValidateSend)
	group.GET(PATH_CHAT, Chat)
}

type PageModel struct {
	User             *models.User
	Path             string
	RequestUri       string
	Request          *http.Request
	WebSiteName      string
	PageTitle        string
	Keywords         string
	Description      string
	Data             interface{}
	Duration         int
	LastUpdateTime   string
	MobileCompatible bool
	Config           Config
}

func NewPageModel(ctx *goweb.Context, title string, data interface{}) *PageModel {
	m := &PageModel{WebSiteName: config.WebsiteName, Data: data, LastUpdateTime: config.LastUpdateTime, MobileCompatible: true, Config: config}
	m.Path = ctx.Request.URL.Path
	m.RequestUri = ctx.Request.RequestURI
	m.Request = ctx.Request
	u, err := GetLoginUser(ctx)
	if err == nil {
		m.User = u
	}
	n := time.Now()
	m.Duration = int(n.Sub(ctx.CT) / time.Millisecond)
	m.PageTitle = title + " - " + config.WebsiteName
	return m
}

const SessionName = "session"

type UserArticleModel struct {
	Articles   []dbservice.ArticleDto
	Categories []dbservice.CategoryDto
	UserId     int
}

func (m UserArticleModel) GetCategoryUrl(name string) string {
	return fmt.Sprintf("/user/%d/article?category=%s", m.UserId, name)
}

func GetLoginUser(ctx *goweb.Context) (*models.User, error) {
	if s, err := auth.GetSessionByToken(ctx); err != nil {
		return nil, err
	} else {
		if u, ok := s.Data[session_user_key].(*models.User); ok {
			return u, nil
		}
		return nil, errors.New("user not logged in")
	}
}
func MustGetLoginUser(ctx *goweb.Context) *models.User {
	u, err := GetLoginUser(ctx)
	if err != nil {
		panic(err)
	}
	return u
}
func UserArticle(context *goweb.Context) {
	category := context.Request.URL.Query().Get("category")
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
	articles := dbservice.GetArticles(queryArticleType, user.Id, "", false, category)
	categories := dbservice.GetCategories(user.Id, queryArticleType)
	model := UserArticleModel{Articles: articles, Categories: categories, UserId: user.Id}
	context.RenderPage(NewPageModel(context, user.UserName, model), "view/layout.html", "view/userLayout.html", "view/userArticle.html")
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
	articles := dbservice.GetArticles(1, 0, key, false, "")
	context.RenderPage(NewPageModel(context, config.WebsiteName, struct {
		Key      string
		Articles []dbservice.ArticleDto
	}{Key: key, Articles: articles}), "view/layout.html", "view/leftRightLayout.html", "view/articlelist.html")
}

type ArticleEditModel struct {
	CategoryList []dbservice.CategoryDto
	Article      dbservice.ArticleDto
	UserId       int
}

func ArticleEdit(context *goweb.Context) {
	categoryList := dbservice.GetCategories(MustGetLoginUser(context).Id, 0)
	model := ArticleEditModel{CategoryList: categoryList, UserId: MustGetLoginUser(context).Id}
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
	context.RenderPage(NewPageModel(context, "写文章", model), "view/layout.html", "view/articleedit.html")
}

func ArticleSave(context *goweb.Context) {
	context.Request.ParseForm()
	id := context.Request.PostForm.Get("id")
	title := context.Request.PostForm.Get("title")
	content := context.Request.PostForm.Get("content")
	categoryId := context.Request.PostForm.Get("categoryId")
	articleType := context.Request.PostForm.Get("type")
	html := goweb.SanitizeHtml(context.Request.PostForm.Get("html"))
	summary := context.Request.PostForm.Get("summary")

	c := context.Request.PostForm.Get("cover")
	var cover *string
	if c == "" {
		cover = nil
	} else {
		cover = &c
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
		newArticleLastInsertId := superdb.ExecuteTransaction(db, dbservice.NewArticle(title, summary, html, content, MustGetLoginUser(context).Id, intArticleType, intCategoryId, config.PostKey, cover))["NewArticleLastInsertId"].(int64)
		intId = int(newArticleLastInsertId)
	} else {
		superdb.ExecuteTransaction(db, dbservice.UpdateArticle(intId, title, summary, html, content, intArticleType, categoryId, config.PostKey, MustGetLoginUser(context).Id, cover))
	}
	context.Success(intId)
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
	if !auth.HasLoggedIn(context) {
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
	context.RenderPage(NewPageModel(context, article.Title, model), "view/layout.html", "view/article.html")
}

type ArticleLockModel struct {
	Id    string
	Error string
	Type  string
}

func ArticleLock(context *goweb.Context) {
	id := context.Request.URL.Query().Get("id")
	t := context.Request.URL.Query().Get("t")
	context.RenderPage(NewPageModel(context, "lock", ArticleLockModel{Id: id, Type: t}), "view/layout.html", "view/articlelock.html")
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
	if !common.Md5Check(*dbservice.GetUser(article.UserId).Level2pwd, pwd) {
		context.RenderPage(NewPageModel(context, "lock", ArticleLockModel{Id: id, Type: strconv.Itoa(t), Error: "二级密码错误"}), "view/layout.html", "view/articlelock.html")
		return
	}
	c, err := aesencryption.Decrypt([]byte(config.PostKey), article.Content)
	if err != nil {
		panic(err)
	}
	article.Content = c
	if t == 1 {
		context.Data["article"] = article
		ArticleEdit(context)
	} else if t == 2 {
		model := ArticleModel{Article: article, Readonly: false}
		html, _ := aesencryption.Decrypt([]byte(config.PostKey), article.Html)
		model.Html = template.HTML(html)
		context.RenderPage(NewPageModel(context, article.Title, model), "view/layout.html", "view/article.html")
	}
}

func Login(ctx *goweb.Context) {
	url := config.OAUTH2Config.AuthCodeURL("state-string", oauth2.AccessTypeOffline)
	http.Redirect(ctx.Writer, ctx.Request, url, 302)
}
func LoginCallback(ctx *goweb.Context) {
	code := ctx.Request.URL.Query().Get("code")
	token, err := config.OAUTH2Config.Exchange(context.Background(), code)
	if err != nil {
		ctx.Writer.Write([]byte(err.Error()))
		return
	}
	s := auth.Login(ctx, token, config.JWKJsonUrl)
	id := s.Claims["sub"].(string)
	name := s.Claims["name"].(string)
	iss := s.Claims["iss"].(string)
	email := s.Claims["email"].(string)
	user, err := dbservice.GetUserByOP(id, iss)
	if err != nil {
		_ = superdb.ExecuteTransaction(db, dbservice.NewUser(name, iss, id, email))["newUserId"].(int)
		user, err = dbservice.GetUserByOP(id, iss)
		if err != nil {
			panic(err)
		}
	}
	u := &models.User{}
	u.Id = user.Id
	u.UserName = user.UserName
	s.Data[session_user_key] = u
	http.Redirect(ctx.Writer, ctx.Request, "/", 302)
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

func sendValidateEmail(context *goweb.Context, userId int) {
	protocol := "https"
	user := dbservice.GetUser(userId)
	if user.Email == nil {
		panic("the user does not have a email")
	}
	activateAddr := protocol + "://" + context.Request.Host + PATH_EMAILVALIDATE + "?email=" + *user.Email + "&code=" + url.QueryEscape(*user.SecurityStamp)
	emailSender.SendEmail(*user.Email, "邮箱激活", fmt.Sprintf("<html><body>"+
		"%s，您好:<br/><br/>"+
		"感谢您注册%s,您的登录邮箱为%s,请点击以下链接激活您的邮箱地址：<br/><br/>"+
		"<a href='%s'>%s</a><br/><br/>"+
		"如果以上链接无法访问，请将该网址复制并粘贴至浏览器窗口中直接访问。", user.UserName, config.WebsiteName, *user.Email, activateAddr, activateAddr)+
		"</body></html>")
}

func CategoryList(context *goweb.Context) {
	categoryList := dbservice.GetCategories(MustGetLoginUser(context).Id, 0)
	context.RenderPage(NewPageModel(context, "我的分类", categoryList), "view/layout.html", "view/categorylist.html")
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
	context.RenderPage(NewPageModel(context, title, model), "view/layout.html", "view/categoryedit.html")
}
func CategorySave(context *goweb.Context) {
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
	id := context.Request.FormValue("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		panic(err)
	}
	superdb.ExecuteTransaction(db, dbservice.CategoryDelete(intId))
	context.Success(nil)
}

func ArticleDelete(context *goweb.Context) {
	id := context.Request.FormValue("id")
	intId, _ := strconv.Atoi(id)
	superdb.ExecuteTransaction(db, dbservice.ArticleDelete(intId, MustGetLoginUser(context).Id, config.PostKey))
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
	context.RenderPage(NewPageModel(context, "设置二级密码", model), "view/layout.html", "view/settingsLeftBar.html", "view/setleveltwopwd.html")
}
func SetLevelTwoPwdPost(context *goweb.Context) {
	oldPwd := context.Request.PostForm.Get("oldPwd")
	newPwd := context.Request.PostForm.Get("newPwd")
	user := dbservice.GetUser(MustGetLoginUser(context).Id)
	if user.Level2pwd != nil {
		if !common.Md5Check(*user.Level2pwd, oldPwd) {
			context.Failed("旧密码有误")
			return
		}
	}
	superdb.ExecuteTransaction(db, dbservice.SetLevelTwoPwd(user.Id, newPwd))
	context.Success(nil)
}

type ProfileModel struct {
	Settings []*SettingsItemModel
}

func Profile(context *goweb.Context) {
	context.RenderPage(NewPageModel(context, "个人资料", ProfileModel{Settings: GetSettingsModel(context.Request.URL.Path)}), "view/layout.html", "view/settingsLeftBar.html", "view/profile.html")
}
func Upload(context *goweb.Context) {
	file, fileHeader, err := context.Request.FormFile("image")
	if err != nil {
		panic(err)
	}
	uuid := uuid.New().String() + ".png"
	path := config.FileLocation + `/image/` + uuid
	url := "/src/image/" + uuid
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
func EmailValidate(context *goweb.Context) {
	email := context.Request.Form.Get("email")
	code := context.Request.Form.Get("code")
	user := dbservice.GetUserByEmail(email)
	if user != nil {
		if user.EmailConfirmed == 0 {
			if code == "" {
				context.RenderPage(NewPageModel(context, "邮箱验证", email), "view/layout.html", "view/emailValidate.html")
			} else {
				dbservice.ValidateEmail(email, code)
				http.Redirect(context.Writer, context.Request, PATH_LOGIN, 302)
			}
		} else {
			http.Redirect(context.Writer, context.Request, PATH_LOGIN, 302)
		}
	} else {
		panic("email not registered")
	}
}

func EmailValidateSend(context *goweb.Context) {
	email := context.Request.Form.Get("email")
	user := dbservice.GetUserByEmail(email)
	sendValidateEmail(context, user.Id)
	context.Success(nil)
}
func Chat(context *goweb.Context) {
	context.RenderPage(NewPageModel(context, "IM", config.UseHttps), "view/layout.html", "view/chat.html")
}
