package service

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	v1 "gymmaster-api/api/report/v1"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/repository"
)

type reportCheckinFakeRepo struct {
	reportFakeRepo
}

func (f *reportCheckinFakeRepo) CheckinHeatmap(context.Context, repository.ReportScope) ([]repository.ReportHeatmapRow, error) {
	return nil, nil
}

func (f *reportCheckinFakeRepo) CheckinTotals(context.Context, repository.ReportScope) (*repository.ReportCheckinTotalsRow, error) {
	return &repository.ReportCheckinTotalsRow{Allowed: 3, Denied: 2, UniqueMembers: 3}, nil
}

func (f *reportCheckinFakeRepo) ExportCheckins(context.Context, repository.ReportScope, int) ([]repository.ReportCheckinExportRow, error) {
	return []repository.ReportCheckinExportRow{
		{CheckedInAt: time.Now(), MemberCode: "DP-1", Result: "DENIED", DenyReason: entity.CheckinDenyDayPassUsed},
		{CheckedInAt: time.Now(), MemberCode: "HV1", Result: "DENIED", DenyReason: entity.CheckinDenyOutsideAccessHours},
	}, nil
}

// C7-01: Excel check-in hiển thị lý do từ chối bằng tiếng Việt (kể cả vé ngày, giờ thấp điểm), không phải mã thô.
func TestReportExportCheckins_VietnameseDenyLabels(t *testing.T) {
	svc := reportTestService(&reportCheckinFakeRepo{})
	data, _, err := svc.Export(context.Background(), &v1.ExportReq{Filter: v1.Filter{From: "2026-10-01", To: "2026-10-14"}, Type: "checkins"})
	require.NoError(t, err)
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()
	head, _ := f.GetCellValue("Danh sách", "H4")
	require.Equal(t, "Lý do từ chối", head)
	r1, _ := f.GetCellValue("Danh sách", "H5")
	r2, _ := f.GetCellValue("Danh sách", "H6")
	assert.Equal(t, CheckinDenyMessage(entity.CheckinDenyDayPassUsed), r1)
	assert.Equal(t, CheckinDenyMessage(entity.CheckinDenyOutsideAccessHours), r2)
	assert.NotContains(t, r1+r2, "_")
}

// FIN-RPT-24: sheet "Theo thời gian" – cột theo loại ghi rõ là thực thu, đặt sau "Thực thu" (không cạnh "Tổng thu").
func TestReportExportRevenue_TimeSheetColumns(t *testing.T) {
	repo := &reportFakeRepo{revenue: []repository.ReportRevenueRow{
		{Bucket: "2026-10-02", BranchID: 1, Method: "CASH", ItemType: "MEMBERSHIP", Kind: "PAYMENT", Amount: 750000},
		{Bucket: "2026-10-02", BranchID: 1, Method: "CASH", ItemType: "MEMBERSHIP", Kind: "REFUND", Amount: -50000},
	}}
	svc := reportTestService(repo)
	data, _, err := svc.Export(context.Background(), &v1.ExportReq{Filter: v1.Filter{From: "2026-10-01", To: "2026-10-03"}, Type: "revenue"})
	require.NoError(t, err)
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()
	rows, err := f.GetRows("Theo thời gian")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 5)
	assert.Equal(t, []string{"Kỳ", "Tổng thu", "Hoàn tiền", "Thực thu", "Thực thu – Gói tập", "Thực thu – PT",
		"Thực thu – Sản phẩm", "Thực thu – Phí", "Thực thu – Khác"}, rows[3])
	note, _ := f.GetCellValue("Theo thời gian", "A3")
	assert.Contains(t, note, "đã trừ hoàn tiền")
}
