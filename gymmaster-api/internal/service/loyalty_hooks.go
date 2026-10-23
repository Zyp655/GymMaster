package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/entity"
)

// Tỷ lệ tích điểm tự động (PLAN Sprint 22): cộng qua hook sau nghiệp vụ, mỗi nguồn chỉ cộng một lần
// (LoyaltyEarner.EarnPoints idempotent theo member + reason + ref_type + ref_id).
const (
	loyaltyVNDPerPoint     = 10000 // thanh toán gói tập/PT: 1 điểm cho mỗi 10.000đ
	loyaltyCheckinPoints   = 5     // check-in hợp lệ: tối đa 1 lần/ngày
	loyaltyReviewPoints    = 20    // đánh giá buổi tập
	loyaltyReferralPoints  = 100   // người giới thiệu, khi người được giới thiệu thanh toán gói đầu tiên
	loyaltyRefCheckinDay   = "CHECKIN_DAY"
	loyaltyRefInvoice      = "INVOICE"
	loyaltyRefReview       = "REVIEW"
	loyaltyRefReferralMemb = "MEMBER"
)

var loyaltyEarnerHook atomic.Pointer[LoyaltyEarner]

// SetLoyaltyEarner đăng ký dịch vụ cộng điểm dùng chung cho các module check-in/đánh giá (module loyalty gọi lúc
// khởi động). nil = tắt tích điểm tự động.
func SetLoyaltyEarner(e LoyaltyEarner) {
	if e == nil {
		loyaltyEarnerHook.Store(nil)
		return
	}
	loyaltyEarnerHook.Store(&e)
}

// awardLoyalty cộng điểm "best effort": lỗi chỉ ghi log, không làm hỏng nghiệp vụ chính (check-in, đánh giá…).
func awardLoyalty(ctx context.Context, memberID uint, points int, reason, refType string, refID uint, note string) {
	p := loyaltyEarnerHook.Load()
	if p == nil || memberID == 0 || points <= 0 {
		return
	}
	if _, _, err := (*p).EarnPoints(ctx, memberID, points, reason, refType, &refID, note); err != nil {
		g.Log().Warningf(ctx, "tích điểm %s cho hội viên %d: %v", reason, memberID, err)
	}
}

// loyaltyCheckinDayRef: mã nguồn theo ngày (YYYYMMDD giờ VN) để check-in chỉ được cộng điểm 1 lần/ngày.
func loyaltyCheckinDayRef(at time.Time) uint {
	n, _ := strconv.ParseUint(at.In(checkinVNLoc).Format("20060102"), 10, 64)
	return uint(n)
}

// awardCheckinPoints: gọi SAU khi transaction check-in đã commit.
func awardCheckinPoints(ctx context.Context, memberID uint, at time.Time) {
	awardLoyalty(ctx, memberID, loyaltyCheckinPoints, entity.LoyaltyReasonCheckin, loyaltyRefCheckinDay, loyaltyCheckinDayRef(at),
		fmt.Sprintf("Check-in ngày %s", at.In(checkinVNLoc).Format("02/01/2006")))
}

// awardReviewPoints: gọi SAU khi đánh giá đã lưu.
func awardReviewPoints(ctx context.Context, memberID, reviewID uint) {
	awardLoyalty(ctx, memberID, loyaltyReviewPoints, entity.LoyaltyReasonReview, loyaltyRefReview, reviewID, "Cảm ơn bạn đã đánh giá buổi tập")
}

// loyaltyInvoicePoints: số điểm cho hóa đơn đã thanh toán đủ (theo tổng phải thu, đã trừ phần xóa nợ).
func loyaltyInvoicePoints(inv *entity.Invoice) int {
	amount := inv.Total - inv.WriteOffAmount
	if amount <= 0 {
		return 0
	}
	return int(math.Floor(amount / loyaltyVNDPerPoint))
}

// ReferrerLookup trả id người giới thiệu của hội viên (nil nếu không có).
type ReferrerLookup func(ctx context.Context, memberID uint) (*uint, error)

// NewLoyaltyInvoicePaidHandler: hook "hóa đơn đã thanh toán đủ" cho dòng gói tập/PT – cộng điểm theo giá trị hóa đơn
// (một lần cho mỗi hóa đơn dù có nhiều dòng) và thưởng người giới thiệu ở lần thanh toán đầu tiên của hội viên.
// Chạy trong transaction thanh toán; EarnPoints không sinh lỗi SQL khi trùng (ON CONFLICT DO NOTHING) nên an toàn.
func NewLoyaltyInvoicePaidHandler(earner LoyaltyEarner, referrerOf ReferrerLookup) InvoiceItemHandler {
	return func(ctx context.Context, inv *entity.Invoice, _ *entity.InvoiceItem) error {
		if earner == nil || inv == nil || inv.MemberID == nil {
			return nil
		}
		memberID := *inv.MemberID
		if pts := loyaltyInvoicePoints(inv); pts > 0 {
			invID := inv.ID
			if _, _, err := earner.EarnPoints(ctx, memberID, pts, entity.LoyaltyReasonRenew, loyaltyRefInvoice, &invID,
				"Tích điểm thanh toán hóa đơn "+inv.InvoiceNo); err != nil {
				g.Log().Warningf(ctx, "tích điểm hóa đơn %s: %v", inv.InvoiceNo, err)
			}
		}
		if referrerOf == nil {
			return nil
		}
		ref, err := referrerOf(ctx, memberID)
		if err != nil || ref == nil || *ref == memberID {
			return nil //nolint:nilerr // không có người giới thiệu / lỗi tra cứu: bỏ qua thưởng, không chặn thanh toán
		}
		if _, _, err := earner.EarnPoints(ctx, *ref, loyaltyReferralPoints, entity.LoyaltyReasonReferral, loyaltyRefReferralMemb, &memberID,
			"Giới thiệu thành công hội viên mới"); err != nil {
			g.Log().Warningf(ctx, "thưởng giới thiệu cho hội viên %d: %v", *ref, err)
		}
		return nil
	}
}
