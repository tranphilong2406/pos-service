package main

import (
	"StoreServer/models"
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeStore struct {
	products []legacyProduct
	skipped  int
	failOn   bson.ObjectID
	written  map[bson.ObjectID][]models.ReferencePrice
}

func (s *fakeStore) candidates(context.Context) ([]legacyProduct, int, error) {
	return s.products, s.skipped, nil
}

func (s *fakeStore) setPrices(_ context.Context, id bson.ObjectID, prices []models.ReferencePrice) error {
	if id == s.failOn {
		return errors.New("write failed")
	}
	if s.written == nil {
		s.written = map[bson.ObjectID][]models.ReferencePrice{}
	}
	s.written[id] = prices
	return nil
}

func twoProducts() []legacyProduct {
	return []legacyProduct{
		{ID: bson.NewObjectID(), Name: "But bi", SellPrice: 5000},
		{ID: bson.NewObjectID(), Name: "Vo", SellPrice: 12000},
	}
}

func TestBackfillDryRunWritesNothing(t *testing.T) {
	store := &fakeStore{products: twoProducts()}
	var out bytes.Buffer

	if err := backfill(context.Background(), store, false, &out); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(store.written) != 0 {
		t.Errorf("Expected a dry run to write nothing, wrote %v", store.written)
	}
	if !strings.Contains(out.String(), "2") || !strings.Contains(out.String(), "-apply") {
		t.Errorf("Expected the count and how to apply, got %q", out.String())
	}
}

func TestBackfillApplyWritesReferencePrice(t *testing.T) {
	products := twoProducts()
	store := &fakeStore{products: products}

	if err := backfill(context.Background(), store, true, &bytes.Buffer{}); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	want := []models.ReferencePrice{{Unit: "Giá bán", Price: 5000}}
	if !reflect.DeepEqual(store.written[products[0].ID], want) {
		t.Errorf("Expected %v, got %v", want, store.written[products[0].ID])
	}
	if len(store.written) != 2 {
		t.Errorf("Expected both products to be written, got %d", len(store.written))
	}
}

func TestBackfillContinuesAfterWriteFailure(t *testing.T) {
	products := twoProducts()
	store := &fakeStore{products: products, failOn: products[0].ID}

	if err := backfill(context.Background(), store, true, &bytes.Buffer{}); err == nil {
		t.Error("Expected an error so the command exits non-zero")
	}
	if _, ok := store.written[products[1].ID]; !ok {
		t.Error("Expected the second product to be written after the first failed")
	}
}

func TestBackfillReportsUndecodableDocuments(t *testing.T) {
	store := &fakeStore{products: nil, skipped: 7}
	var out bytes.Buffer

	if err := backfill(context.Background(), store, false, &out); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(out.String(), "7") {
		t.Errorf("Expected the skipped count in the output, got %q", out.String())
	}
}

func TestCandidateFilter(t *testing.T) {
	filter := candidateFilter()

	if value, ok := filter["deleted_time"]; !ok || value != nil {
		t.Errorf("Expected deleted products to be excluded, got %v", filter["deleted_time"])
	}
	if !reflect.DeepEqual(filter["sell_price"], bson.M{"$gt": 0}) {
		t.Errorf("Expected sell_price > 0, got %v", filter["sell_price"])
	}
	wantOr := []bson.M{
		{"reference_prices": nil},
		{"reference_prices": bson.M{"$size": 0}},
	}
	if !reflect.DeepEqual(filter["$or"], wantOr) {
		t.Errorf("Expected products without reference prices, got %v", filter["$or"])
	}
}
