package main

import (
	"encoding/json"
	"os"
	"time"
)
type Config struct{
	FileLocation string
	Host string
	SqlDataSourceName string
	WebsiteName string
	Key string
	LastUpdateTime time.Time
	ConcurrenceNum int
}
var config Config
func ReadConfig()Config {
	file, _ := os.Open("conf.json")
	defer file.Close()
	dec := json.NewDecoder(file)
	var v  map[string]interface{}
	var c  Config
	dec.Decode(&v)
	dec.Decode(&c)
	info,_:=file.Stat()
	loc, _ := time.LoadLocation("Local")
	tm:=info.ModTime().In(loc)
	return Config{
		FileLocation:v["FileLocation"].(string),
		Host:v["Host"].(string),
		SqlDataSourceName:v["SqlDataSourceName"].(string),
		WebsiteName:v["WebsiteName"].(string),
		Key:v["Key"].(string),
		LastUpdateTime:tm,
		ConcurrenceNum:int(v["ConcurrenceNum"].(float64)),
	}
}
