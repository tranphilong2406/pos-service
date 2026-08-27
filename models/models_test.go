package models_test

import (
	"StoreServer/models"
	"net/http"
	"testing"

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
