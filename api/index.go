package api

import (
	"github.com/ycjcl868/telegram-image-bot/controller"
	"net/http"

	"github.com/labstack/echo/v4"
)

var srv http.Handler

func init() {
	ctl := controller.Controller{}
	e := echo.New()
	e.GET("/", ctl.Hello)
	e.POST("/api/telegram", ctl.Telegram)
	srv = e
}

func Handler(w http.ResponseWriter, r *http.Request) {
	srv.ServeHTTP(w, r)
}
