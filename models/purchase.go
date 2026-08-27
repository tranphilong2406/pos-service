package models

import (
	"StoreServer/job"
	"StoreServer/utils/response"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type purchaseStatus string

const (
	Pending  purchaseStatus = "pending"
	Approved purchaseStatus = "approved"
	Rejected purchaseStatus = "rejected"
)

func ConvertPurchaseStatus(status string) purchaseStatus {
	switch status {
	case "pending":
		return Pending
	case "approved":
		return Approved
	case "rejected":
		return Rejected
	default:
		return Pending
	}
}

var PurchaseDB = job.DB{
	ColName:     "purchase",
	DBName:      "",
	TemplateObj: Purchase{},
}

type Purchase struct {
	ID          bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	SupplierID  string         `bson:"supplier_id" json:"supplier_id"`
	Products    []ProductOrder `bson:"products" json:"products"`
	TotalPrice  float64        `bson:"total_price" json:"total_price"`
	Status      purchaseStatus `bson:"status" json:"status"` // e.g., "pending", "approved", "rejected"
	CreatedTime *time.Time     `bson:"created_time" json:"created_time"`
	UpdatedTime *time.Time     `bson:"updated_time" json:"updated_time"`
	DeletedTime *time.Time     `bson:"deleted_time" json:"deleted_time,omitempty"`
}

func (r Purchase) Validate() response.Response {
	res := response.Response{
		Code:    http.StatusOK,
		Message: "OK",
		Data:    nil,
	}

	if len(r.Products) < 1 {
		res.Code = http.StatusBadRequest
		res.Message = "At least 1 product in order"
		return res
	}
	if r.TotalPrice <= 0 {
		res.Code = http.StatusBadRequest
		res.Message = "Price must be greater than 0"
		return res
	}

	return res
}

func InitPurchaseDB() {
	PurchaseDB.Init("store")
	_ = PurchaseDB.EnsureIndexes([]mongo.IndexModel{
		{Keys: bson.D{{Key: "deleted_time", Value: 1}, {Key: "created_time", Value: -1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "created_time", Value: -1}}},
		{Keys: bson.D{{Key: "supplier_id", Value: 1}, {Key: "deleted_time", Value: 1}}},
	})
}

