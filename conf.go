package main

import (
	"encoding/json"
	"os"

	"golang.org/x/oauth2"
)

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
	OAUTH2Config   *oauth2.Config
	JWKJsonUrl     string
}

var config Config

func ReadConfig(filePath string) Config {
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

	return c
}
