package api

import "math"

// Pagination describes page-based list metadata.
type Pagination struct {
	Page         int  `json:"page"`
	Limit        int  `json:"limit"`
	Total        int  `json:"total"`
	TotalPages   int  `json:"total_pages"`
	NextPage     *int `json:"next_page,omitempty"`
	PreviousPage *int `json:"previous_page,omitempty"`
}

// normalizePaging resolves page/offset and clamps limit to configured bounds.
func normalizePaging(filter Filter, defaultLimit, maxLimit int) (limit, offset, page int) {
	limit = filter.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	page = filter.Page
	if page > 0 {
		offset = (page - 1) * limit
	} else {
		offset = filter.Offset
		if offset < 0 {
			offset = 0
		}
		if limit > 0 {
			page = offset/limit + 1
		} else {
			page = 1
		}
	}
	return limit, offset, page
}

// paginate slices items and builds page metadata.
func paginate[T any](items []T, filter Filter, defaultLimit, maxLimit int) ([]T, Pagination) {
	limit, offset, page := normalizePaging(filter, defaultLimit, maxLimit)
	total := len(items)

	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	totalPages := 0
	if limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	var nextPage, prevPage *int
	if page < totalPages {
		n := page + 1
		nextPage = &n
	}
	if page > 1 {
		p := page - 1
		prevPage = &p
	}

	return items[offset:end], Pagination{
		Page:         page,
		Limit:        limit,
		Total:        total,
		TotalPages:   totalPages,
		NextPage:     nextPage,
		PreviousPage: prevPage,
	}
}
