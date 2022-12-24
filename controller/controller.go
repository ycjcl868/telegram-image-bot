package controller

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type Controller struct {
}

func (controller *Controller) Hello(c echo.Context) error {
	return c.HTML(http.StatusOK, "<h1>Hello from Go!</h1>")
}
