package handlers

import (
	"net/http"
	"strconv"
)

const (
	DefaultPageSize = 25
)

// GetLimitParam extracts limit from the request query string (falls back to page_size, defaults to 25).
func GetLimitParam(r *http.Request) int {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = r.URL.Query().Get("page_size")
	}
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l >= 1 {
			return l
		}
	}
	return DefaultPageSize
}
