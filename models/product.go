package models

import (
	"StoreServer/job"
	"StoreServer/utils/response"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)


var ProductDB = job.DB{
	ColName:     "product",
	DBName:      "",
	TemplateObj: Product{},
}

type Product struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id" form:"id"`
	CategoryID  string        `bson:"category_id" json:"category_id" form:"category_id"`
	SupplierID  string        `bson:"supplier_id" json:"supplier_id" form:"supplier_id"`
	Name        string        `bson:"name" json:"name" form:"name"`
	Description string        `bson:"description" json:"description" form:"description"`
	Image       string        `bson:"image" json:"image" form:"image" `
	Stock       int           `bson:"stock" json:"stock" form:"stock"`
	CreatedTime *time.Time    `json:"created_time" bson:"created_time"`
	UpdatedTime *time.Time    `json:"updated_time" bson:"updated_time"`
	DeletedTime *time.Time    `json:"deleted_time" bson:"deleted_time,omitempty"`
}

type ProductOrder struct {
	ID       bson.ObjectID `bson:"_id" json:"id"`
	Quantity int           `bson:"quantity" json:"quantity"`
	Buy      int           `bson:"buy" json:"buy"`
	Price    int           `bson:"price" json:"price"`
	Discount int           `bson:"discount" json:"discount"`
}

func (p *Product) Validate() response.Response {
	res := response.Response{
		Code:    http.StatusOK,
		Message: "OK",
		Data:    nil,
	}

	if p.Name == "" {
		res.Code = http.StatusBadRequest
		res.Message = "Name cannot be empty"
		return res
	}
	if p.Stock < 0 {
		res.Code = http.StatusBadRequest
		res.Message = "Stock cannot be negative"
		return res
	}

	return res
}


type ListProduct struct {
	Products []Product `bson:"products" json:"products"`
}

func InitProductDB() {
	ProductDB.Init("store")
	_ = ProductDB.EnsureIndexes([]mongo.IndexModel{
		{Keys: bson.D{{Key: "deleted_time", Value: 1}, {Key: "created_time", Value: -1}}},
		{Keys: bson.D{{Key: "category_id", Value: 1}, {Key: "deleted_time", Value: 1}}},
		{Keys: bson.D{{Key: "supplier_id", Value: 1}, {Key: "deleted_time", Value: 1}}},
		{Keys: bson.D{{Key: "name", Value: 1}}},
	})
}

