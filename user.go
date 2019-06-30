package main

import (
	"encoding/json"
	"github.com/github-123456/gostudy/aesencryption"
	"github.com/github-123456/goweb"
	"net/http"
)

type User struct {
	Id       int
	UserName string
}

func MustGetLoginUser(c *goweb.Context) User {
	u, err := GetLoginUser(c)
	if err != nil {
		panic(err)
	}
	return u
}

func GetLoginUser(c *goweb.Context) (User, error) {
	cookie, err := c.Request.Cookie(SessionName)
	if err != nil {
		return User{}, err
	}
	plain, err := aesencryption.Decrypt([]byte(config.Key), cookie.Value)
	if err != nil {
		return User{}, err
	}
	var user User
	json.Unmarshal([]byte(plain), &user)
	return user, nil
}

func IsLogin(c *goweb.Context) bool {
	_, err := GetLoginUser(c)
	return err == nil
}

func AuthMiddleware() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		if !IsLogin(ctx) {
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
