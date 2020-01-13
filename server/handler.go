package server

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
	"github.com/swishcloud/goblog/common"
	"github.com/swishcloud/goblog/storage/models"
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
)

func (s *GoBlogServer) BindHandlers() {
	group := s.engine.RouterGroup
	auth := group.Group()
	auth.Use(s.AuthMiddleware())

	group.GET("/", s.ArticleList())
	group.RegexMatch(regexp.MustCompile(`^/u/\d+/post/\d+$`), s.Article())
	group.RegexMatch(regexp.MustCompile(`^/user/\d+/article$`), s.UserArticle())
	group.RegexMatch(regexp.MustCompile(`/static/.+`), func(context *goweb.Context) {
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))).ServeHTTP(context.Writer, context.Request)
	})
	group.RegexMatch(regexp.MustCompile(`/src/.+`), func(context *goweb.Context) {
		http.StripPrefix("/src/", http.FileServer(http.Dir(s.config.FileLocation))).ServeHTTP(context.Writer, context.Request)
	})
	group.GET(PATH_ARTICLELIST, s.ArticleList())
	auth.GET(PATH_ARTICLEEDIT, s.ArticleEdit())
	auth.POST(PATH_ARTICLESAVE, s.ArticleSave())
	auth.POST(PATH_ARTICLEDELETE, s.ArticleDelete())
	auth.GET(PATH_ARTICLELOCK, s.ArticleLock())
	auth.POST(PATH_ARTICLELOCK, s.ArticleLockPost())
	group.GET(PATH_LOGIN, s.Login())
	group.GET(PATH_LOGIN_CALLBACK, s.LoginCallback())
	group.POST(PATH_LOGOUT, s.LogoutPost())
	auth.GET(PATH_CATEGORYLIST, s.CategoryList())
	auth.GET(PATH_CATEGORYEDIT, s.CategoryEdit())
	auth.POST(PATH_CATEGORYSAVE, s.CategorySave())
	auth.POST(PATH_CATEGORYDELETE, s.CategoryDelete())
	auth.GET(PATH_SETLEVELTWOPWD, s.SetLevelTwoPwd())
	auth.POST(PATH_SETLEVELTWOPWD, s.SetLevelTwoPwdPost())
	auth.GET(PATH_PROFILE, s.Profile())
	auth.POST(PATH_UPLOAD, s.Upload())
}

type PageModel struct {
	User             *models.UserDto
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

func (s *GoBlogServer) AuthMiddleware() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		if !auth.HasLoggedIn(ctx, s.config.IntrospectTokenURL) {
			if ctx.Request.Method == "GET" {
				http.Redirect(ctx.Writer, ctx.Request, PATH_LOGIN+"?redirectUri="+ctx.Request.RequestURI, 302)
			} else {
				ctx.Failed("not logged in")
			}
			ctx.Abort()
		}
		ctx.Next()
	}
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

func (s *GoBlogServer) GetLoginUser(ctx *goweb.Context) (*models.UserDto, error) {
	if s, err := auth.GetSessionByToken(ctx, s.config.IntrospectTokenURL); err != nil {
		return nil, err
	} else {
		if u, ok := s.Data[session_user_key].(*models.UserDto); ok {
			return u, nil
		}
		return nil, errors.New("user not logged in")
	}
}
func (s *GoBlogServer) MustGetLoginUser(ctx *goweb.Context) *models.UserDto {
	u, err := s.GetLoginUser(ctx)
	if err != nil {
		panic(err)
	}
	return u
}
func (s *GoBlogServer) UserArticle() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		category := ctx.Request.URL.Query().Get("category")
		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(ctx.Request.URL.Path))
		user := s.GetStorage(ctx).GetUser(id)
		loginUser, err := s.GetLoginUser(ctx)
		if err != nil {
			http.Redirect(ctx.Writer, ctx.Request, PATH_LOGIN, 302)
		}
		var queryArticleType int
		if loginUser.Id == user.Id {
			queryArticleType = 0
		} else {
			queryArticleType = 1
		}
		articles := s.GetStorage(ctx).GetArticles(queryArticleType, user.Id, "", category, s.config.PostKey)
		categories := s.GetStorage(ctx).GetCategories(user.Id, queryArticleType)
		model := UserArticleModel{Articles: articles, Categories: categories, UserId: user.Id}
		ctx.RenderPage(s.NewPageModel(ctx, user.UserName, model), "templates/layout.html", "templates/userLayout.html", "templates/userArticle.html")
	}
}

