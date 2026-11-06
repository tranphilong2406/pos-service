package api

import (
	"StoreServer/models"
	"reflect"
	"testing"
)

// A PUT replaces the whole list, so an empty list must reach the database to clear it.
func TestProductUpdateDocAlwaysSetsReferencePrices(t *testing.T) {
	doc := productUpdateDoc(models.Product{Name: "But", ReferencePrices: []models.ReferencePrice{}})
	got, ok := doc["reference_prices"].([]models.ReferencePrice)
	if !ok || got == nil || len(got) != 0 {
		t.Errorf("Expected a non-nil empty list, got %#v", doc["reference_prices"])
	}

	want := []models.ReferencePrice{{Unit: "Lốc", Price: 50000}}
	doc = productUpdateDoc(models.Product{Name: "But", ReferencePrices: want})
	if !reflect.DeepEqual(doc["reference_prices"], want) {
		t.Errorf("Expected %v, got %v", want, doc["reference_prices"])
	}
}

func TestProductUpdateDocKeepsOptionalFieldRules(t *testing.T) {
	doc := productUpdateDoc(models.Product{Name: "But", Description: "Xanh", Stock: 3})
	for _, key := range []string{"category_id", "supplier_id", "image"} {
		if _, ok := doc[key]; ok {
			t.Errorf("Expected empty %s to be left out so the stored value stays", key)
		}
	}
	if doc["name"] != "But" || doc["description"] != "Xanh" || doc["stock"] != 3 {
		t.Errorf("Expected name, description and stock to be set, got %v", doc)
	}
	if _, ok := doc["updated_time"]; !ok {
		t.Error("Expected updated_time to be set")
	}

	doc = productUpdateDoc(models.Product{Name: "But", CategoryID: "c1", SupplierID: "s1", Image: "https://x/y.jpg"})
	if doc["category_id"] != "c1" || doc["supplier_id"] != "s1" || doc["image"] != "https://x/y.jpg" {
		t.Errorf("Expected set optional fields to be kept, got %v", doc)
	}
}
