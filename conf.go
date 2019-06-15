package main

import (
	"encoding/json"
	"os"
	"time"
)

type Config struct {
	FileLocation      string
	Host              string
	SqlDataSourceName string
	WebsiteName       string
	Key               string
	PostKey           string
	LastUpdateTime    time.Time
	ConcurrenceNum    int
}

var config Config

func ReadConfig() Config {
	file, _ := os.Open("conf.json")
	defer file.Close()
	dec := json.NewDecoder(file)
	var c Config
	dec.Decode(&c)

	info, _ := file.Stat()
	loc, _ := time.LoadLocation("Local")
	tm := info.ModTime().In(loc)
	c.LastUpdateTime = tm

	return c
}
