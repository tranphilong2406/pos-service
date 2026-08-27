package models

import (
	"StoreServer/job"
	"StoreServer/utils/response"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type roleType string

const (
	AdminRole   roleType = "admin"
	CashierRole roleType = "cashier"
	ManagerRole roleType = "manager"
)

func ConvertRoleType(role string) roleType {
	switch role {
	case "admin":
		return AdminRole
	case "cashier":
		return CashierRole
	case "manager":
		return ManagerRole
	default:
		return CashierRole
	}
}

var AuthDB = job.DB{
	ColName:     "auth",
	DBName:      "",
	TemplateObj: User{},
}

type User struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserName    string        `bson:"name" json:"name" form:"name" binding:"required"`
	Password    string        `bson:"password" json:"password" form:"password"`
	FullName    string        `bson:"full_name" json:"full_name" form:"full_name"`
	Role        roleType      `bson:"role" json:"role" form:"role"`
	CreatedTime *time.Time    `json:"created_time" bson:"created_time"`
	UpdatedTime *time.Time    `json:"updated_time" bson:"updated_time"`
	DeletedTime *time.Time    `json:"deleted_time" bson:"deleted_time,omitempty"`
}

func (u *User) Validate() response.Response {
	res := response.Response{
		Code:    http.StatusOK,
		Message: "OK",
		Data:    nil,
	}

	if u.UserName == "" {
		res.Code = http.StatusBadRequest
		res.Message = "User name is required"
		return res
	}
	if u.Role == "" {
		u.Role = CashierRole
	}

	return res
}

type UserLogin struct {
	UserName string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func InitAuthDB() {
	AuthDB.Init("store")
	_ = AuthDB.EnsureIndexes([]mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "deleted_time", Value: 1}},
		},
	})
}

