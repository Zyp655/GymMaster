package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToQuery_Normalize(t *testing.T) {
	orderable := []string{"id", "name"}

	q := PageReq{Page: 0, Limit: 1000, OrderBy: "name; DROP TABLE users", OrderDir: "asc"}.ToQuery(orderable, "id")
	assert.Equal(t, 1, q.Page)
	assert.Equal(t, MaxLimit, q.Limit)
	assert.Equal(t, "id", q.OrderBy, "cột không nằm trong whitelist → dùng mặc định")
	assert.Equal(t, "ASC", q.OrderDir)

	q = PageReq{Page: 3, Limit: 20, OrderBy: "name", OrderDir: "lung tung", Keyword: "  cầu giấy "}.ToQuery(orderable, "id")
	assert.Equal(t, "name", q.OrderBy)
	assert.Equal(t, "DESC", q.OrderDir)
	assert.Equal(t, "cầu giấy", q.Keyword)
}

func TestNewPageRes(t *testing.T) {
	res := NewPageRes[int](nil, 21, Query{Page: 2, Limit: 10})
	assert.Equal(t, 3, res.TotalPage)
	assert.NotNil(t, res.Items, "items luôn là mảng, không phải null")
}
