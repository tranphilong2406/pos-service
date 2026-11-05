package auth

import (
	"StoreServer/models"
	myerror "StoreServer/utils/error"
	"StoreServer/utils/jwt"
	"StoreServer/utils/response"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

func Login(c *gin.Context) {
	var req models.UserLogin
	if err := c.ShouldBind(&req); err != nil {
		response.MyResponse.Error(c, myerror.AnyError(http.StatusBadRequest, err))
		return
	}

	if req.UserName == "" || req.Password == "" {
		response.MyResponse.Error(c, myerror.EmptyParam())
		return
	}

	filter := bson.M{
		"name":         req.UserName,
		"deleted_time": nil,
	}

	// Unknown user and wrong password get the same answer so usernames cannot be enumerated.
	invalidLogin := myerror.AnyError(http.StatusUnauthorized, errors.New("invalid username or password"))

	res := models.AuthDB.QueryOne(filter)
	if res.Code == http.StatusNotFound {
		response.MyResponse.Error(c, invalidLogin)
		return
	}
	if res.Code != http.StatusOK {
		c.JSON(res.Code, res)
		return
	}

	user := res.Data.(*models.User)

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		response.MyResponse.Error(c, invalidLogin)
		return
	}

	token, err := jwt.GenerateToken(user.ID, user.UserName, string(user.Role))
	if err != nil {
		response.MyResponse.Error(c, myerror.AnyError(http.StatusInternalServerError, err))
		return
	}

	response.MyResponse.Success(c, gin.H{
		"token": token,
		"role":  user.Role,
	})
}
