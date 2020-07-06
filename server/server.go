package server

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/swishcloud/goblog/storage"
	"github.com/swishcloud/gostudy/common"
	"github.com/swishcloud/goweb"
	"golang.org/x/oauth2"
)

type GoBlogServer struct {
	engine          *goweb.Engine
	storage         storage.Storage
	config          *Config
	skip_tls_verify bool
	httpClient      *http.Client
	rac             *common.RestApiClient
}

func NewGoBlogServer(configPath string, skip_tls_verify bool) *GoBlogServer {
	s := new(GoBlogServer)
	s.rac = common.NewRestApiClient(skip_tls_verify)
	s.config = readConfig(configPath)
	s.skip_tls_verify = skip_tls_verify
	s.httpClient = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: skip_tls_verify}}}
	http.DefaultClient = s.httpClient
	s.engine = goweb.Default()
	s.engine.ConcurrenceNumSem = make(chan int, s.config.ConcurrenceNum)
	s.engine.WM.HandlerWidget = &HandlerWidget{s: s}
	s.BindHandlers()
	return s
}
func (s *GoBlogServer) Serve() {
	log.Println("listening on:", s.config.Host)
	err := http.ListenAndServe(s.config.Host, s.engine)
	if err != nil {
		log.Fatal(err)
	}
}

func (server *GoBlogServer) GetStorage(ctx *goweb.Context) storage.Storage {
	m := ctx.Data["storage"]
	if m == nil {
		m = storage.NewSQLManager(server.config.SqlDataSourceName)
		ctx.Data["storage"] = m
	}
	return m.(storage.Storage)
}

func (s *GoBlogServer) NewPageModel(ctx *goweb.Context, title string, data interface{}) *PageModel {
	m := &PageModel{WebSiteName: s.config.WebsiteName, Data: data, LastUpdateTime: s.config.LastUpdateTime, MobileCompatible: true, Config: *s.config}
	m.Path = ctx.Request.URL.Path
	m.RequestUri = ctx.Request.RequestURI
	m.Request = ctx.Request
	u, err := s.GetLoginUser(ctx)
	if err == nil {
		m.User = u
	}
	n := time.Now()
	m.Duration = int(n.Sub(ctx.CT) / time.Millisecond)
	m.PageTitle = title + " - " + s.config.WebsiteName
	return m
}

type Config struct {
	FileLocation      string
	Host              string
	SqlDataSourceName string
	WebsiteName       string
	Key               string
	PostKey           string
	ConcurrenceNum    int
	SmtpUsername      string
	SmtpPassword      string
	SmtpAddr          string
	UseHttps          bool
	OAuthClientId     string
	OAuthTokenUrl     string
	OAuthAuthUrl      string
	OAuthSecret       string

	//not read from configuration file
	LastUpdateTime string

	OAUTH2Config           *oauth2.Config
	JWKJsonUrl             string
	OAuthLogoutRedirectUrl string
	OAuthLogoutUrl         string
	IntrospectTokenURL     string
}

func readConfig(filePath string) *Config {
	file, _ := os.Open(filePath)
	defer file.Close()
	dec := json.NewDecoder(file)
	var c Config
	dec.Decode(&c)

	info, err := file.Stat()
	if err != nil {
		panic(err)
	}
	tm := info.ModTime().Local()
	c.LastUpdateTime = tm.Format("2006-01-02 15:04:05")

	c.OAUTH2Config = &oauth2.Config{
		ClientID:     c.OAuthClientId,
		ClientSecret: c.OAuthSecret,
		Scopes:       []string{"offline", "openid", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  c.OAuthAuthUrl,
			TokenURL: c.OAuthTokenUrl,
		},
	}

	return &c
}

type ErrorPageModel struct {
	Status int
	Desc   string
}

type HandlerWidget struct {
	s *GoBlogServer
}

func (*HandlerWidget) Pre_Process(ctx *goweb.Context) {
}
func (hw *HandlerWidget) Post_Process(ctx *goweb.Context) {
	m := ctx.Data["storage"]
	if m != nil {
		if ctx.Ok {
			m.(storage.Storage).Commit()
		} else {
			m.(storage.Storage).Rollback()
		}
	}
	if ctx.Err != nil {
		data := struct {
			Desc string
		}{Desc: ctx.Err.Error()}
		model := hw.s.NewPageModel(ctx, "ERROR", data)
		ctx.RenderPage(model, "templates/layout.html", "templates/error.html")
	}
}
