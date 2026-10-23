package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gymmaster-api/internal/entity"
)

type earnCall struct {
	memberID uint
	points   int
	reason   string
	refType  string
	refID    uint
}

// fakeEarner mô phỏng EarnPoints idempotent theo (member, reason, refType, refID).
type fakeEarner struct {
	calls []earnCall
	seen  map[earnCall]bool
}

func (f *fakeEarner) EarnPoints(_ context.Context, memberID uint, points int, reason, refType string, refID *uint, _ string) (bool, int, error) {
	c := earnCall{memberID: memberID, reason: reason, refType: refType, refID: *refID}
	if f.seen == nil {
		f.seen = map[earnCall]bool{}
	}
	if f.seen[c] {
		return false, 0, nil
	}
	f.seen[c] = true
	c.points = points
	f.calls = append(f.calls, c)
	return true, 0, nil
}

func TestLoyaltyInvoicePoints(t *testing.T) {
	assert.Equal(t, 30, loyaltyInvoicePoints(&entity.Invoice{Total: 1350000, WriteOffAmount: 1050000}), "phần nợ đã xóa không được tích điểm")
	assert.Equal(t, 135, loyaltyInvoicePoints(&entity.Invoice{Total: 1359999}))
	assert.Equal(t, 0, loyaltyInvoicePoints(&entity.Invoice{Total: 9999}))
	assert.Equal(t, 0, loyaltyInvoicePoints(&entity.Invoice{Total: 0}))
}

func TestLoyaltyInvoicePaidHandler_InvoiceAndReferralOnce(t *testing.T) {
	e := &fakeEarner{}
	referrer := uint(7)
	h := NewLoyaltyInvoicePaidHandler(e, func(context.Context, uint) (*uint, error) { return &referrer, nil })
	member := uint(3)
	inv := &entity.Invoice{ID: 11, InvoiceNo: "HD1", MemberID: &member, Total: 500000}

	ctx := context.Background()
	// Hai dòng (gói + PT) cùng hóa đơn → hook chạy 2 lần nhưng chỉ cộng 1 lần.
	require.NoError(t, h(ctx, inv, &entity.InvoiceItem{ItemType: entity.InvoiceItemMembership}))
	require.NoError(t, h(ctx, inv, &entity.InvoiceItem{ItemType: entity.InvoiceItemPT}))
	// Hóa đơn thứ hai của cùng hội viên: điểm hóa đơn mới, KHÔNG thưởng giới thiệu lần nữa.
	inv2 := &entity.Invoice{ID: 12, InvoiceNo: "HD2", MemberID: &member, Total: 200000}
	require.NoError(t, h(ctx, inv2, &entity.InvoiceItem{ItemType: entity.InvoiceItemMembership}))

	assert.Equal(t, []earnCall{
		{memberID: 3, points: 50, reason: entity.LoyaltyReasonRenew, refType: loyaltyRefInvoice, refID: 11},
		{memberID: 7, points: loyaltyReferralPoints, reason: entity.LoyaltyReasonReferral, refType: loyaltyRefReferralMemb, refID: 3},
		{memberID: 3, points: 20, reason: entity.LoyaltyReasonRenew, refType: loyaltyRefInvoice, refID: 12},
	}, e.calls)
}

func TestLoyaltyInvoicePaidHandler_WalkInSkipped(t *testing.T) {
	e := &fakeEarner{}
	h := NewLoyaltyInvoicePaidHandler(e, nil)
	require.NoError(t, h(context.Background(), &entity.Invoice{Total: 500000}, &entity.InvoiceItem{}))
	assert.Empty(t, e.calls, "hóa đơn khách vãng lai (không gắn hội viên) không tích điểm")
}

func TestAwardCheckinPoints_OncePerDay(t *testing.T) {
	e := &fakeEarner{}
	SetLoyaltyEarner(e)
	t.Cleanup(func() { SetLoyaltyEarner(nil) })
	ctx := context.Background()
	morning := time.Date(2026, 10, 22, 6, 0, 0, 0, checkinVNLoc)
	awardCheckinPoints(ctx, 5, morning)
	awardCheckinPoints(ctx, 5, morning.Add(10*time.Hour)) // cùng ngày
	awardCheckinPoints(ctx, 5, morning.Add(20*time.Hour)) // 02:00 hôm sau (giờ VN)
	require.Len(t, e.calls, 2)
	assert.Equal(t, uint(20261022), e.calls[0].refID)
	assert.Equal(t, uint(20261023), e.calls[1].refID)
	assert.Equal(t, loyaltyCheckinPoints, e.calls[0].points)

	SetLoyaltyEarner(nil)
	awardCheckinPoints(ctx, 5, morning.AddDate(0, 0, 3))
	assert.Len(t, e.calls, 2, "chưa đăng ký hook → không tích điểm")
}
