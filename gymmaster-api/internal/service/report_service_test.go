package service

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	v1 "gymmaster-api/api/report/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

func reportDate(s string) time.Time {
	t, _ := time.ParseInLocation(consts.DateLayout, s, reportLoc)
	return t
}

func TestReportParseRange(t *testing.T) {
	now := time.Date(2026, 10, 14, 9, 0, 0, 0, reportLoc)

	r, err := reportParseRange("", "", now)
	require.NoError(t, err)
	assert.Equal(t, "2026-10-01", r.Period().From)
	assert.Equal(t, "2026-10-14", r.Period().To)
	assert.Equal(t, 14, r.Days())
	// [Start, End) là nửa đêm giờ VN.
	assert.Equal(t, time.Date(2026, 9, 30, 17, 0, 0, 0, time.UTC), r.Start().UTC())
	assert.Equal(t, time.Date(2026, 10, 14, 17, 0, 0, 0, time.UTC), r.End().UTC())

	// Chỉ có to → from = đầu tháng của to.
	r, err = reportParseRange("", "2026-08-20", now)
	require.NoError(t, err)
	assert.Equal(t, "2026-08-01", r.Period().From)

	prev := reportRange{From: reportDate("2026-10-01"), To: reportDate("2026-10-14")}.Previous()
	assert.Equal(t, "2026-09-17", prev.Period().From)
	assert.Equal(t, "2026-09-30", prev.Period().To)

	_, err = reportParseRange("2026-10-10", "2026-10-01", now)
	assert.Error(t, err)
	_, err = reportParseRange("2020-01-01", "2026-10-01", now)
	assert.Error(t, err)

	long, _ := reportParseRange("2025-01-01", "2026-10-01", now)
	_, err = reportCheckGroupBy(long, "day")
	assert.Error(t, err)
	gb, err := reportCheckGroupBy(long, "month")
	require.NoError(t, err)
	assert.Equal(t, "month", gb)
}

func TestReportBuckets(t *testing.T) {
	r := reportRange{From: reportDate("2026-09-29"), To: reportDate("2026-10-14")}
	days := reportBuckets(r, "day")
	assert.Len(t, days, 16)
	assert.Equal(t, "2026-09-29", days[0])
	assert.Equal(t, "2026-10-14", days[15])

	// Tuần bắt đầu thứ 2 (trùng date_trunc('week')): 29/09/2026 là thứ 3 → mốc 28/09.
	weeks := reportBuckets(r, "week")
	assert.Equal(t, []string{"2026-09-28", "2026-10-05", "2026-10-12"}, weeks)
	assert.Equal(t, "Tuần 28/09", reportBucketLabel(weeks[0], "week"))

	months := reportBuckets(reportRange{From: reportDate("2026-01-31"), To: reportDate("2026-03-01")}, "month")
	assert.Equal(t, []string{"2026-01-01", "2026-02-01", "2026-03-01"}, months)
	assert.Equal(t, "02/2026", reportBucketLabel(months[1], "month"))
	assert.Equal(t, "14/10", reportBucketLabel("2026-10-14", "day"))

	// Chủ nhật thuộc tuần bắt đầu từ thứ 2 trước đó.
	assert.Equal(t, "2026-10-12", reportBucketStart(reportDate("2026-10-18"), "week").Format(consts.DateLayout))
}

func TestReportChangeAndPct(t *testing.T) {
	assert.Nil(t, reportChange(100, 0))
	assert.Equal(t, 50.0, *reportChange(150, 100))
	assert.Equal(t, -25.0, *reportChange(75, 100))
	assert.Equal(t, 33.3, reportPct(1, 3))
	assert.Equal(t, 0.0, reportPct(1, 0))
}

