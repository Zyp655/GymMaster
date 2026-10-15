package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/systemext/v1"
)

func TestValidateLegacyImportRows(t *testing.T) {
	rows := []v1.LegacyImportRow{
		{
			FullName:     "Nguyễn Văn A",
			Phone:        "0912345678",
			SubStartDate: "2026-01-01",
			SubEndDate:   "2026-12-31",
			DebtAmount:   500000,
		},
		{
			FullName: "Trùng SĐT",
			Phone:    "0912345678",
		},
		{
			FullName: "",
			Phone:    "0988777666",
		},
		{
			FullName:     "Sai ngày gói tập",
			Phone:        "0977666555",
			SubStartDate: "2026-12-31",
			SubEndDate:   "2026-01-01",
		},
	}

	okCount, errs := ValidateLegacyImportRows(rows)
	assert.Equal(t, 1, okCount)
	require.Len(t, errs, 3)
	assert.Equal(t, 2, errs[0].Row)
	assert.Contains(t, errs[0].Message, "Trùng số điện thoại")
	assert.Equal(t, 3, errs[1].Row)
	assert.Equal(t, 4, errs[2].Row)
}
