package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/swishcloud/goblog/chat"
	"github.com/swishcloud/goblog/common"
	"github.com/swishcloud/goblog/storage"
	"github.com/swishcloud/goblog/storage/models"
	"github.com/swishcloud/gostudy/aesencryption"
	"github.com/swishcloud/goweb"
	"github.com/swishcloud/goweb/auth"
	"golang.org/x/oauth2"
)

const session_user_key = "session_user"
const (
	PATH_ARTICLELIST    = "/articlelist"
	PATH_ARTICLEEDIT    = "/articleedit"
	PATH_ARTICLESAVE    = "/articlesave"
	PATH_ARTICLEDELETE  = "/articledelete"
	PATH_ARTICLELOCK    = "/articlelock"
	PATH_LOGIN          = "/login"
	PATH_LOGIN_CALLBACK = "/login-callback"
	PATH_LOGOUT         = "/logout"
	PATH_CATEGORYLIST   = "/categories"
	PATH_CATEGORYEDIT   = "/categoryedit"
	PATH_CATEGORYSAVE   = "/categorysave"
	PATH_CATEGORYDELETE = "/categorydelete"
	PATH_SETLEVELTWOPWD = "/setlevel2pwd"
	PATH_PROFILE        = "/profile"
	PATH_UPLOAD         = "/upload"
	PATH_WEBSOCKET      = "/ws"
	PATH_CHAT           = "/chat"
)

func BindHandlers(group *goweb.RouterGroup) {
	group.Use(StorageMiddleware())
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
	group.GET(PATH_WEBSOCKET, chat.WebSocket(config.SqlDataSourceName))
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
	group.GET(PATH_CHAT, Chat)
}

func StorageMiddleware() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		ctx.Next()
		m := ctx.Data["storage"]
		if m != nil {
			m.(storage.Storage).Commit()
		}
	}
}
func GetStorage(ctx *goweb.Context) storage.Storage {
	m := ctx.Data["storage"]
	if m == nil {
		m = storage.NewSQLManager(config.SqlDataSourceName)
		ctx.Data["storage"] = m
	}
	return m.(storage.Storage)
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
	Articles   []models.ArticleDto
	Categories []models.CategoryDto
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
	user := GetStorage(context).GetUser(id)
	loginUser := MustGetLoginUser(context)
	var queryArticleType int
	if loginUser.Id == user.Id {
		queryArticleType = 0
	} else {
		queryArticleType = 1
	}
	articles := GetStorage(context).GetArticles(queryArticleType, user.Id, "", false, category)
	categories := GetStorage(context).GetCategories(user.Id, queryArticleType)
	model := UserArticleModel{Articles: articles, Categories: categories, UserId: user.Id}
	context.RenderPage(NewPageModel(context, user.UserName, model), "templates/layout.html", "templates/userLayout.html", "templates/userArticle.html")
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
	articles := GetStorage(context).GetArticles(1, 0, key, false, "")
	context.RenderPage(NewPageModel(context, config.WebsiteName, struct {
		Key      string
		Articles []models.ArticleDto
	}{Key: key, Articles: articles}), "templates/layout.html", "templates/leftRightLayout.html", "templates/articlelist.html")
}

type ArticleEditModel struct {
	CategoryList []models.CategoryDto
	Article      models.ArticleDto
	UserId       int
}

func ArticleEdit(context *goweb.Context) {
	categoryList := GetStorage(context).GetCategories(MustGetLoginUser(context).Id, 0)
	model := ArticleEditModel{CategoryList: categoryList, UserId: MustGetLoginUser(context).Id}
	if article, ok := context.Data["article"].(*models.ArticleDto); ok {
		model.Article = *article
	} else {
		id := context.Request.URL.Query().Get("id")
		if id != "" {
			intId, err := strconv.Atoi(id)
			if err != nil {
				panic(err)
			}
			article := GetStorage(context).GetArticle(intId)
			if article.ArticleType == 3 {
				http.Redirect(context.Writer, context.Request, PATH_ARTICLELOCK+"?id="+strconv.Itoa(article.Id)+"&t=1", 302)
				return
			}
			model.Article = *article
		}
	}
	context.RenderPage(NewPageModel(context, "写文章", model), "templates/layout.html", "templates/articleedit.html")
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
		intId = GetStorage(context).NewArticle(title, summary, html, content, MustGetLoginUser(context).Id, intArticleType, intCategoryId, config.PostKey, cover)
	} else {
		GetStorage(context).UpdateArticle(intId, title, summary, html, content, intArticleType, categoryId, config.PostKey, MustGetLoginUser(context).Id, cover)
	}
	context.Success(intId)
}

type ArticleModel struct {
	Article  *models.ArticleDto
	Html     template.HTML
	Readonly bool
}

func Article(context *goweb.Context) {
	re := regexp.MustCompile(`\d+$`)
	id, _ := strconv.Atoi(re.FindString(context.Request.URL.Path))
	article := GetStorage(context).GetArticle(id)
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
	context.RenderPage(NewPageModel(context, article.Title, model), "templates/layout.html", "templates/article.html")
}

type ArticleLockModel struct {
	Id    string
	Error string
	Type  string
}

