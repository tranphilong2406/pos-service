package gettime_test

import (
	gettime "StoreServer/utils/get_time"
	"testing"
	"time"
)

func TestRangeFromKeyword(t *testing.T) {
	keywords := []string{"today", "last_7_days", "this_month", "last_month", "year_to_date", "unknown"}

	for _, kw := range keywords {
		start, end := gettime.RangeFromKeyword(kw)
		if start.After(end) {
			t.Errorf("Keyword %s returned start time %v after end time %v", kw, start, end)
		}
		if end.After(time.Now().Add(time.Second)) {
			t.Errorf("Keyword %s returned end time in the future: %v", kw, end)
		}
	}
}
