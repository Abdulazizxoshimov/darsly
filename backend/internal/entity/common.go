package entity

import "time"

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

type ListResponse[T any] struct {
	Items      []T `json:"items"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

type Filter struct {
	Page        int        `form:"page"         json:"page"`
	Limit       int        `form:"limit"        json:"limit"`
	SortBy      string     `form:"sort_by"      json:"sort_by"`
	SortOrder   SortOrder  `form:"sort_order"   json:"sort_order"`
	Search      string     `form:"search"       json:"search"`
	CreatedFrom *time.Time `form:"created_from" json:"created_from,omitempty"`
	CreatedTo   *time.Time `form:"created_to"   json:"created_to,omitempty"`
}

func (f *Filter) Offset() int {
	if f.Page < 1 {
		return 0
	}
	return (f.Page - 1) * f.GetLimit()
}

func (f *Filter) GetLimit() int {
	if f.Limit < 1 {
		return 20
	}
	if f.Limit > 100 {
		return 100
	}
	return f.Limit
}

func (f *Filter) GetSortOrder() SortOrder {
	if f.SortOrder == SortAsc || f.SortOrder == SortDesc {
		return f.SortOrder
	}
	return SortDesc
}
