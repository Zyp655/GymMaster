package dto

import (
	"slices"
	"strings"

	"gorm.io/gorm"
)

const (
	DefaultLimit = 10
	MaxLimit     = 100
)

// PageReq được nhúng vào các Req danh sách (api/<module>/v1) để có sẵn tham số phân trang.
type PageReq struct {
	Page     int    `json:"page"      in:"query" d:"1"    v:"min:1#Trang phải >= 1"                    dc:"Trang hiện tại"`
	Limit    int    `json:"limit"     in:"query" d:"10"   v:"between:1,100#Số bản ghi mỗi trang từ 1 đến 100" dc:"Số bản ghi mỗi trang"`
	Keyword  string `json:"keyword"   in:"query"                                                        dc:"Từ khóa tìm kiếm"`
	OrderBy  string `json:"order_by"  in:"query"                                                        dc:"Cột sắp xếp"`
	OrderDir string `json:"order_dir" in:"query" d:"DESC" v:"in:ASC,DESC,asc,desc#Chiều sắp xếp phải là ASC hoặc DESC" dc:"ASC | DESC"`
}

// PageRes là dạng chung của dữ liệu danh sách có phân trang.
type PageRes[T any] struct {
	Items       []T   `json:"items"`
	Total       int64 `json:"total"`
	TotalPage   int   `json:"total_page"`
	CurrentPage int   `json:"current_page"`
	Limit       int   `json:"limit"`
}

// Query là tham số danh sách đã được chuẩn hóa, truyền từ service xuống repository.
type Query struct {
	Page     int
	Limit    int
	Keyword  string
	OrderBy  string
	OrderDir string
	Filters  map[string]any
}

// ToQuery chuẩn hóa PageReq; orderable là whitelist cột cho phép sắp xếp (chống SQL injection qua order_by).
func (p PageReq) ToQuery(orderable []string, defaultOrder string) Query {
	q := Query{Page: p.Page, Limit: p.Limit, Keyword: strings.TrimSpace(p.Keyword), Filters: map[string]any{}}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 {
		q.Limit = DefaultLimit
	}
	if q.Limit > MaxLimit {
		q.Limit = MaxLimit
	}
	q.OrderBy = defaultOrder
	if slices.Contains(orderable, p.OrderBy) {
		q.OrderBy = p.OrderBy
	}
	q.OrderDir = "DESC"
	if strings.EqualFold(p.OrderDir, "ASC") {
		q.OrderDir = "ASC"
	}
	return q
}

// Paginate là GORM scope áp dụng ORDER BY + LIMIT/OFFSET.
func (q Query) Paginate(db *gorm.DB) *gorm.DB {
	return db.Order(q.OrderBy + " " + q.OrderDir).Limit(q.Limit).Offset((q.Page - 1) * q.Limit)
}

// NewPageRes dựng PageRes từ danh sách đã map và tổng số bản ghi.
func NewPageRes[T any](items []T, total int64, q Query) *PageRes[T] {
	if items == nil {
		items = []T{}
	}
	totalPage := int((total + int64(q.Limit) - 1) / int64(q.Limit))
	return &PageRes[T]{Items: items, Total: total, TotalPage: totalPage, CurrentPage: q.Page, Limit: q.Limit}
}

// Map chuyển slice entity sang slice DTO.
func Map[S any, D any](src []S, fn func(S) D) []D {
	out := make([]D, 0, len(src))
	for _, s := range src {
		out = append(out, fn(s))
	}
	return out
}