func TestReportBuildRevenue(t *testing.T) {
	buckets := []string{"2026-10-01", "2026-10-02", "2026-10-03"}
	rows := []repository.ReportRevenueRow{
		{Bucket: "2026-10-01", BranchID: 1, Method: "CASH", ItemType: "MEMBERSHIP", Kind: "PAYMENT", Amount: 1000000},
		{Bucket: "2026-10-01", BranchID: 2, Method: "TRANSFER", ItemType: "PRODUCT", Kind: "PAYMENT", Amount: 200000.4},
		{Bucket: "2026-10-03", BranchID: 1, Method: "CASH", ItemType: "MEMBERSHIP", Kind: "REFUND", Amount: -300000},
		{Bucket: "2026-10-03", BranchID: 1, Method: "CARD", ItemType: "PT", Kind: "PAYMENT", Amount: 500000},
	}
	res := reportBuildRevenue(rows, buckets, "day", map[uint]string{1: "CN 1", 2: "CN 2"})

	assert.Equal(t, 1700000.0, res.Totals.Payments)
	assert.Equal(t, 300000.0, res.Totals.Refunds)
	assert.Equal(t, 1400000.0, res.Totals.Net)
	require.Len(t, res.Series, 3)
	assert.Equal(t, 1200000.0, res.Series[0].Total)
	assert.Equal(t, 200000.0, res.Series[0].Product)
	assert.Equal(t, 0.0, res.Series[1].Total) // ngày không phát sinh vẫn có điểm
	assert.Equal(t, 200000.0, res.Series[2].Total)
	assert.Equal(t, -300000.0, res.Series[2].Membership)
	assert.Equal(t, 300000.0, res.Series[2].Refunds)

	// by_type luôn đủ 5 loại theo thứ tự cố định.
	require.Len(t, res.ByType, 5)
	assert.Equal(t, "MEMBERSHIP", res.ByType[0].Key)
	assert.Equal(t, 700000.0, res.ByType[0].Value)
	assert.Equal(t, 50.0, res.ByType[0].Pct)
	assert.Equal(t, "Tiền mặt", res.ByMethod[0].Label)
	assert.Equal(t, 700000.0, res.ByMethod[0].Value)

	require.Len(t, res.ByBranch, 2)
	assert.Equal(t, uint(1), res.ByBranch[0].BranchID)
	assert.Equal(t, "CN 1", res.ByBranch[0].BranchName)
	assert.Equal(t, 1200000.0, res.ByBranch[0].Amount)
}

func TestReportBuildHeatmap(t *testing.T) {
	// 05/10/2026 (CN) → 18/10/2026 (CN): 14 ngày, mỗi thứ xuất hiện 2 lần.
	r := reportRange{From: reportDate("2026-10-05"), To: reportDate("2026-10-18")}
	res := reportBuildHeatmap([]repository.ReportHeatmapRow{
		{Dow: 1, Hour: 18, Count: 10}, {Dow: 1, Hour: 7, Count: 4}, {Dow: 7, Hour: 9, Count: 6}, {Dow: 9, Hour: 1, Count: 99},
	}, r)
	require.Len(t, res.Matrix, 7)
	assert.Equal(t, "T2", res.Matrix[0].Label)
	assert.Equal(t, "CN", res.Matrix[6].Label)
	assert.Equal(t, int64(10), res.Matrix[0].Hours[18])
	assert.Equal(t, int64(14), res.Matrix[0].Total)
	assert.Equal(t, 7.0, res.Matrix[0].AvgDay)
	assert.Equal(t, int64(20), res.Total) // dòng dow=9 không hợp lệ bị bỏ
	assert.Equal(t, int64(10), res.Max)
	require.NotNil(t, res.Peak)
	assert.Equal(t, "T2 18:00", res.Peak.Label)
	assert.Equal(t, []int{18, 9, 7}, res.PeakHours)
	assert.Equal(t, 1.4, res.AvgPerDay)
	assert.Len(t, res.ByHour, 24)
}

func TestReportBuildOccupancy(t *testing.T) {
	rows := []repository.ReportClassRow{
		{ClassTypeID: 1, ClassTypeName: "Yoga", TrainerID: 10, TrainerName: "An", Hour: 18, Dow: 1, Capacity: 20, Booked: 15, Attended: 12, NoShow: 3},
		{ClassTypeID: 1, ClassTypeName: "Yoga", TrainerID: 11, TrainerName: "Bình", Hour: 6, Dow: 2, Capacity: 20, Booked: 5, Attended: 5},
		{ClassTypeID: 2, ClassTypeName: "Spin", TrainerID: 10, TrainerName: "An", Hour: 18, Dow: 2, Capacity: 10, Booked: 10, Attended: 8, NoShow: 2, Waitlist: 4},
	}
	sum, types, trainers, slots, days := reportBuildOccupancy(rows)
	assert.Equal(t, int64(3), sum.Sessions)
	assert.Equal(t, int64(50), sum.Capacity)
	assert.Equal(t, 60.0, sum.FillRate)
	assert.Equal(t, 83.3, sum.AttendanceRate)

	require.Len(t, types, 2)
	assert.Equal(t, "Yoga", types[0].Label) // nhiều buổi nhất trước
	assert.Equal(t, 50.0, types[0].FillRate)
	require.Len(t, trainers, 2)
	assert.Equal(t, "An", trainers[0].Label)
	assert.Equal(t, 83.3, trainers[0].FillRate)
	require.Len(t, slots, 2)
	assert.Equal(t, "06:00", slots[0].Key) // khung giờ theo thứ tự thời gian
	assert.Equal(t, "18:00", slots[1].Key)
	require.Len(t, days, 2)
	assert.Equal(t, "T2", days[0].Label)
	assert.Equal(t, int64(4), days[1].Waitlist)
}

