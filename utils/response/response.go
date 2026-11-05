package response

import (
	myerr "StoreServer/utils/error"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Response struct {
	Message  string      `json:"message"`
	Data     interface{} `json:"data"`
	Code     int         `json:"code"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Total    int         `json:"total"`
}

var MyResponse Response

func Respond(c *gin.Context, response *Response) {
	c.JSON(response.Code, response)
}

// Success, Created and Error all write Response so every endpoint returns the same JSON shape.
func (Response) Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: http.StatusOK, Message: "ok", Data: data})
}

func (Response) Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{Code: http.StatusCreated, Message: "ok", Data: data})
}

func (Response) Error(c *gin.Context, err myerr.CustomError) {
	c.JSON(err.HTTPCode, Response{Code: err.HTTPCode, Message: err.Error()})
}
