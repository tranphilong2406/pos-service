package models_test

import (
	"StoreServer/models"
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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

// userWithRole decodes a user the way a request body is bound.
func userWithRole(t *testing.T, role string) models.User {
	t.Helper()
	var u models.User
	if err := json.Unmarshal([]byte(`{"name":"someone","role":"`+role+`"}`), &u); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	return u
}

func TestUserValidationRejectsUnknownRole(t *testing.T) {
	for _, role := range []string{"superadmin", "Admin", "owner"} {
		u := userWithRole(t, role)
		if res := u.Validate(); res.Code != http.StatusBadRequest {
			t.Errorf("Expected role %q to return 400, got %d", role, res.Code)
		}
	}
}

func TestUserValidationAcceptsKnownRoles(t *testing.T) {
	for _, role := range []string{"admin", "manager", "cashier"} {
		u := userWithRole(t, role)
		if res := u.Validate(); res.Code != http.StatusOK {
			t.Errorf("Expected role %q to be valid, got %d: %s", role, res.Code, res.Message)
		}
	}
}

func TestUserValidationDefaultsEmptyRoleToCashier(t *testing.T) {
	u := userWithRole(t, "")
	if res := u.Validate(); res.Code != http.StatusOK || u.Role != models.CashierRole {
		t.Errorf("Expected empty role to default to cashier, got %d role %q", res.Code, u.Role)
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

func TestProductValidateNormalizesReferencePrices(t *testing.T) {
	empty := models.Product{Name: "But"}
	if res := empty.Validate(); res.Code != http.StatusOK {
		t.Fatalf("Expected 200 without reference prices, got %d: %s", res.Code, res.Message)
	}
	if empty.ReferencePrices == nil || len(empty.ReferencePrices) != 0 {
		t.Errorf("Expected a non-nil empty list so Mongo does not store null, got %#v", empty.ReferencePrices)
	}

	spaced := models.Product{Name: "But", ReferencePrices: []models.ReferencePrice{{Unit: "  Lốc 12 cây ", Price: 50000}}}
	if res := spaced.Validate(); res.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", res.Code, res.Message)
	}
	if spaced.ReferencePrices[0].Unit != "Lốc 12 cây" {
		t.Errorf("Expected trimmed unit, got %q", spaced.ReferencePrices[0].Unit)
	}
}

func distinctReferencePrices(n int) []models.ReferencePrice {
	prices := make([]models.ReferencePrice, n)
	for i := range prices {
		prices[i] = models.ReferencePrice{Unit: fmt.Sprintf("Unit %d", i), Price: 1000}
	}
	return prices
}

func TestProductValidateRejectsBadReferencePrices(t *testing.T) {
	cases := map[string][]models.ReferencePrice{
		"blank unit":      {{Unit: "   ", Price: 1000}},
		"unit too long":   {{Unit: strings.Repeat("ố", 51), Price: 1000}},
		"zero price":      {{Unit: "Lốc", Price: 0}},
		"negative price":  {{Unit: "Lốc", Price: -1}},
		"duplicate units": {{Unit: "Lốc", Price: 1}, {Unit: "lốc", Price: 2}},
		"too many rows":   distinctReferencePrices(11),
	}
	for name, prices := range cases {
		t.Run(name, func(t *testing.T) {
			p := models.Product{Name: "But", ReferencePrices: prices}
			if res := p.Validate(); res.Code != http.StatusBadRequest {
				t.Errorf("Expected 400, got %d", res.Code)
			}
		})
	}
}

func TestProductValidateAcceptsReferencePriceLimits(t *testing.T) {
	cases := map[string][]models.ReferencePrice{
		// 50 characters but 150 bytes: the limit counts characters.
		"unit at limit": {{Unit: strings.Repeat("ố", 50), Price: 1000}},
		"rows at limit": distinctReferencePrices(10),
	}
	for name, prices := range cases {
		t.Run(name, func(t *testing.T) {
			p := models.Product{Name: "But", ReferencePrices: prices}
			if res := p.Validate(); res.Code != http.StatusOK {
				t.Errorf("Expected 200, got %d: %s", res.Code, res.Message)
			}
		})
	}
}

func bindProductFromMultipart(t *testing.T, referencePrices ...string) (models.Product, error) {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	if err := form.WriteField("name", "But"); err != nil {
		t.Fatal(err)
	}
	for _, line := range referencePrices {
		if err := form.WriteField("reference_prices", line); err != nil {
			t.Fatal(err)
		}
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", &body)
	c.Request.Header.Set("Content-Type", form.FormDataContentType())

	var p models.Product
	err := c.ShouldBind(&p)
	return p, err
}

// The frontend sends FormData with one JSON field per line, so this is the contract it relies on.
func TestProductBindsReferencePricesFromMultipart(t *testing.T) {
	p, err := bindProductFromMultipart(t, `{"unit":"1 cây","price":5000}`, `{"unit":"Lốc 12 cây","price":50000}`)
	if err != nil {
		t.Fatalf("Expected two lines to bind, got %v", err)
	}
	want := []models.ReferencePrice{{Unit: "1 cây", Price: 5000}, {Unit: "Lốc 12 cây", Price: 50000}}
	if len(p.ReferencePrices) != 2 || p.ReferencePrices[0] != want[0] || p.ReferencePrices[1] != want[1] {
		t.Errorf("Expected %v, got %v", want, p.ReferencePrices)
	}

	p, err = bindProductFromMultipart(t)
	if err != nil || len(p.ReferencePrices) != 0 {
		t.Errorf("Expected no lines and no error, got %v, %v", p.ReferencePrices, err)
	}

	if _, err = bindProductFromMultipart(t, `{"unit":"x","price":"abc"}`); err == nil {
		t.Error("Expected a non-numeric price to fail binding")
	}
}
