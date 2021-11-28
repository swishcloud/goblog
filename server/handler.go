package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/swishcloud/goblog/internal"
	"github.com/swishcloud/goblog/storage/models"
	"github.com/swishcloud/gostudy/common"
	"github.com/swishcloud/gostudy/keygenerator"
	"github.com/swishcloud/goweb"
	"github.com/swishcloud/goweb/auth"
	"golang.org/x/oauth2"
)

const session_user_key = "session_user"
const csrf_state_cookie_name = "crft_state"
const (
	PATH_ARTICLELIST             = "/articlelist"
	PATH_ARTICLEEDIT             = "/articleedit"
	PATH_ARTICLESAVE             = "/articlesave"
	PATH_ARTICLEDELETE           = "/articledelete"
	PATH_ARTICLELOCK             = "/articlelock"
	PATH_ARTICLEHISTORIES        = "/articlehistories"
	PATH_LOGIN                   = "/login"
	PATH_LOGIN_CALLBACK          = "/login-callback"
	PATH_LOGOUT                  = "/logout"
	PATH_CATEGORYLIST            = "/categories"
	PATH_CATEGORYEDIT            = "/categoryedit"
	PATH_CATEGORYSAVE            = "/categorysave"
	PATH_CATEGORYDELETE          = "/categorydelete"
	PATH_SETLEVELTWOPWD          = "/setlevel2pwd"
	PATH_PROFILE                 = "/profile"
	PATH_UPLOAD                  = "/upload"
	PATH_FriendlyLink            = "/friendly-link"
	PATH_FriendlyLinkApply       = "/friendly-link-apply"
	PATH_FriendlyLinkApplyList   = "/friendly-link-apply-list"
	PATH_FriendlyLinkApplyActive = "/friendly_link_apply_active"
	PATH_WEBSOCKET               = "/ws"
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
	auth.GET(PATH_ARTICLEHISTORIES, s.ArticleHistories())
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
	group.GET(PATH_FriendlyLink, s.FriendlyLink())
	group.GET(PATH_FriendlyLinkApply, s.FriendlyLinkApply())
	group.POST(PATH_FriendlyLinkApply, s.FriendlyLinkApplyPOST())
	group.GET(PATH_FriendlyLinkApplyList, s.FriendlyLinkApplyList())
	group.PUT(PATH_FriendlyLinkApplyActive, s.FriendlyLinkApplyActivePUT())
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
	FileCache        *FileCache
}

