package api

import (
	"StoreServer/models"
	"StoreServer/utils"
	myerror "StoreServer/utils/error"
	gettime "StoreServer/utils/get_time"
	"StoreServer/utils/response"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func CreatePurchase(c *gin.Context) {
	var req models.Purchase

	if err := c.ShouldBind(&req); err != nil {
		response.MyResponse.Error(c, myerror.AnyError(http.StatusBadRequest, err))
		return
	}

	if req.Status == "" {
		req.Status = models.Pending
	}

	if ok := req.Validate(); ok.Code != http.StatusOK {
		c.JSON(ok.Code, ok)
		return
	}

	res := models.PurchaseDB.Create(req)
	c.JSON(res.Code, res)
}

func ApprovePurchase(c *gin.Context) {
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

	purchase := models.PurchaseDB.QueryOne(filter)
	if purchase.Code != http.StatusOK {
		c.JSON(purchase.Code, purchase)
		return
	}

	purchaseData := purchase.Data.(*models.Purchase)
	if purchaseData.Status != models.Pending {
		c.JSON(http.StatusBadRequest, response.Response{
			Code:    http.StatusBadRequest,
			Message: "Only purchases in pending status can be approved",
		})
		return
	}

	// Update purchase status to approved first with conditional match
	statusFilter := bson.M{
		"_id":    objID,
		"status": models.Pending,
	}
	statusUpdate := bson.M{
		"$set": bson.M{
			"status":       models.Approved,
			"updated_time": time.Now(),
		},
	}
	updateStatusRes := models.PurchaseDB.UpdateRaw(statusFilter, statusUpdate)
	if updateStatusRes.Code != http.StatusOK {
		c.JSON(http.StatusBadRequest, response.Response{
			Code:    http.StatusBadRequest,
			Message: "Purchase is no longer in pending status",
		})
		return
	}

	// Atomically increase stock for each product
	for _, prodItem := range purchaseData.Products {
		prodFilter := bson.M{
			"_id": prodItem.ID,
		}
		prodUpdate := bson.M{
			"$inc": bson.M{"stock": prodItem.Quantity},
			"$set": bson.M{
				"updated_time": time.Now(),
			},
		}
		_ = models.ProductDB.UpdateRaw(prodFilter, prodUpdate)
	}


	c.JSON(http.StatusOK, response.Response{
		Code:    http.StatusOK,
		Message: "Purchase approved successfully",
	})
}

func RejectPurchase(c *gin.Context) {
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
		"status":       models.Pending,
		"deleted_time": nil,
	}

	update := bson.M{
		"$set": bson.M{
			"status":       models.Rejected,
			"updated_time": time.Now(),
		},
	}

	res := models.PurchaseDB.UpdateRaw(filter, update)
	if res.Code != http.StatusOK {
		c.JSON(http.StatusBadRequest, response.Response{
			Code:    http.StatusBadRequest,
			Message: "Purchase is not in pending status or does not exist",
		})
		return
	}

	c.JSON(http.StatusOK, response.Response{
		Code:    http.StatusOK,
		Message: "Purchase rejected successfully",
	})
}

func GetPurchase(c *gin.Context) {
	page := utils.ParseInt(c.Query("page"), 1)
	pageSize := utils.ParseInt(c.Query("page_size"), 10)
	timeRange := c.Query("time_range")
	status := c.Query("status")

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
		filter["status"] = models.ConvertPurchaseStatus(status)
	}

	offset := (page - 1) * pageSize
	res := models.PurchaseDB.Query(filter, offset, pageSize)
	if res.Code != http.StatusOK {
		c.JSON(res.Code, res)
		return
	}

	res.Data = res.Data.([]models.Purchase)
	res.Page = page
	res.PageSize = pageSize

	c.JSON(res.Code, res)
}

func GetPurchaseByID(c *gin.Context) {
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

	result := models.PurchaseDB.QueryOne(filter)
	c.JSON(result.Code, result)
}

func UpdatePurchase(c *gin.Context) {
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

	var req models.Purchase
	if err := c.ShouldBind(&req); err != nil {
		response.MyResponse.Error(c, myerror.AnyError(http.StatusBadRequest, err))
		return
	}

	req.ID = objID

	if ok := req.Validate(); ok.Code != http.StatusOK {
		c.JSON(ok.Code, ok)
		return
	}

	filter := bson.M{
		"_id":          objID,
		"status":       models.Pending,
		"deleted_time": nil,
	}

	res := models.PurchaseDB.UpdateOne(filter, req)
	c.JSON(res.Code, res)
}

func DeletePurchase(c *gin.Context) {
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

	res := models.PurchaseDB.DeleteOne(filter)
	c.JSON(res.Code, res)
}

