// Command backfillprices gives products that still hold a legacy sell_price a reference
// price. Without -apply it only counts them and writes nothing.
//
//	go run ./cmd/backfillprices
//	go run ./cmd/backfillprices -apply
//
// Run -apply only after the frontend that sends reference_prices is deployed: the old one
// does not send them, so saving a product there clears the list.
package main

import (
	"StoreServer/job"
	"StoreServer/models"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// legacyUnit labels the backfilled price: the old field did not say which unit it was for.
const legacyUnit = "Giá bán"

// sampleSize is how many products a dry run prints so the data can be checked by eye.
const sampleSize = 5

type legacyProduct struct {
	ID        bson.ObjectID `bson:"_id"`
	Name      string        `bson:"name"`
	SellPrice int           `bson:"sell_price"`
}

type productStore interface {
	// candidates returns the products to backfill and how many documents could not be decoded.
	candidates(ctx context.Context) ([]legacyProduct, int, error)
	setPrices(ctx context.Context, id bson.ObjectID, prices []models.ReferencePrice) error
}

func main() {
	os.Exit(run())
}

func run() int {
	apply := flag.Bool("apply", false, "write the reference prices instead of only counting")
	flag.Parse()

	job.DBConnect()
	defer job.Disconnect()
	models.InitProductDB()

	if err := backfill(context.Background(), mongoStore{}, *apply, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

// candidateFilter matches live products with a legacy price and no reference prices, which
// makes running the command again safe. {field: nil} also matches a missing field.
func candidateFilter() bson.M {
	return bson.M{
		"deleted_time": nil,
		"sell_price":   bson.M{"$gt": 0},
		"$or": []bson.M{
			{"reference_prices": nil},
			{"reference_prices": bson.M{"$size": 0}},
		},
	}
}

func backfill(ctx context.Context, s productStore, apply bool, out io.Writer) error {
	products, skipped, err := s.candidates(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Products to backfill: %d\n", len(products))
	if skipped > 0 {
		fmt.Fprintf(out, "Documents skipped because they could not be read: %d\n", skipped)
	}

	if !apply {
		for i, p := range products {
			if i == sampleSize {
				break
			}
			fmt.Fprintf(out, "  %s: %d\n", p.Name, p.SellPrice)
		}
		fmt.Fprintln(out, "Nothing was written. Run again with -apply to write.")
		return nil
	}

	var failures []error
	written := 0
	for _, p := range products {
		prices := []models.ReferencePrice{{Unit: legacyUnit, Price: p.SellPrice}}
		if err := s.setPrices(ctx, p.ID, prices); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", p.ID.Hex(), err))
			continue
		}
		written++
	}
	fmt.Fprintf(out, "Products written: %d\n", written)
	return errors.Join(failures...)
}

type mongoStore struct{}

func (mongoStore) candidates(ctx context.Context) ([]legacyProduct, int, error) {
	projection := options.Find().SetProjection(bson.M{"name": 1, "sell_price": 1})
	cursor, err := models.ProductDB.GetCollection().Find(ctx, candidateFilter(), projection)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var products []legacyProduct
	skipped := 0
	for cursor.Next(ctx) {
		var p legacyProduct
		if err := cursor.Decode(&p); err != nil {
			skipped++
			continue
		}
		products = append(products, p)
	}
	return products, skipped, cursor.Err()
}

func (mongoStore) setPrices(_ context.Context, id bson.ObjectID, prices []models.ReferencePrice) error {
	filter := candidateFilter()
	filter["_id"] = id
	res := models.ProductDB.UpdateRaw(filter, bson.M{"$set": bson.M{
		"reference_prices": prices,
		"updated_time":     time.Now(),
	}})
	if res.Code != http.StatusOK {
		return errors.New(res.Message)
	}
	return nil
}