func TestReportBuildRetention(t *testing.T) {
	m9, m10 := "2026-09", "2026-10"
	sum, months := reportBuildRetention([]repository.ReportRetentionRow{
		{Month: &m9, Expired: 10, Renewed: 6, Pending: 0},
		{Month: &m10, Expired: 4, Renewed: 1, Pending: 2},
		{Month: nil, Expired: 14, Renewed: 7, Pending: 2},
	}, reportRange{From: reportDate("2026-08-01"), To: reportDate("2026-10-14")})
	require.Len(t, months, 3)
	assert.Equal(t, "2026-08", months[0].Month) // tháng không có dữ liệu vẫn có
	assert.Equal(t, int64(0), months[0].Expired)
	assert.Equal(t, 60.0, months[1].RenewalRate)
	assert.Equal(t, int64(4), months[1].Churned)
	assert.Equal(t, 40.0, months[1].ChurnRate)
	assert.Equal(t, int64(1), months[2].Churned)
	assert.Equal(t, "10/2026", months[2].Label)
	assert.Equal(t, 50.0, sum.RenewalRate)
	assert.Equal(t, int64(5), sum.Churned)
	assert.Equal(t, "Tổng", sum.Label)
}

func TestReportBuildProductsAndSubscriptions(t *testing.T) {
	res := reportBuildProducts([]repository.ReportProductRow{
		{ProductID: 1, Name: "Whey", CategoryName: "TPBS", Quantity: 3, Revenue: 3000000, Cost: 2100000},
		{ProductID: 2, Name: "Nước", Quantity: 50, Revenue: 500000, Cost: 250000},
		{ProductID: 3, Name: "BCAA", CategoryName: "TPBS", Quantity: 2, Revenue: 800000, Cost: 500000},
	}, 2)
	require.Len(t, res.Top, 2)
	assert.Equal(t, "Whey", res.Top[0].Name)
	assert.Equal(t, 900000.0, res.Top[0].GrossProfit)
	assert.Equal(t, 30.0, res.Top[0].Margin)
	assert.Equal(t, 3, res.Products)
	assert.Equal(t, 4300000.0, res.Totals.Revenue)
	assert.Equal(t, 1450000.0, res.Totals.GrossProfit)
	require.Len(t, res.ByCategory, 2)
	assert.Equal(t, "TPBS", res.ByCategory[0].CategoryName)
	assert.Equal(t, "Chưa phân loại", res.ByCategory[1].CategoryName)

	subs := reportBuildSubscriptions([]repository.ReportSubEventRow{
		{Bucket: "2026-10-01", Kind: "NEW", Count: 3}, {Bucket: "2026-10-01", Kind: "RENEWED", Count: 1},
		{Bucket: "2026-10-02", Kind: "CANCELLED", Count: 2},
	}, []repository.ReportPlanRow{{PlanID: 1, Collected: 300}, {PlanID: 2, Collected: 100}},
		[]string{"2026-10-01", "2026-10-02"}, "day")
	assert.Equal(t, int64(3), subs.Counts.New)
	assert.Equal(t, int64(2), subs.Series[1].Cancelled)
	assert.Equal(t, 75.0, subs.ByPlan[0].Pct)
}

// --- Service với repository giả ---

type reportFakeRepo struct {
	repository.ReportRepository // các method không dùng sẽ panic nếu bị gọi
	scopes                      []repository.ReportScope
	revenue                     []repository.ReportRevenueRow
}

func (f *reportFakeRepo) BranchNames(context.Context) (map[uint]string, error) {
	return map[uint]string{1: "Cầu Giấy"}, nil
}

func (f *reportFakeRepo) BranchExists(_ context.Context, id uint) (bool, error) { return id == 1, nil }

func (f *reportFakeRepo) RevenueFlows(_ context.Context, s repository.ReportScope, _ string) ([]repository.ReportRevenueRow, error) {
	f.scopes = append(f.scopes, s)
	if len(f.scopes) == 1 {
		return f.revenue, nil
	}
	return []repository.ReportRevenueRow{{Bucket: "2026-09-01", Kind: "PAYMENT", Amount: 500000}}, nil
}