func (s *GoBlogServer) AuthMiddleware() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		if !auth.HasLoggedIn(s.rac, ctx, s.config.OAUTH2Config, s.config.IntrospectTokenURL, s.skip_tls_verify) {
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

func (m UserArticleModel) GetCategoryUrl(id int) string {
	return fmt.Sprintf("/user/%d/article?category=%d", m.UserId, id)
}

func (s *GoBlogServer) GetLoginUser(ctx *goweb.Context) (*models.UserDto, error) {
	if s, err := auth.GetSessionByToken(s.rac, ctx, s.config.OAUTH2Config, s.config.IntrospectTokenURL, s.skip_tls_verify); err != nil {
		internal.Logger.Println("GET USER FAILED:", err)
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
		loginUser := s.MustGetLoginUser(ctx)
		category, err := strconv.Atoi(ctx.Request.URL.Query().Get("category"))
		var categoryId *int
		if err != nil {
			categoryId = &s.GetStorage(ctx).GetCategories(loginUser.Id, 0)[0].Id
		} else {
			categoryId = &category
		}
		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(ctx.Request.URL.Path))
		user := s.GetStorage(ctx).GetUser(id)
		var queryArticleType int
		if loginUser.Id == user.Id {
			queryArticleType = 0
		} else {
			queryArticleType = 1
		}
		articles := s.GetStorage(ctx).GetArticles(queryArticleType, user.Id, "", categoryId, s.config.PostKey, nil)
		categories := s.GetStorage(ctx).GetCategories(user.Id, queryArticleType)
		model := UserArticleModel{Articles: articles, Categories: categories, UserId: user.Id}
		ctx.RenderPage(s.NewPageModel(ctx, user.UserName, model), "templates/layout.html", "templates/userLayout.html", "templates/userArticle.html")
	}
}
func (s *GoBlogServer) ArticleHistories() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id, err := strconv.Atoi(ctx.Request.URL.Query().Get("id"))
		if err != nil {
			panic(err)
		}
		user := s.MustGetLoginUser(ctx)
		articles := s.GetStorage(ctx).GetArticles(4, user.Id, "", nil, s.config.PostKey, &id)
		data := struct {
			Articles []models.ArticleDto
		}{
			Articles: articles,
		}
		model := s.NewPageModel(ctx, "HISTORY", data)
		ctx.RenderPage(model, "templates/layout.html", "templates/userLayout.html", "templates/articlehistories.html")
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
		keys := ctx.Request.URL.Query()["key"]
		var key string
		if len(keys) > 0 {
			key = keys[0]
		} else {
			key = ""
		}
		articles := s.GetStorage(ctx).GetArticles(1, 0, key, nil, s.config.PostKey, nil)
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
		model.Article.ArticleType = 2
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
				if article.ArticleType == 4 {
					panic("YOU CAN NOT EDIT A BACKUP KIND OF ARTICLE")
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
		shareDeadlineTime := ctx.Request.PostForm.Get("shareDeadlineTime") + ":00Z"
		var shareDeadlineTimePtr *time.Time
		if articleType == "5" {
			tmp, err := time.Parse(time.RFC3339, shareDeadlineTime)
			if err != nil {
				panic(err)
			}
			c, err := ctx.Request.Cookie("tom")
			if err != nil {
				panic(err)
			}
			tom, err := strconv.Atoi(c.Value)
			if err != nil {
				panic(err)
			}
			tmp = tmp.Add(time.Duration(int64(time.Minute) * int64(tom))).UTC()
			shareDeadlineTimePtr = &tmp
		}
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
			intId = s.GetStorage(ctx).NewArticle(title, summary, html, content, s.MustGetLoginUser(ctx).Id, intArticleType, shareDeadlineTimePtr, intCategoryId, s.config.PostKey, cover, nil, &now, nil, "New article by user")
		} else {
			s.GetStorage(ctx).UpdateArticle(intId, title, summary, html, content, intArticleType, shareDeadlineTimePtr, categoryId, s.config.PostKey, s.MustGetLoginUser(ctx).Id, cover)
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
			s.showErrorPage(ctx, http.StatusNotFound, "404 THE ARTICLE DOES NOT EXIST")
			return
		}
		model := ArticleModel{Article: article, Readonly: true}
		if !auth.HasLoggedIn(s.rac, ctx, s.config.OAUTH2Config, s.config.IntrospectTokenURL, s.skip_tls_verify) {
			if article.ArticleType == 1 {
				//public article, do nothing
			} else if article.ArticleType == 5 {
				//shared article, check deadline
				if !article.ShareDeadlineTime.After(time.Now().UTC()) {
					s.showErrorPage(ctx, http.StatusForbidden, "THE LINK IS NO LONGER VALID")
					return
				}
			} else {
				s.showErrorPage(ctx, http.StatusForbidden, "CAN NOT ACCESS THIS SORT OF ARTICLE")
				return
			}
		} else {
			loginUserId := s.MustGetLoginUser(ctx).Id
			if article.ArticleType != 1 {
				if article.UserId != loginUserId {
					s.showErrorPage(ctx, http.StatusUnauthorized, "NO PERMISSION")
					return
				}
			}
			if article.ArticleType == 3 {
				http.Redirect(ctx.Writer, ctx.Request, PATH_ARTICLELOCK+"?id="+strconv.Itoa(article.Id)+"&t=2", 302)
				return
			}
			if article.UserId == loginUserId && article.ArticleType != 4 {
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
			s.showErrorPage(ctx, http.StatusUnauthorized, "NO PERMISSION")
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
		state, err := keygenerator.NewKey(20, false, false, false, false)
		if err != nil {
			panic(err)
		}
		state = base64.URLEncoding.EncodeToString([]byte(state))
		cookie := http.Cookie{Name: csrf_state_cookie_name, Value: state, Path: "/"}
		http.SetCookie(ctx.Writer, &cookie)
		if err != nil {
			panic(err)
		}
		url := s.config.OAUTH2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
		http.Redirect(ctx.Writer, ctx.Request, url, 302)
	}
}

