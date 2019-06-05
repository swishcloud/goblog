package main

import (
	"encoding/json"
	"os"
)
type Config struct{
	FileLocation string
	Host string
	SqlDataSourceName string
	WebsiteName string
	Key string
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
	return Config{FileLocation:v["FileLocation"].(string), Host:v["Host"].(string),SqlDataSourceName:v["SqlDataSourceName"].(string),WebsiteName:v["WebsiteName"].(string),Key:v["Key"].(string)}
}
