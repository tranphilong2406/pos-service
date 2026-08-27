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

func CreateProduct(c *gin.Context) {
	var req models.Product

	if err := c.ShouldBind(&req); err != nil {
		response.MyResponse.Error(c, myerror.AnyError(http.StatusBadRequest, err))
		return
	}

	if ok := req.Validate(); ok.Code != http.StatusOK {
		c.JSON(ok.Code, ok)
		return
	}

	res := models.ProductDB.Create(req)
	c.JSON(res.Code, res)
}

func GetProduct(c *gin.Context) {
	page := utils.ParseInt(c.Query("page"), 1)
	pageSize := utils.ParseInt(c.Query("page_size"), 10)
	categoryID := c.Query("category_id")
	name := c.Query("search")
	supplierID := c.Query("supplier_id")
	filter := bson.M{
		"deleted_time": nil,
	}

	if categoryID != "" {
		filter["category_id"] = categoryID
	}

	if supplierID != "" {
		filter["supplier_id"] = supplierID
	}

	if name != "" {
		filter["name"] = bson.M{"$regex": regexp.QuoteMeta(name), "$options": "i"}
	}

	offset := (page - 1) * pageSize
	res := models.ProductDB.Query(filter, offset, pageSize)
	if res.Code != http.StatusOK {
		c.JSON(res.Code, res)
		return
	}

	res.Data = res.Data.([]models.Product)
	c.JSON(res.Code, res)
}

func UpdateProduct(c *gin.Context) {
	var req models.Product
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
		"description":  req.Description,
		"stock":        req.Stock,
		"updated_time": time.Now(),
	}


	if req.CategoryID != "" {
		updateDoc["category_id"] = req.CategoryID
	}
	if req.SupplierID != "" {
		updateDoc["supplier_id"] = req.SupplierID
	}
	if req.Image != "" {
		updateDoc["image"] = req.Image
	}

	filter := bson.M{
		"_id":          objID,
		"deleted_time": nil,
	}

	updating := models.ProductDB.UpdateRaw(filter, bson.M{"$set": updateDoc})
	c.JSON(updating.Code, updating)
}

func DeleteProduct(c *gin.Context) {
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

	res := models.ProductDB.DeleteOne(filter)
	c.JSON(res.Code, res)
}

