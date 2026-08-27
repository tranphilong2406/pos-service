package models

import (
	"StoreServer/job"
	"StoreServer/utils/response"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var CategoryDB = job.DB{
	ColName:     "category",
	DBName:      "",
	TemplateObj: Category{},
}

type Category struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string        `bson:"name" json:"name"`
	Description string        `bson:"description" json:"description"`
	CreatedTime *time.Time    `json:"created_time" bson:"created_time"`
	UpdatedTime *time.Time    `json:"updated_time" bson:"updated_time"`
	DeletedTime *time.Time    `json:"deleted_time" bson:"deleted_time,omitempty"`
}

func (c Category) Validate() response.Response {
	res := response.Response{
		Code:    http.StatusOK,
		Message: "OK",
		Data:    nil,
	}

	if c.Name == "" {
		res.Code = http.StatusBadRequest
		res.Message = "Name cannot be empty"
		return res
	}

	return res
}

func InitCategoryDB() {
	CategoryDB.Init("store")
	_ = CategoryDB.EnsureIndexes([]mongo.IndexModel{
		{Keys: bson.D{{Key: "deleted_time", Value: 1}, {Key: "created_time", Value: -1}}},
		{Keys: bson.D{{Key: "name", Value: 1}}},
	})
}

