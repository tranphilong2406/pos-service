package api

import (
	"StoreServer/models"
	"StoreServer/utils"
	myerror "StoreServer/utils/error"
	"StoreServer/utils/response"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

func CreateUser(c *gin.Context) {
	var req models.User
	if err := c.ShouldBind(&req); err != nil {
		response.MyResponse.Error(c, myerror.AnyError(http.StatusBadRequest, err))
		return
	}

	if req.Password == "" {
		c.JSON(http.StatusBadRequest, response.Response{
			Code:    http.StatusBadRequest,
			Message: "Password is required",
		})
		return
	}

	if ok := req.Validate(); ok.Code != http.StatusOK {
		c.JSON(ok.Code, ok)
		return
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.MyResponse.Error(c, myerror.AnyError(http.StatusInternalServerError, err))
		return
	}

	req.Password = string(passHash)

	res := models.AuthDB.Create(req)
	// Create echoes the inserted bson.M, which bypasses User.MarshalJSON.
	if doc, ok := res.Data.(bson.M); ok {
		delete(doc, "password")
	}
	c.JSON(res.Code, res)
}

func GetUser(c *gin.Context) {
	_, pageSize, offset := utils.ParsePagination(c.Query("page"), c.Query("page_size"))
	filter := bson.M{
		"deleted_time": nil,
	}

	res := models.AuthDB.Query(filter, offset, pageSize)
	if res.Code != http.StatusOK {
		c.JSON(res.Code, res)
		return
	}

	res.Data = res.Data.([]models.User)
	c.JSON(res.Code, res)
}

func UpdateUser(c *gin.Context) {
	var req models.User

	if err := c.ShouldBind(&req); err != nil {
		response.MyResponse.Error(c, myerror.AnyError(http.StatusBadRequest, err))
		return
	}

	id := c.Param("id")
	if id != "" {
		objID, err := bson.ObjectIDFromHex(id)
		if err != nil {
			response.MyResponse.Error(c, myerror.AnyError(http.StatusBadRequest, err))
			return
		}
		req.ID = objID
	}

	if req.ID.IsZero() {
		response.MyResponse.Error(c, myerror.EmptyParam())
		return
	}

	if ok := req.Validate(); ok.Code != http.StatusOK {
		c.JSON(ok.Code, ok)
		return
	}

	updateDoc := bson.M{
		"name":         req.UserName,
		"full_name":    req.FullName,
		"role":         req.Role,
		"updated_time": time.Now(),
	}

	if req.Password != "" {
		passHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			response.MyResponse.Error(c, myerror.AnyError(http.StatusInternalServerError, err))
			return
		}
		updateDoc["password"] = string(passHash)
	}

	filter := bson.M{
		"_id":          req.ID,
		"deleted_time": nil,
	}

	res := models.AuthDB.UpdateRaw(filter, bson.M{"$set": updateDoc})
	c.JSON(res.Code, res)
}

func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.MyResponse.Error(c, myerror.EmptyParam())
		return
	}

	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		response.MyResponse.Error(c, myerror.AnyError(http.StatusBadRequest, err))
		return
	}

	filter := bson.M{
		"_id":          objID,
		"deleted_time": nil,
	}

	res := models.AuthDB.DeleteOne(filter)
	c.JSON(res.Code, res)
}