type ArticleListItemModel struct {
	Id         int
	Title      string
	Content    string
	InsertTime string
}

func (s *GoBlogServer) ArticleList() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		keys, _ := ctx.Request.URL.Query()["key"]
		var key string
		if len(keys) > 0 {
			key = keys[0]
		} else {
			key = ""
		}
		articles := s.GetStorage(ctx).GetArticles(1, 0, key, "", s.config.PostKey)
		ctx.RenderPage(s.NewPageModel(ctx, s.config.WebsiteName, struct {
			Key      string
			Articles []models.ArticleDto
		}{Key: key, Articles: articles}), "templates/layout.html", "templates/leftRightLayout.html", "templates/articlelist.html")
	}
}

type ArticleEditModel struct {
	CategoryList []models.CategoryDto
	Article      models.ArticleDto
	UserId       int
}

func (s *GoBlogServer) ArticleEdit() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		categoryList := s.GetStorage(ctx).GetCategories(s.MustGetLoginUser(ctx).Id, 0)
		model := ArticleEditModel{CategoryList: categoryList, UserId: s.MustGetLoginUser(ctx).Id}
		if article, ok := ctx.Data["article"].(*models.ArticleDto); ok {
			model.Article = *article
		} else {
			id := ctx.Request.URL.Query().Get("id")
			if id != "" {
				intId, err := strconv.Atoi(id)
				if err != nil {
					panic(err)
				}
				article := s.GetStorage(ctx).GetArticle(intId, s.config.PostKey)
				if article.ArticleType == 3 {
					http.Redirect(ctx.Writer, ctx.Request, PATH_ARTICLELOCK+"?id="+strconv.Itoa(article.Id)+"&t=1", 302)
					return
				}
				model.Article = *article
			}
		}
		ctx.RenderPage(s.NewPageModel(ctx, "写文章", model), "templates/layout.html", "templates/articleedit.html")
	}
}

func (s *GoBlogServer) ArticleSave() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		ctx.Request.ParseForm()
		id := ctx.Request.PostForm.Get("id")
		title := ctx.Request.PostForm.Get("title")
		content := ctx.Request.PostForm.Get("content")
		categoryId := ctx.Request.PostForm.Get("categoryId")
		articleType := ctx.Request.PostForm.Get("type")
		html := goweb.SanitizeHtml(ctx.Request.PostForm.Get("html"))
		summary := ctx.Request.PostForm.Get("summary")

		c := ctx.Request.PostForm.Get("cover")
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
		now := time.Now().UTC()
		if intId == 0 {
			intId = s.GetStorage(ctx).NewArticle(title, summary, html, content, s.MustGetLoginUser(ctx).Id, intArticleType, intCategoryId, s.config.PostKey, cover, nil, &now, nil, "New article by user")
		} else {
			s.GetStorage(ctx).UpdateArticle(intId, title, summary, html, content, intArticleType, categoryId, s.config.PostKey, s.MustGetLoginUser(ctx).Id, cover)
		}
		ctx.Success(intId)
	}
}

type ArticleModel struct {
	Article  *models.ArticleDto
	Html     template.HTML
	Readonly bool
}

