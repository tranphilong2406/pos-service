package models_test

import (
	"StoreServer/models"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestProductValidation(t *testing.T) {
	validProd := models.Product{
		Name:  "Notebook",
		Stock: 10,
	}
	if res := validProd.Validate(); res.Code != http.StatusOK {
		t.Errorf("Expected valid product to return 200, got %d: %s", res.Code, res.Message)
	}

	invalidProd := models.Product{
		Name:  "",
		Stock: -5,
	}
	if res := invalidProd.Validate(); res.Code != http.StatusBadRequest {
		t.Errorf("Expected invalid product to return 400, got %d", res.Code)
	}
}


func TestOrderValidation(t *testing.T) {
	validOrder := models.Order{
		Products: []models.ProductOrder{
			{ID: bson.NewObjectID(), Quantity: 2, Price: 10000},
		},
		TotalPrice: 20000,
	}
	if res := validOrder.Validate(); res.Code != http.StatusOK {
		t.Errorf("Expected valid order to return 200, got %d: %s", res.Code, res.Message)
	}

	emptyOrder := models.Order{
		Products:   []models.ProductOrder{},
		TotalPrice: 0,
	}
	if res := emptyOrder.Validate(); res.Code != http.StatusBadRequest {
		t.Errorf("Expected empty order to return 400, got %d", res.Code)
	}
}

func TestPurchaseValidation(t *testing.T) {
	validPurchase := models.Purchase{
		SupplierID: "sup123",
		Products: []models.ProductOrder{
			{ID: bson.NewObjectID(), Quantity: 10, Buy: 4000},
		},
		TotalPrice: 40000,
	}
	if res := validPurchase.Validate(); res.Code != http.StatusOK {
		t.Errorf("Expected valid purchase to return 200, got %d: %s", res.Code, res.Message)
	}
}

func TestPurchaseValidationRejectsNonPositiveQuantity(t *testing.T) {
	for _, qty := range []int{0, -3} {
		p := models.Purchase{
			SupplierID: "sup123",
			Products: []models.ProductOrder{
				{ID: bson.NewObjectID(), Quantity: qty, Buy: 4000},
			},
			TotalPrice: 40000,
		}
		if res := p.Validate(); res.Code != http.StatusBadRequest {
			t.Errorf("Expected quantity %d to return 400, got %d", qty, res.Code)
		}
	}
}

func TestPurchaseValidationRejectsMissingProductID(t *testing.T) {
	p := models.Purchase{
		SupplierID: "sup123",
		Products: []models.ProductOrder{
			{Quantity: 10, Buy: 4000},
		},
		TotalPrice: 40000,
	}
	if res := p.Validate(); res.Code != http.StatusBadRequest {
		t.Errorf("Expected missing product ID to return 400, got %d", res.Code)
	}
}

func TestPurchaseUpdateDocOnlyHasEditableFields(t *testing.T) {
	now := time.Now()
	p := models.Purchase{
		ID:         bson.NewObjectID(),
		SupplierID: "sup123",
		Products: []models.ProductOrder{
			{ID: bson.NewObjectID(), Quantity: 10, Buy: 4000},
		},
		TotalPrice:  40000,
		Status:      models.Approved,
		CreatedTime: &now,
	}

	doc := p.UpdateDoc()

	for _, key := range []string{"_id", "status", "created_time", "deleted_time"} {
		if _, ok := doc[key]; ok {
			t.Errorf("UpdateDoc must not set %q", key)
		}
	}
	if doc["supplier_id"] != "sup123" {
		t.Errorf("Expected supplier_id sup123, got %v", doc["supplier_id"])
	}
	if doc["total_price"] != 40000.0 {
		t.Errorf("Expected total_price 40000, got %v", doc["total_price"])
	}
	if _, ok := doc["products"]; !ok {
		t.Error("Expected products in UpdateDoc")
	}
}

func TestUserJSONOmitsPassword(t *testing.T) {
	const hash = "$2a$10$abcdefghijklmnopqrstuv"
	users := []models.User{{UserName: "admin", Password: hash}}

	b, err := json.Marshal(users)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if strings.Contains(string(b), "password") || strings.Contains(string(b), hash) {
		t.Errorf("Password hash leaked in JSON: %s", b)
	}
	if !strings.Contains(string(b), `"name":"admin"`) {
		t.Errorf("Expected other fields to be kept, got %s", b)
	}
}

func TestUserJSONStillAcceptsPassword(t *testing.T) {
	var u models.User
	if err := json.Unmarshal([]byte(`{"name":"admin","password":"secret"}`), &u); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if u.Password != "secret" {
		t.Errorf("Expected password to be read from request body, got %q", u.Password)
	}
}

func TestUserValidation(t *testing.T) {
	validUser := models.User{
		UserName: "admin",
	}
	if res := validUser.Validate(); res.Code != http.StatusOK {
		t.Errorf("Expected valid user to return 200, got %d: %s", res.Code, res.Message)
	}

	emptyUser := models.User{
		UserName: "",
	}
	if res := emptyUser.Validate(); res.Code != http.StatusBadRequest {
		t.Errorf("Expected empty username to return 400, got %d", res.Code)
	}
}
