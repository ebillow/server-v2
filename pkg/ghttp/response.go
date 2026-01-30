package ghttp

import (
	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	CodeRaw string      `json:"codeRaw,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(200, &Response{
		Code: 0,
		Data: data,
	})
}

func Fail(c *gin.Context, statusCode int, code int, msg string) {
	c.JSON(statusCode, &Response{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
}