func (s *GoBlogServer) Article() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		re := regexp.MustCompile(`\d+$`)
		id, _ := strconv.Atoi(re.FindString(ctx.Request.URL.Path))
		article := s.GetStorage(ctx).GetArticle(id, s.config.PostKey)
		if article == nil {
			ctx.ShowErrorPage(http.StatusNotFound, "page not found")
			return
		}
		model := ArticleModel{Article: article, Readonly: true}
		if !auth.HasLoggedIn(ctx, s.config.IntrospectTokenURL) {
			if article.ArticleType != 1 {
				http.Redirect(ctx.Writer, ctx.Request, PATH_LOGIN, 302)
				return
			}
		} else {
			loginUserId := s.MustGetLoginUser(ctx).Id
			if article.ArticleType != 1 {
				if article.UserId != loginUserId {
					ctx.ShowErrorPage(http.StatusUnauthorized, "")
					return
				}
			}
			if article.ArticleType == 3 {
				http.Redirect(ctx.Writer, ctx.Request, PATH_ARTICLELOCK+"?id="+strconv.Itoa(article.Id)+"&t=2", 302)
				return
			}
			if article.UserId == loginUserId {
				model.Readonly = false
			}
		}
		model.Html = template.HTML(model.Article.Html)
		ctx.RenderPage(s.NewPageModel(ctx, article.Title, model), "templates/layout.html", "templates/article.html")
	}
}

type ArticleLockModel struct {
	Id    string
	Error string
	Type  string
}

func (s *GoBlogServer) ArticleLock() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.URL.Query().Get("id")
		t := ctx.Request.URL.Query().Get("t")
		ctx.RenderPage(s.NewPageModel(ctx, "lock", ArticleLockModel{Id: id, Type: t}), "templates/layout.html", "templates/articlelock.html")
	}
}

func (s *GoBlogServer) ArticleLockPost() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.PostForm.Get("id")
		pwd := ctx.Request.PostForm.Get("pwd")
		t, _ := strconv.Atoi(ctx.Request.PostForm.Get("type"))
		intId, _ := strconv.Atoi(id)
		article := s.GetStorage(ctx).GetArticle(intId, s.config.PostKey)
		if article.UserId != s.MustGetLoginUser(ctx).Id {
			ctx.ShowErrorPage(http.StatusUnauthorized, "")
			return
		}
		if !common.Md5Check(*s.GetStorage(ctx).GetUser(article.UserId).Level2pwd, pwd) {
			ctx.RenderPage(s.NewPageModel(ctx, "lock", ArticleLockModel{Id: id, Type: strconv.Itoa(t), Error: "二级密码错误"}), "templates/layout.html", "templates/articlelock.html")
			return
		}
		if t == 1 {
			ctx.Data["article"] = article
			s.ArticleEdit()(ctx)
		} else if t == 2 {
			model := ArticleModel{Article: article, Readonly: false}
			model.Html = template.HTML(article.Html)
			ctx.RenderPage(s.NewPageModel(ctx, article.Title, model), "templates/layout.html", "templates/article.html")
		}
	}
}

func (s *GoBlogServer) Login() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		url := s.config.OAUTH2Config.AuthCodeURL("state-string", oauth2.AccessTypeOffline)
		http.Redirect(ctx.Writer, ctx.Request, url, 302)
	}
}

func (s *GoBlogServer) LoginCallback() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		code := ctx.Request.URL.Query().Get("code")
		token, err := s.config.OAUTH2Config.Exchange(context.Background(), code)
		if err != nil {
			ctx.Writer.Write([]byte(err.Error()))
			return
		}
		session := auth.Login(ctx, token, s.config.JWKJsonUrl)
		id := session.Claims["sub"].(string)
		name := session.Claims["name"].(string)
		iss := session.Claims["iss"].(string)
		email := session.Claims["email"].(string)
		avatar := session.Claims["avatar"].(string)
		user, err := s.GetStorage(ctx).GetUserByOP(id, iss)
		if err != nil {
			s.GetStorage(ctx).NewUser(name, iss, id, email, avatar)
			user, err = s.GetStorage(ctx).GetUserByOP(id, iss)
			if err != nil {
				panic(err)
			}
		}
		session.Data[session_user_key] = user
		http.Redirect(ctx.Writer, ctx.Request, "/", 302)
	}
}

func (s *GoBlogServer) LogoutPost() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		auth.Logout(ctx, s.config.IntrospectTokenURL, func(id_token string) {
			http.Redirect(ctx.Writer, ctx.Request, s.config.OAuthLogoutUrl+"?post_logout_redirect_uri="+s.config.OAuthLogoutRedirectUrl+"&id_token_hint="+id_token, 302)
		})
	}
}
func (s *GoBlogServer) CategoryList() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		categoryList := s.GetStorage(ctx).GetCategories(s.MustGetLoginUser(ctx).Id, 0)
		ctx.RenderPage(s.NewPageModel(ctx, "我的分类", categoryList), "templates/layout.html", "templates/categorylist.html")
	}
}

