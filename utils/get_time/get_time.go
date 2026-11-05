package gettime

import "time"

func RangeFromKeyword(keyword string) (time.Time, time.Time) {
	now := time.Now()
	loc := now.Location()

	switch keyword {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		return start, now
	case "last_7_days":
		return now.AddDate(0, 0, -7), now
	case "this_month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		return start, now
	case "last_month":
		firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		lastMonth := firstOfThisMonth.AddDate(0, -1, 0)
		start := time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, loc)
		// MongoDB stores milliseconds, so ending 1ns early still covers the whole last second with $lte.
		end := firstOfThisMonth.Add(-time.Nanosecond)
		return start, end
	case "year_to_date":
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc)
		return start, now
	default:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		return start, now
	}
}