func (s *GoBlogServer) LoginCallback() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		state := ctx.Request.URL.Query().Get("state")
		common.DelCookie(ctx.Writer, csrf_state_cookie_name)
		if cookie, err := ctx.Request.Cookie(csrf_state_cookie_name); err != nil {
			ctx.Failed("state cookie does not present")
			return
		} else {
			if cookie.Value != state {
				ctx.Failed("csrf verification failed")
				return
			}
		}
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
		internal.Logger.Println("A user logged successfully:" + user.UserName)
		http.Redirect(ctx.Writer, ctx.Request, "/", 302)
	}
}

func (s *GoBlogServer) LogoutPost() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		auth.Logout(s.rac, ctx, s.config.OAUTH2Config, s.config.IntrospectTokenURL, s.skip_tls_verify, func(id_token string) {
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
		dirPath, err := s.config.ImageDirPath()
		if err != nil {
			panic(err)
		}
		path := path.Join(dirPath, uuid)
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

func (s *GoBlogServer) FriendlyLink() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		links, err := s.MemoryCache.FriendlyLinks(true)
		if err != nil {
			panic(err)
		}
		ctx.RenderPage(s.NewPageModel(ctx, "友情链接", links), "templates/layout.html", "templates/friendlylink.html")

	}
}
func (s *GoBlogServer) FriendlyLinkApply() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		ctx.RenderPage(s.NewPageModel(ctx, "友链申请", nil), "templates/layout.html", "templates/leftRightLayout.html", "templates/friendlylinkapply.html")

	}
}
func (s *GoBlogServer) FriendlyLinkApplyPOST() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		name := ctx.Request.FormValue("name")
		description := ctx.Request.FormValue("description")
		pageLink := strings.TrimSpace(ctx.Request.FormValue("pageLink"))
		err := s.checkFriendlyLink(pageLink)
		if err != nil {
			ctx.Failed(err.Error())
			return
		}
		u, err := url.Parse(pageLink)
		if err != nil {
			ctx.Failed("The URL format is error")
			return
		}
		s.GetStorage(ctx).NewFriendlyLink(name, u.Host, description, pageLink)
		ctx.Success(nil)
	}
}
func (s *GoBlogServer) checkFriendlyLink(pageLink string) error {
	u, err := url.Parse(pageLink)
	if err != nil {
		return errors.New("The URL format is error")
	}
	if strings.ToLower(u.Scheme) != "https" {
		return errors.New("Only supports HTTPS scheme")
	}

	home_url := "https://" + u.Host

	err = checkIfContaineLink(s.rac, home_url, u.Path)
	if err != nil {
		return err
	}
	err = checkIfContaineLink(s.rac, pageLink, "https://"+s.config.Website_domain)
	if err != nil {
		return err
	}
	return nil
}
func checkIfContaineLink(rac *common.RestApiClient, url, link string) error {
	rar := common.NewRestApiRequest("GET", url, nil)
	resp, err := rac.Do(rar)
	if err != nil || resp.StatusCode != 200 {
		return errors.New("cannot access page at " + url + " normally")
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New("cannot access page at " + url + " normally")
	}
	html := string(b)
	regex := regexp.MustCompile("<a.+?</a>")
	found := regex.FindAllString(html, -1)
	ok := false
	for _, item := range found {
		if strings.Index(item, link+`"`) != -1 {
			ok = true
		}
	}
	if !ok {
		return errors.New("The page at " + url + " does not contain LINK of " + link)
	}
	return nil
}

func (s *GoBlogServer) FriendlyLinkApplyList() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		links, err := s.MemoryCache.FriendlyLinks(true)
		if err != nil {
			panic(err)
		}
		ctx.RenderPage(s.NewPageModel(ctx, "友链申请列表", links), "templates/layout.html", "templates/friendlylinkapplylist.html")

	}
}
func (s *GoBlogServer) FriendlyLinkApplyActivePUT() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.FormValue("id")
		val := ctx.Request.FormValue("val")
		active, err := strconv.ParseBool(val)
		if err != nil {
			panic(err)
		}
		s.GetStorage(ctx).SetFriendlyLinkActiveStatus(id, active)
		ctx.Success(nil)
	}
}
func (s *GoBlogServer) ErrorPage() goweb.ErrorPageFunc {
	return func(context *goweb.Context, status int, desc string) {
		context.RenderPage(s.NewPageModel(context, string(status), ErrorPageModel{Status: status, Desc: desc}), "templates/layout.html", "templates/error.html")
	}

}