func (f *reportFakeRepo) ExportInvoices(context.Context, repository.ReportScope, int) ([]repository.ReportInvoiceExportRow, error) {
	return []repository.ReportInvoiceExportRow{
		{InvoiceNo: "HD1", IssuedAt: time.Now(), MemberName: "A", Total: 1000000, Paid: 1000000, Status: "PAID", Source: "COUNTER"},
		{InvoiceNo: "HD2", IssuedAt: time.Now(), MemberName: "B", Total: 500000, Status: "UNPAID", Source: "POS"},
	}, nil
}

func reportTestService(repo repository.ReportRepository) *reportService {
	return &reportService{repo: repo, now: func() time.Time { return time.Date(2026, 10, 14, 10, 0, 0, 0, reportLoc) }}
}

func TestReportServiceRevenueScopeAndCompare(t *testing.T) {
	repo := &reportFakeRepo{revenue: []repository.ReportRevenueRow{
		{Bucket: "2026-10-02", BranchID: 1, Method: "CASH", ItemType: "MEMBERSHIP", Kind: "PAYMENT", Amount: 750000},
	}}
	svc := reportTestService(repo)
	// Nhân viên chi nhánh 1.
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 5, Roles: []string{consts.RoleManager}, BranchIDs: []uint{1}})

	res, err := svc.Revenue(ctx, &v1.RevenueReq{GroupBy: v1.GroupBy{GroupBy: "day"}})
	require.NoError(t, err)
	assert.Equal(t, []uint{1}, repo.scopes[0].BranchIDs)
	assert.Equal(t, "2026-10-01", repo.scopes[0].FromDate)
	assert.Equal(t, "2026-09-17", repo.scopes[1].FromDate) // kỳ trước cùng độ dài
	assert.Len(t, res.Series, 14)
	assert.Equal(t, 750000.0, res.Totals.Net)
	assert.Equal(t, 500000.0, res.Previous.Net)
	require.NotNil(t, res.Previous.ChangePct)
	assert.Equal(t, 50.0, *res.Previous.ChangePct)

	// Chi nhánh khác → 403.
	_, err = svc.Revenue(ctx, &v1.RevenueReq{Filter: v1.Filter{BranchID: 2}})
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), reportErrCode(err))
}

func reportErrCode(err error) int { return gerror.Code(err).Code() }

func TestReportServiceExportInvoices(t *testing.T) {
	svc := reportTestService(&reportFakeRepo{})
	data, name, err := svc.Export(context.Background(), &v1.ExportReq{Filter: v1.Filter{From: "2026-10-01", To: "2026-10-14"}, Type: "invoices"})
	require.NoError(t, err)
	assert.Equal(t, "bao-cao-hoa-don_20261001_20261014.xlsx", name)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()
	assert.Equal(t, []string{"Tổng hợp", "Danh sách"}, f.GetSheetList())
	title, _ := f.GetCellValue("Tổng hợp", "A1")
	assert.Equal(t, "BÁO CÁO HÓA ĐƠN", title)
	header, _ := f.GetCellValue("Danh sách", "A4")
	assert.Equal(t, "Số HĐ", header)
	wo, _ := f.GetCellValue("Danh sách", "I4")
	assert.Equal(t, "Xóa nợ (hủy gói)", wo)
	total, _ := f.GetCellValue("Danh sách", "J7") // dòng tổng sau 2 hóa đơn (cột I = xóa nợ)
	assert.Contains(t, total, "1,500,000")
	style, _ := f.GetCellStyle("Danh sách", "J5")
	st, err := f.GetStyle(style)
	require.NoError(t, err)
	require.NotNil(t, st.CustomNumFmt)
	assert.Contains(t, *st.CustomNumFmt, "#,##0")
	hs, _ := f.GetCellStyle("Danh sách", "A4")
	hst, _ := f.GetStyle(hs)
	assert.True(t, hst.Font.Bold)
	panes, _ := f.GetPanes("Danh sách")
	assert.True(t, panes.Freeze)
	w, _ := f.GetColWidth("Danh sách", "D")
	assert.Equal(t, 26.0, w)
}

// D6 (FIN-RPT-09/10): revenue/overview với branch_id không tồn tại → 404 (không trả báo cáo toàn số 0).
func TestReportServiceUnknownBranch404(t *testing.T) {
	svc := reportTestService(&reportFakeRepo{})
	owner := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	_, err := svc.Revenue(owner, &v1.RevenueReq{Filter: v1.Filter{BranchID: 999999}})
	require.Error(t, err)
	assert.Equal(t, rescode.NotFound.Code(), reportErrCode(err))
	_, err = svc.Overview(owner, &v1.OverviewReq{BranchID: 999999})
	require.Error(t, err)
	assert.Equal(t, rescode.NotFound.Code(), reportErrCode(err))
}
