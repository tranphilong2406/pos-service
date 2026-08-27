package api

import (
	"StoreServer/models"
	"StoreServer/utils"
	myerror "StoreServer/utils/error"
	gettime "StoreServer/utils/get_time"
	"StoreServer/utils/response"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func CreateOrder(c *gin.Context) {
	var req models.Order
	if err := c.ShouldBind(&req); err != nil {
		response.MyResponse.Error(c, myerror.AnyError(http.StatusBadRequest, err))
		return
	}

	if ok := req.Validate(); ok.Code != http.StatusOK {
		c.JSON(ok.Code, ok)
		return
	}

	decrementedIndices := make([]int, 0, len(req.Products))

	// Atomically deduct stock with pre-condition stock >= quantity
	for idx, prodItem := range req.Products {
		if prodItem.Quantity <= 0 {
			rollbackStock(req.Products, decrementedIndices)
			c.JSON(http.StatusBadRequest, response.Response{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Invalid quantity for product ID %s", prodItem.ID.Hex()),
			})
			return
		}

		filter := bson.M{
			"_id":          prodItem.ID,
			"deleted_time": nil,
			"stock":        bson.M{"$gte": prodItem.Quantity},
		}
		update := bson.M{
			"$inc": bson.M{"stock": -prodItem.Quantity},
			"$set": bson.M{"updated_time": time.Now()},
		}

		updateRes := models.ProductDB.UpdateRaw(filter, update)
		if updateRes.Code != http.StatusOK {
			// Insufficient stock or product not found -> Roll back already decremented items
			rollbackStock(req.Products, decrementedIndices)
			c.JSON(http.StatusBadRequest, response.Response{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Insufficient stock or product not found for ID: %s", prodItem.ID.Hex()),
			})
			return
		}

		decrementedIndices = append(decrementedIndices, idx)
	}

	res := models.OrderDB.Create(req)
	if res.Code != http.StatusCreated && res.Code != http.StatusOK {
		rollbackStock(req.Products, decrementedIndices)
		c.JSON(res.Code, res)
		return
	}

	c.JSON(res.Code, res)
}

func rollbackStock(products []models.ProductOrder, indices []int) {
	for _, idx := range indices {
		prod := products[idx]
		filter := bson.M{"_id": prod.ID}
		update := bson.M{
			"$inc": bson.M{"stock": prod.Quantity},
			"$set": bson.M{"updated_time": time.Now()},
		}
		_ = models.ProductDB.UpdateRaw(filter, update)
	}
}

func GetOrder(c *gin.Context) {
	page := utils.ParseInt(c.Query("page"), 1)
	pageSize := utils.ParseInt(c.Query("page_size"), 10)
	status := c.Query("status")
	timeRange := c.Query("time_range")

	filter := bson.M{
		"deleted_time": nil,
	}

	if timeRange != "" {
		timeStart, timeEnd := gettime.RangeFromKeyword(timeRange)
		filter["created_time"] = bson.M{
			"$gte": timeStart,
			"$lte": timeEnd,
		}
	}

	if status != "" && status != "all" {
		filter["status"] = utils.ParseBool(status)
	}

	offset := (page - 1) * pageSize
	res := models.OrderDB.Query(filter, offset, pageSize)
	if res.Code != http.StatusOK {
		c.JSON(res.Code, res)
		return
	}

	res.Data = res.Data.([]models.Order)
	res.Page = page
	res.PageSize = pageSize

	c.JSON(res.Code, res)
}

func UpdateStatusOrder(c *gin.Context) {
	var req models.UpdateOrder
	if err := c.ShouldBindJSON(&req); err != nil {
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

	filter := bson.M{"_id": req.ID, "deleted_time": nil}
	update := bson.M{"status": req.Status}

	res := models.OrderDB.UpdateOne(filter, update)
	c.JSON(res.Code, res)
}

func GetOrderByID(c *gin.Context) {
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

	result := models.OrderDB.QueryOne(filter)
	c.JSON(result.Code, result)
}

func DeleteOrder(c *gin.Context) {
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

	res := models.OrderDB.DeleteOne(filter)
	c.JSON(res.Code, res)
}

