package api

import (
	"StoreServer/models"
	"StoreServer/utils"
	myerror "StoreServer/utils/error"
	gettime "StoreServer/utils/get_time"
	"StoreServer/utils/logger"
	"StoreServer/utils/response"
	"fmt"
	"net/http"
	"strings"
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

	// New purchases always start pending; stock is only added through ApprovePurchase.
	req.Status = models.Pending

	if ok := req.Validate(); ok.Code != http.StatusOK {
		c.JSON(ok.Code, ok)
		return
	}

	if !checkRef(c, &models.SupplierDB, "supplier_id", req.SupplierID) || !checkProducts(c, req.Products) {
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

	// A product may have been deleted since the purchase was created.
	if !checkProducts(c, purchaseData.Products) {
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
	var failed []string
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
		if res := models.ProductDB.UpdateRaw(prodFilter, prodUpdate); res.Code != http.StatusOK {
			failed = append(failed, prodItem.ID.Hex())
		}
	}

	if len(failed) > 0 {
		// The purchase is already approved; report it instead of pretending the stock was added.
		logger.GetLogger().Error(fmt.Sprintf("purchase %s approved but stock not added for products %v", objID.Hex(), failed))
		c.JSON(http.StatusInternalServerError, response.Response{
			Code:    http.StatusInternalServerError,
			Message: "Purchase approved but stock could not be added for products: " + strings.Join(failed, ", "),
		})
		return
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
	page, pageSize, offset := utils.ParsePagination(c.Query("page"), c.Query("page_size"))
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

	current := models.PurchaseDB.QueryOne(bson.M{"_id": objID, "deleted_time": nil})
	if current.Code != http.StatusOK {
		c.JSON(current.Code, current)
		return
	}
	if current.Data.(*models.Purchase).Status != models.Pending {
		c.JSON(http.StatusBadRequest, response.Response{
			Code:    http.StatusBadRequest,
			Message: "Only purchases in pending status can be edited",
		})
		return
	}

	if !checkRef(c, &models.SupplierDB, "supplier_id", req.SupplierID) || !checkProducts(c, req.Products) {
		return
	}

	// Keep the status in the filter: the purchase may be approved between the check above and this update.
	filter := bson.M{
		"_id":          objID,
		"status":       models.Pending,
		"deleted_time": nil,
	}

	res := models.PurchaseDB.UpdateOne(filter, req.UpdateDoc())
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

