package models

import (
	"StoreServer/job"
	"StoreServer/utils/response"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	MaxReferencePrices  = 10
	MaxReferenceUnitLen = 50
)

var ProductDB = job.DB{
	ColName:     "product",
	DBName:      "",
	TemplateObj: Product{},
}

// ReferencePrice is a suggested selling price for one unit, such as a single pen or a
// pack of 12. It only guides staff: the price of a sale is still set on the order.
type ReferencePrice struct {
	Unit  string `bson:"unit" json:"unit"`
	Price int    `bson:"price" json:"price"`
}

type Product struct {
	ID              bson.ObjectID    `bson:"_id,omitempty" json:"id" form:"id"`
	CategoryID      string           `bson:"category_id" json:"category_id" form:"category_id"`
	SupplierID      string           `bson:"supplier_id" json:"supplier_id" form:"supplier_id"`
	Name            string           `bson:"name" json:"name" form:"name"`
	Description     string           `bson:"description" json:"description" form:"description"`
	Image           string           `bson:"image" json:"image" form:"image" `
	Stock           int              `bson:"stock" json:"stock" form:"stock"`
	ReferencePrices []ReferencePrice `bson:"reference_prices" json:"reference_prices" form:"reference_prices"`
	CreatedTime     *time.Time       `json:"created_time" bson:"created_time"`
	UpdatedTime     *time.Time       `json:"updated_time" bson:"updated_time"`
	DeletedTime     *time.Time       `json:"deleted_time" bson:"deleted_time,omitempty"`
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
	if message := p.normalizeReferencePrices(); message != "" {
		res.Code = http.StatusBadRequest
		res.Message = message
		return res
	}

	return res
}

// normalizeReferencePrices trims units and turns nil into an empty list so Mongo does not
// store null. It returns what is wrong with the list, or an empty string when it is valid.
func (p *Product) normalizeReferencePrices() string {
	if p.ReferencePrices == nil {
		p.ReferencePrices = []ReferencePrice{}
	}
	if len(p.ReferencePrices) > MaxReferencePrices {
		return fmt.Sprintf("At most %d reference prices are allowed", MaxReferencePrices)
	}

	seen := make(map[string]struct{}, len(p.ReferencePrices))
	for i := range p.ReferencePrices {
		line := &p.ReferencePrices[i]
		line.Unit = strings.TrimSpace(line.Unit)
		if line.Unit == "" {
			return "Reference price unit cannot be empty"
		}
		if utf8.RuneCountInString(line.Unit) > MaxReferenceUnitLen {
			return fmt.Sprintf("Reference price unit cannot exceed %d characters", MaxReferenceUnitLen)
		}
		if line.Price <= 0 {
			return "Reference price must be greater than 0"
		}
		key := strings.ToLower(line.Unit)
		if _, duplicate := seen[key]; duplicate {
			return "Reference price units must be unique"
		}
		seen[key] = struct{}{}
	}
	return ""
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