func ArticleLock(context *goweb.Context) {
	id := context.Request.URL.Query().Get("id")
	t := context.Request.URL.Query().Get("t")
	context.RenderPage(NewPageModel(context, "lock", ArticleLockModel{Id: id, Type: t}), "templates/layout.html", "templates/articlelock.html")
}
func ArticleLockPost(context *goweb.Context) {
	id := context.Request.PostForm.Get("id")
	pwd := context.Request.PostForm.Get("pwd")
	t, _ := strconv.Atoi(context.Request.PostForm.Get("type"))
	intId, _ := strconv.Atoi(id)
	article := GetStorage(context).GetArticle(intId)
	if article.UserId != MustGetLoginUser(context).Id {
		context.ShowErrorPage(http.StatusUnauthorized, "")
		return
	}
	if !common.Md5Check(*GetStorage(context).GetUser(article.UserId).Level2pwd, pwd) {
		context.RenderPage(NewPageModel(context, "lock", ArticleLockModel{Id: id, Type: strconv.Itoa(t), Error: "二级密码错误"}), "templates/layout.html", "templates/articlelock.html")
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
		context.RenderPage(NewPageModel(context, article.Title, model), "templates/layout.html", "templates/article.html")
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
	user, err := GetStorage(ctx).GetUserByOP(id, iss)
	if err != nil {
		GetStorage(ctx).NewUser(name, iss, id, email)
		user, err = GetStorage(ctx).GetUserByOP(id, iss)
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

func LogoutPost(ctx *goweb.Context) {
	auth.Logout(ctx, func(id_token string) {
		http.Redirect(ctx.Writer, ctx.Request, config.OAuthLogoutUrl+"?post_logout_redirect_uri="+config.OAuthLogoutRedirectUrl+"&id_token_hint="+id_token, 302)
	})
}

// func sendValidateEmail(context *goweb.Context, userId int) {
// 	protocol := "https"
// 	user := GetStorage(context).GetUser(userId)
// 	if user.Email == nil {
// 		panic("the user does not have a email")
// 	}
// 	activateAddr := protocol + "://" + context.Request.Host + PATH_EMAILVALIDATE + "?email=" + *user.Email + "&code=" + url.QueryEscape(*user.SecurityStamp)
// 	emailSender.SendEmail(*user.Email, "邮箱激活", fmt.Sprintf("<html><body>"+
// 		"%s，您好:<br/><br/>"+
// 		"感谢您注册%s,您的登录邮箱为%s,请点击以下链接激活您的邮箱地址：<br/><br/>"+
// 		"<a href='%s'>%s</a><br/><br/>"+
// 		"如果以上链接无法访问，请将该网址复制并粘贴至浏览器窗口中直接访问。", user.UserName, config.WebsiteName, *user.Email, activateAddr, activateAddr)+
// 		"</body></html>")
// }

func CategoryList(context *goweb.Context) {
	categoryList := GetStorage(context).GetCategories(MustGetLoginUser(context).Id, 0)
	context.RenderPage(NewPageModel(context, "我的分类", categoryList), "templates/layout.html", "templates/categorylist.html")
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
	context.RenderPage(NewPageModel(context, title, model), "templates/layout.html", "templates/categoryedit.html")
}
func CategorySave(context *goweb.Context) {
	name := context.Request.PostForm.Get("name")
	id := context.Request.PostForm.Get("id")
	if id == "" {
		GetStorage(context).NewCategory(name, MustGetLoginUser(context).Id)
	} else {
		intId, _ := strconv.Atoi(id)
		GetStorage(context).UpdateCategory(name, intId, MustGetLoginUser(context).Id)
	}
	context.Success(nil)
}
func CategoryDelete(context *goweb.Context) {
	id := context.Request.FormValue("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		panic(err)
	}
	GetStorage(context).CategoryDelete(intId)
	context.Success(nil)
}

func ArticleDelete(context *goweb.Context) {
	id := context.Request.FormValue("id")
	intId, _ := strconv.Atoi(id)
	GetStorage(context).ArticleDelete(intId, MustGetLoginUser(context).Id, config.PostKey)
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
	user := GetStorage(context).GetUser(MustGetLoginUser(context).Id)
	existLevel2Pwd := user.Level2pwd != nil
	model := SetLevelTwoPwdModel{Settings: GetSettingsModel(context.Request.URL.Path)}
	model.ExistLevel2Pwd = existLevel2Pwd
	context.RenderPage(NewPageModel(context, "设置二级密码", model), "templates/layout.html", "templates/settingsLeftBar.html", "templates/setleveltwopwd.html")
}
func SetLevelTwoPwdPost(context *goweb.Context) {
	oldPwd := context.Request.PostForm.Get("oldPwd")
	newPwd := context.Request.PostForm.Get("newPwd")
	user := GetStorage(context).GetUser(MustGetLoginUser(context).Id)
	if user.Level2pwd != nil {
		if !common.Md5Check(*user.Level2pwd, oldPwd) {
			context.Failed("旧密码有误")
			return
		}
	}
	GetStorage(context).SetLevelTwoPwd(user.Id, newPwd)
	context.Success(nil)
}

type ProfileModel struct {
	Settings []*SettingsItemModel
}

func Profile(context *goweb.Context) {
	context.RenderPage(NewPageModel(context, "个人资料", ProfileModel{Settings: GetSettingsModel(context.Request.URL.Path)}), "templates/layout.html", "templates/settingsLeftBar.html", "templates/profile.html")
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
func Chat(context *goweb.Context) {
	context.RenderPage(NewPageModel(context, "IM", config.UseHttps), "templates/layout.html", "templates/chat.html")
}
