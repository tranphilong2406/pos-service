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

func CreateSupplier(c *gin.Context) {
	var req models.Supplier

	if err := c.ShouldBind(&req); err != nil {
		response.MyResponse.Error(c, myerror.AnyError(http.StatusBadRequest, err))
		return
	}

	if ok := req.Validate(); ok.Code != http.StatusOK {
		c.JSON(ok.Code, ok)
		return
	}

	res := models.SupplierDB.Create(req)
	c.JSON(res.Code, res)
}

func GetSupplier(c *gin.Context) {
	page := utils.ParseInt(c.Query("page"), 1)
	pageSize := utils.ParseInt(c.Query("page_size"), 10)
	name := c.Query("search")
	filter := bson.M{
		"deleted_time": nil,
	}

	if name != "" {
		filter["name"] = bson.M{"$regex": regexp.QuoteMeta(name), "$options": "i"}
	}
	offset := (page - 1) * pageSize
	res := models.SupplierDB.Query(filter, offset, pageSize)
	if res.Code != http.StatusOK {
		c.JSON(res.Code, res)
		return
	}

	res.Data = res.Data.([]models.Supplier)
	c.JSON(res.Code, res)
}

func UpdateSupplier(c *gin.Context) {
	var req models.Supplier

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
		c.JSON(ok.Code, ok)
		return
	}

	updateDoc := bson.M{
		"name":         req.Name,
		"phone":        req.Phone,
		"email":        req.Email,
		"address":      req.Address,
		"updated_time": time.Now(),
	}

	filter := bson.M{
		"_id":          objID,
		"deleted_time": nil,
	}

	updating := models.SupplierDB.UpdateRaw(filter, bson.M{"$set": updateDoc})
	c.JSON(updating.Code, updating)
}

func DeleteSupplier(c *gin.Context) {
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

	res := models.SupplierDB.DeleteOne(filter)
	c.JSON(res.Code, res)
}

