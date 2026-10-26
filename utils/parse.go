package utils

import (
	"strconv"
	"time"
)

func ParseInt(s string, def int) int {
	if s == "" {
		return def
	}
	num, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return num
}

const (
	DefaultPageSize = 10
	MaxPageSize     = 100
)

// ParsePagination clamps page to >= 1 and pageSize to [1, MaxPageSize] so the skip
// sent to MongoDB is never negative and a single request cannot dump a whole collection.
func ParsePagination(pageStr, pageSizeStr string) (page, pageSize, offset int) {
	page = ParseInt(pageStr, 1)
	if page < 1 {
		page = 1
	}

	pageSize = ParseInt(pageSizeStr, DefaultPageSize)
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	return page, pageSize, (page - 1) * pageSize
}

func ParseTime(s string) time.Time {
	if s == "" {
		return time.Now()
	}
	// Assuming the time is in a specific format, you can add parsing logic here if needed.
	timeParse, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Now()
	}
	return timeParse
}

func ParseBool(s string) bool {
	if s == "" {
		return true
	}
	
	b, err := strconv.ParseBool(s)
	if err != nil {
		return true
	}
	return b
}
