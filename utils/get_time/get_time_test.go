package gettime_test

import (
	gettime "StoreServer/utils/get_time"
	"testing"
	"time"
)

func TestLastMonthEndsRightBeforeThisMonth(t *testing.T) {
	now := time.Now()
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	_, end := gettime.RangeFromKeyword("last_month")
	if !end.Add(time.Nanosecond).Equal(firstOfThisMonth) {
		t.Errorf("Expected last_month to end 1ns before %v, got %v", firstOfThisMonth, end)
	}
}

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
