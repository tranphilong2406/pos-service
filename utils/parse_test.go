package utils_test

import (
	"StoreServer/utils"
	"testing"
)

func TestParsePagination(t *testing.T) {
	cases := []struct {
		page, pageSize                     string
		wantPage, wantPageSize, wantOffset int
	}{
		{"", "", 1, 10, 0},
		{"3", "20", 3, 20, 40},
		{"0", "10", 1, 10, 0},
		{"-2", "10", 1, 10, 0},
		{"2", "0", 2, 10, 10},
		{"2", "-5", 2, 10, 10},
		{"2", "5000", 2, 100, 100},
		{"abc", "xyz", 1, 10, 0},
	}

	for _, tc := range cases {
		page, pageSize, offset := utils.ParsePagination(tc.page, tc.pageSize)
		if page != tc.wantPage || pageSize != tc.wantPageSize || offset != tc.wantOffset {
			t.Errorf("ParsePagination(%q, %q) = (%d, %d, %d), want (%d, %d, %d)",
				tc.page, tc.pageSize, page, pageSize, offset, tc.wantPage, tc.wantPageSize, tc.wantOffset)
		}
	}
}