func (s *GoBlogServer) CategoryEdit() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id, _ := strconv.Atoi(ctx.Request.URL.Query().Get("id"))
		category := s.GetStorage(ctx).GetCategory(id)
		title := "编辑分类"
		if category == nil {
			category = new(models.CategoryDto)
			title = "新增分类"
		}
		ctx.RenderPage(s.NewPageModel(ctx, title, category), "templates/layout.html", "templates/categoryedit.html")
	}
}

func (s *GoBlogServer) CategorySave() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		name := ctx.Request.PostForm.Get("name")
		id, err := strconv.Atoi(ctx.Request.PostForm.Get("id"))
		if err != nil {
			panic(err)
		}
		if id == 0 {
			s.GetStorage(ctx).NewCategory(name, s.MustGetLoginUser(ctx).Id)
		} else {
			s.GetStorage(ctx).UpdateCategory(name, id, s.MustGetLoginUser(ctx).Id)
		}
		ctx.Success(nil)
	}
}

func (s *GoBlogServer) CategoryDelete() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.FormValue("id")
		intId, err := strconv.Atoi(id)
		if err != nil {
			panic(err)
		}
		s.GetStorage(ctx).CategoryDelete(intId)
		ctx.Success(nil)
	}
}

func (s *GoBlogServer) ArticleDelete() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.FormValue("id")
		intId, _ := strconv.Atoi(id)
		s.GetStorage(ctx).ArticleDelete(intId, s.MustGetLoginUser(ctx).Id, s.config.PostKey)
		ctx.Success(nil)
	}
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

func (s *GoBlogServer) SetLevelTwoPwd() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		user := s.GetStorage(ctx).GetUser(s.MustGetLoginUser(ctx).Id)
		existLevel2Pwd := user.Level2pwd != nil
		model := SetLevelTwoPwdModel{Settings: GetSettingsModel(ctx.Request.URL.Path)}
		model.ExistLevel2Pwd = existLevel2Pwd
		ctx.RenderPage(s.NewPageModel(ctx, "设置二级密码", model), "templates/layout.html", "templates/settingsLeftBar.html", "templates/setleveltwopwd.html")
	}
}

func (s *GoBlogServer) SetLevelTwoPwdPost() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		oldPwd := ctx.Request.PostForm.Get("oldPwd")
		newPwd := ctx.Request.PostForm.Get("newPwd")
		user := s.GetStorage(ctx).GetUser(s.MustGetLoginUser(ctx).Id)
		if user.Level2pwd != nil {
			if !common.Md5Check(*user.Level2pwd, oldPwd) {
				ctx.Failed("旧密码有误")
				return
			}
		}
		s.GetStorage(ctx).SetLevelTwoPwd(user.Id, newPwd)
		ctx.Success(nil)
	}
}

type ProfileModel struct {
	Settings []*SettingsItemModel
}

func (s *GoBlogServer) Profile() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		ctx.RenderPage(s.NewPageModel(ctx, "个人资料", ProfileModel{Settings: GetSettingsModel(ctx.Request.URL.Path)}), "templates/layout.html", "templates/settingsLeftBar.html", "templates/profile.html")

	}
}
func (s *GoBlogServer) Upload() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		file, fileHeader, err := ctx.Request.FormFile("image")
		if err != nil {
			panic(err)
		}
		uuid := uuid.New().String() + ".png"
		path := s.config.FileLocation + `/image/` + uuid
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
		ctx.Writer.Header().Add("Content-Type", "application/json")
		ctx.Writer.Write(json)
	}
}

func (s *GoBlogServer) ErrorPage() goweb.ErrorPageFunc {
	return func(context *goweb.Context, status int, desc string) {
		context.RenderPage(s.NewPageModel(context, string(status), ErrorPageModel{Status: status, Desc: desc}), "templates/layout.html", "templates/error.html")
	}

}
