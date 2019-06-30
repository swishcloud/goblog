package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	FileLocation      string
	Host              string
	SqlDataSourceName string
	WebsiteName       string
	Key               string
	PostKey           string
	ConcurrenceNum    int
	SmtpUsername string
	SmtpPassword string
	SmtpAddr string

	//not read from configuration file
	LastUpdateTime    string
}

var config Config

func ReadConfig() Config {
	file, _ := os.Open("conf.json")
	defer file.Close()
	dec := json.NewDecoder(file)
	var c Config
	dec.Decode(&c)

	info, err := file.Stat()
	if err!=nil{
		panic(err)
	}
	tm := info.ModTime().Local()
	c.LastUpdateTime = tm.Format("2006-01-02 15:04:05")

	return c
}
