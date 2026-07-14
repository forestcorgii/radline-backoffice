package handlers

import (
	"math"
	"net/http"
	"strconv"
)

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
	Pages        []int
}

type PaginationView struct {
	Pagination
	Path     string
	Target   string
	TargetID string
	Include  string
}

func NewPagination(page, pageSize, totalRecords int) Pagination {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}
	totalPages := int(math.Ceil(float64(totalRecords) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	pages := []int{}
	// Show up to 5 pages around the current page
	start := page - 2
	if start < 1 {
		start = 1
	}
	end := start + 4
	if end > totalPages {
		end = totalPages
		start = end - 4
		if start < 1 {
			start = 1
		}
	}
	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}

	startRec := (page-1)*pageSize + 1
	if totalRecords == 0 {
		startRec = 0
	}
	endRec := page * pageSize
	if endRec > totalRecords {
		endRec = totalRecords
	}

	return Pagination{
		CurrentPage:  page,
		PageSize:     pageSize,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		HasPrev:      page > 1,
		HasNext:      page < totalPages,
		PrevPage:     page - 1,
		NextPage:     page + 1,
		StartRecord:  startRec,
		EndRecord:    endRec,
		Pages:        pages,
	}
}

func GetPageParam(r *http.Request) int {
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		return 1
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 1
	}
	return page
}
