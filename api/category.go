package api

import (
	"StoreServer/models"
	"StoreServer/utils"
	myerror "StoreServer/utils/error"
	"StoreServer/utils/response"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func CreateCategory(c *gin.Context) {
	var req models.Category

	if err := c.ShouldBind(&req); err != nil {
		response.MyResponse.Error(c, myerror.AnyError(http.StatusBadRequest, err))
		return
	}

	if ok := req.Validate(); ok.Code != http.StatusOK {
		c.JSON(ok.Code, ok)
		return
	}

	res := models.CategoryDB.Create(req)
	c.JSON(res.Code, res)
}

func GetCategory(c *gin.Context) {
	page := utils.ParseInt(c.Query("page"), 1)
	pageSize := utils.ParseInt(c.Query("page_size"), 10)
	name := c.Query("search")
	filter := bson.M{
		"deleted_time": nil,
	}

	if name != "" {
		filter["name"] = bson.M{
			"$regex":   regexp.QuoteMeta(name),
			"$options": "i",
		}
	}

	offset := (page - 1) * pageSize
	res := models.CategoryDB.Query(filter, offset, pageSize)
	c.JSON(res.Code, res)
}

func UpdateCategory(c *gin.Context) {
	var req models.Category

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

	if err := c.ShouldBind(&req); err != nil {
		response.MyResponse.Error(c, myerror.AnyError(http.StatusBadRequest, err))
		return
	}

	if ok := req.Validate(); ok.Code != http.StatusOK {
		response.MyResponse.Error(c, myerror.CustomError{
			ErrorMessage: ok.Message,
			HTTPCode:     ok.Code,
		})
		return
	}

	updateDoc := bson.M{
		"name":         req.Name,
		"description":  req.Description,
		"updated_time": time.Now(),
	}

	filter := bson.M{
		"_id":          objID,
		"deleted_time": nil,
	}

	updating := models.CategoryDB.UpdateRaw(filter, bson.M{"$set": updateDoc})
	c.JSON(updating.Code, updating)
}

func DeleteCategory(c *gin.Context) {
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

	res := models.CategoryDB.DeleteOne(filter)
	c.JSON(res.Code, res)
}

