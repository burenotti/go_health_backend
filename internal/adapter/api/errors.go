package api

import (
	"fmt"
	"github.com/labstack/echo/v4"
)

type JsonErrorModel struct {
	Message string `json:"message"`
}

func JsonError(c echo.Context, status int, content any) error {
	data := &JsonErrorModel{Message: fmt.Sprintf("%v", content)}
	return c.JSON(status, data)
}
