package server

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"

	"github.com/swishcloud/goblog/internal"
	"github.com/swishcloud/goblog/storage/models"

	"github.com/swishcloud/goblog/storage"
	"github.com/swishcloud/gostudy/common"
	"github.com/swishcloud/gostudy/logger"
	"github.com/swishcloud/goweb"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

const cache_dir string = ".cache"

type GoBlogServer struct {
	engine          *goweb.Engine
	storage         storage.Storage
	config          *Config
	FileCache       *FileCache
	MemoryCache     *MemoryCache
	skip_tls_verify bool
	httpClient      *http.Client
	rac             *common.RestApiClient
}

func NewGoBlogServer(configPath string, skip_tls_verify bool) *GoBlogServer {
	s := new(GoBlogServer)
	s.rac = common.NewRestApiClient(skip_tls_verify)
	s.config = readConfig(configPath)
	s.FileCache = &FileCache{config: s.config}
	s.MemoryCache = &MemoryCache{server: s}
	err := s.FileCache.reload()
	if err != nil {
		panic(err)
	}
	s.skip_tls_verify = skip_tls_verify
	s.httpClient = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: skip_tls_verify}}}
	http.DefaultClient = s.httpClient
	s.engine = goweb.Default()
	s.engine.ConcurrenceNumSem = make(chan int, s.config.ConcurrenceNum)
	s.engine.WM.HandlerWidget = &HandlerWidget{s: s}
	internal.LoggerWriter = logger.NewFileConcurrentWriter(s.config.Log_file)
	internal.Logger = logger.NewLogger(internal.LoggerWriter, "GOBLOG")
	s.engine.Logger = logger.NewLogger(internal.LoggerWriter, "GOWEB")
	s.BindHandlers()
	return s
}
func (s *GoBlogServer) Serve() {
	go s.periodicTask()
	internal.Logger.Println("listening on:", s.config.Host)
	err := http.ListenAndServeTLS(s.config.Host, s.config.Tls_cert_file, s.config.Tls_key_file, s.engine)
	if err != nil {
		log.Fatal(err)
	}
}
func (s *GoBlogServer) periodicTask() {
	for {
		err := s.updateHomeWallpaper()
		if err != nil {
			internal.Logger.Println("update home wallpaper from Bing failed:", err)
		} else {
			internal.Logger.Println("update home wallpaper from Bing succeed")
		}
		time.Sleep(time.Minute * 60)
	}
}
func (s *GoBlogServer) updateHomeWallpaper() (err error) {
	defer func() {
		if tmp := recover(); err != nil {
			err = errors.New(fmt.Sprint(tmp))
		}
	}()
	url, err := common.GetBingHomeWallpaper(s.rac)
	if err != nil {
		return err
	}
	reg := regexp.MustCompile("id=[^&]+")
	filename := reg.FindString(url)
	runes := []rune(filename)
	filename = string(runes[3:])
	if filename != s.FileCache.HomeWallpaper {
		dirPath, err := s.config.ImageDirPath()
		if err != nil {
			return err
		}
		file_path := path.Join(dirPath, filename)
		err = common.DownloadFile(s.rac, url, file_path)
		if err != nil {
			return err
		}
		s.FileCache.HomeWallpaper = filename
		s.FileCache.Save()
	} else {
		internal.Logger.Println("home wallpaper is up to date,skip downloading")
	}
	return nil
}

func (server *GoBlogServer) showErrorPage(ctx *goweb.Context, status int, msg string) {
	data := struct {
		Desc string
	}{Desc: msg}
	model := server.NewPageModel(ctx, "ERROR", data)
	ctx.Writer.WriteHeader(status)
	ctx.RenderPage(model, "templates/layout.html", "templates/error.html")
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
	m := &PageModel{WebSiteName: s.config.WebsiteName, Data: data, LastUpdateTime: s.config.LastUpdateTime, MobileCompatible: true, Config: *s.config, FileCache: s.FileCache}
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
	Website_domain    string
	SqlDataSourceName string
	WebsiteName       string
	Key               string
	PostKey           string
	ConcurrenceNum    int
	SmtpUsername      string
	SmtpPassword      string
	SmtpAddr          string
	OAuthClientId     string
	OAuthTokenUrl     string
	OAuthAuthUrl      string
	OAuthSecret       string
	Tls_cert_file     string
	Tls_key_file      string
	Log_file          string

	//not read from configuration file
	LastUpdateTime string

	OAUTH2Config           *oauth2.Config
	JWKJsonUrl             string
	OAuthLogoutRedirectUrl string
	OAuthLogoutUrl         string
	IntrospectTokenURL     string
}

func (config *Config) ImageDirPath() (string, error) {
	if config.FileLocation == "" {
		return "", errors.New("FileLocation value is empty")
	}
	path := path.Join(config.FileLocation, "image")
	if err := os.Mkdir(path, os.ModePerm); err != nil && !os.IsExist(err) {
		return "", err
	}
	return path, nil
}

func (config *Config) cachePath(filename string) (string, error) {
	path := path.Join(config.FileLocation, cache_dir, filename)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil && !os.IsExist(err) {
		return "", err
	}
	return path, nil
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

func (hw *HandlerWidget) Pre_Process(ctx *goweb.Context) {
	referer := ctx.Request.Header.Get("Referer")
	if referer == "" {
		return
	}
	u, err := url.Parse(referer)
	if err != nil {
		panic(err)
	}
	links, err := hw.s.MemoryCache.FriendlyLinks(false)
	if err != nil {
		internal.Logger.Println("FriendlyLinks ERROR:", err)
		return
	}
	for _, item := range links {
		if item.Website_url == u.Host {
			hw.s.GetStorage(ctx).FreshFriendlyLinkAccessTime(item.Id)
		}
	}
}
func (hw *HandlerWidget) Post_Process(ctx *goweb.Context) {
	m := ctx.Data["storage"]
	if m != nil {
		m.(storage.Storage).Commit()
	}
	if ctx.Err != nil {
		if ctx.Request.Method == http.MethodPost {
			ctx.Failed(ctx.Err.Error())
		} else {
			data := struct {
				Desc string
			}{Desc: ctx.Err.Error()}
			model := hw.s.NewPageModel(ctx, "ERROR", data)
			ctx.RenderPage(model, "templates/layout.html", "templates/error.html")
		}
	}
}

type FileCache struct {
	HomeWallpaper string
	config        *Config
}

func (cache *FileCache) reload() error {
	path, err := cache.path()
	if err != nil {
		return err
	}
	exist, err := common.CheckIfFileExists(path)
	if err != nil {
		return err
	}
	if !exist {
		return nil
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, cache)
}
func (cache *FileCache) path() (string, error) {
	return cache.config.cachePath("filecache.yaml")
}
func (cache *FileCache) Save() error {
	out, err := yaml.Marshal(cache)
	if err != nil {
		return err
	}
	path, err := cache.path()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, out, os.ModePerm)
	return err
}

type MemoryCache struct {
	friendlyLinks []models.FriendlyLink
	server        *GoBlogServer
}

func (cache *MemoryCache) FriendlyLinks(forceRefresh bool) ([]models.FriendlyLink, error) {
	if cache.friendlyLinks == nil || forceRefresh {
		store := storage.NewSQLManager(cache.server.config.SqlDataSourceName)
		links, err := store.GetFriendlyLinks()
		if err != nil {
			return nil, err
		}
		cache.friendlyLinks = links
		store.Commit()
	}
	return cache.friendlyLinks, nil
}
