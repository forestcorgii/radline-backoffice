package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// PaginationParams holds the raw pagination inputs from the request.
type PaginationParams struct {
	Page     int
	PageSize int
}

// Pagination holds the computed pagination state.
type Pagination struct {
	CurrentPage  int
	PageSize     int
	TotalRecords int
	TotalPages   int
	HasPrev      bool
	HasNext      bool
	PrevPage     int
	NextPage     int
	StartRecord  int
	EndRecord    int
	PageNumbers  []PageNumber
}

// PageNumber represents a single pagination button (number, ellipsis, first, last).
type PageNumber struct {
	Label      string // "1", "2", "...", "«", "»"
	Page       int    // Page number to navigate to (0 for ellipsis)
	IsActive   bool
	IsEllipsis bool
}

// PaginationView is the full view model passed to the template.
type PaginationView struct {
	Pagination
	Path       string   // The base URL path for pagination links
	Target     string   // HTMX target selector (e.g., "#brands-results")
	TargetID   string   // Unique ID prefix for this pagination instance
	QSPreserve []string // Query params to preserve (e.g., ["search", "filter", "sort"])
	IsOOB      bool     // If true, hx-swap-oob="true" is set on the container
}

const (
	DefaultPageSize = 25
	MaxPageSize     = 500
)

var PageSizeOptions = []int{25, 50, 100, 200}

// GetPaginationParams extracts page and pageSize from the request query string.
func GetPaginationParams(r *http.Request) PaginationParams {
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p >= 1 {
			page = p
		}
	}

	pageSizeStr := r.URL.Query().Get("page_size")
	pageSize := DefaultPageSize
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps >= 1 {
			pageSize = ps
			if pageSize > MaxPageSize {
				pageSize = MaxPageSize
			}
		}
	}

	return PaginationParams{Page: page, PageSize: pageSize}
}

// Offset returns the SQL OFFSET value for the current page.
func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// BuildPagination creates a fully computed Pagination from parameters and total count.
func BuildPagination(params PaginationParams, totalRecords int) Pagination {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = DefaultPageSize
	}

	totalPages := int(math.Ceil(float64(totalRecords) / float64(params.PageSize)))
	if totalPages < 1 {
		totalPages = 1
	}
	if params.Page > totalPages {
		params.Page = totalPages
	}

	// Compute start/end record numbers
	startRec := (params.Page-1)*params.PageSize + 1
	if totalRecords == 0 {
		startRec = 0
	}
	endRec := params.Page * params.PageSize
	if endRec > totalRecords {
		endRec = totalRecords
	}

	// Build smart page numbers with ellipsis
	pageNumbers := buildPageNumbers(params.Page, totalPages)

	return Pagination{
		CurrentPage:  params.Page,
		PageSize:     params.PageSize,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		HasPrev:      params.Page > 1,
		HasNext:      params.Page < totalPages,
		PrevPage:     params.Page - 1,
		NextPage:     params.Page + 1,
		StartRecord:  startRec,
		EndRecord:    endRec,
		PageNumbers:  pageNumbers,
	}
}

// buildPageNumbers generates a compact pagination bar with ellipsis.
// Strategy:
//   - Always show first and last page
//   - Show window of pages around current page (max 5)
//   - Insert "..." when there are gaps
func buildPageNumbers(current, total int) []PageNumber {
	if total <= 1 {
		return nil
	}

	var nums []PageNumber

	// Always show first page
	if current > 3 {
		nums = append(nums, PageNumber{Label: "1", Page: 1})
		if current > 4 {
			nums = append(nums, PageNumber{Label: "...", Page: 0, IsEllipsis: true})
		}
	}

	// Window around current page
	start := current - 2
	if start < 1 {
		start = 1
	}
	end := start + 4
	if end > total {
		end = total
		start = end - 4
		if start < 1 {
			start = 1
		}
	}

	for i := start; i <= end; i++ {
		nums = append(nums, PageNumber{
			Label:    strconv.Itoa(i),
			Page:     i,
			IsActive: i == current,
		})
	}

	// Always show last page
	if current < total-2 {
		if current < total-3 {
			nums = append(nums, PageNumber{Label: "...", Page: 0, IsEllipsis: true})
		}
		nums = append(nums, PageNumber{Label: strconv.Itoa(total), Page: total})
	}

	return nums
}

// BuildView creates a PaginationView for templates.
func (p Pagination) BuildView(path, target, targetID string, qsPreserve []string, r *http.Request) PaginationView {
	isOOB := r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") != "main-content"
	return PaginationView{
		Pagination: p,
		Path:       path,
		Target:     target,
		TargetID:   targetID,
		QSPreserve: qsPreserve,
		IsOOB:      isOOB,
	}
}

// PreservedParams returns a comma-separated list of CSS selectors for hx-include.
func (pv PaginationView) PreservedParams() string {
	if len(pv.QSPreserve) == 0 {
		return ""
	}
	selectors := make([]string, len(pv.QSPreserve))
	for i, p := range pv.QSPreserve {
		selectors[i] = fmt.Sprintf("[name='%s']", p)
	}
	return strings.Join(selectors, ",")
}
