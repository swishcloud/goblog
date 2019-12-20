package main

import (
	"net/http"

	"github.com/swishcloud/goweb"
	"github.com/swishcloud/goweb/auth"
)

func AuthMiddleware() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		if !auth.HasLoggedIn(ctx) {
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
