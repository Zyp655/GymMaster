package service

import (
	"context"
	"errors"
	"time"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/repository"
)

// Các tác vụ định kỳ của module gói tập. Mỗi bản ghi được xử lý trong transaction riêng để một lỗi
// không chặn các bản ghi khác; lỗi đầu tiên được trả về sau khi đã xử lý hết.

func (s *subscriptionService) ExpireDue(ctx context.Context, today time.Time) (int, error) {
	today = subDateOf(today) // chấp nhận cả time.Now() lẫn ngày 00:00 UTC
	var firstErr error
	keep := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	n := 0

	ids, err := s.repo.IDsToExpire(ctx, today)
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		changed := false
		keep(s.tx.WithTx(ctx, func(ctx context.Context) error {
			sub, err := s.repo.FindByID(ctx, id, true)
			if err != nil {
				return err
			}
			if !subShouldExpire(sub, today) {
				return nil
			}
			if err := s.repo.UpdateFields(ctx, sub.ID, map[string]any{"status": entity.SubStatusExpired}); err != nil {
				return err
			}
			if err := s.repo.CancelOpenFreezes(ctx, sub.ID, entity.FreezeStatusCancelled); err != nil {
				return err
			}
			changed = true
			return s.event(ctx, sub.ID, entity.SubEventExpired, map[string]any{"end_date": subFmt(sub.EndDate)})
		}))
		if changed {
			n++
		}
	}

	// PENDING quá hạn thanh toán: tạo trước 00:00 (giờ VN) của ngày today − 3.
	cutoff := time.Date(today.Year(), today.Month(), today.Day()-subPendingTTLDays, 0, 0, 0, 0, subVNLoc)
	stale, err := s.repo.StalePending(ctx, cutoff)
	if err != nil {
		keep(err)
		return n, firstErr
	}
	for _, sub := range stale {
		changed := false
		keep(s.tx.WithTx(ctx, func(ctx context.Context) error {
			cur, err := s.repo.FindByID(ctx, sub.ID, true)
			if err != nil || cur.Status != entity.SubStatusPending {
				return err
			}
			if cur.InvoiceID != nil {
				inv, err := s.invoices.FindInvoice(ctx, *cur.InvoiceID)
				if err != nil {
					return err
				}
				if inv.Status == entity.InvoiceStatusUnpaid && inv.PaidAmount == 0 {
					// VoidHandler chuyển gói sang CANCELLED (và khôi phục gói trước nếu là nâng cấp/chuyển nhượng).
					if err := s.invoices.Void(ctx, inv.ID, "Quá hạn thanh toán"); err != nil {
						return err
					}
					changed = true
					return nil
				}
				return nil
			}
			if err := s.repo.UpdateFields(ctx, cur.ID, map[string]any{"status": entity.SubStatusCancelled}); err != nil {
				return err
			}
			changed = true
			return s.event(ctx, cur.ID, entity.SubEventCancelled, map[string]any{"reason": "Quá hạn thanh toán"})
		}))
		if changed {
			n++
		}
	}
	return n, firstErr
}

// subShouldExpire: gói ACTIVE/FROZEN có end_date < today.
func subShouldExpire(sub *entity.Subscription, today time.Time) bool {
	return (sub.Status == entity.SubStatusActive || sub.Status == entity.SubStatusFrozen) && subNorm(sub.EndDate).Before(subNorm(today))
}

func (s *subscriptionService) ApplyDueFreezes(ctx context.Context, today time.Time) (int, error) {
	today = subDateOf(today) // chấp nhận cả time.Now() lẫn ngày 00:00 UTC
	var firstErr error
	n := 0
	process := func(ids []uint, fn func(ctx context.Context, sub *entity.Subscription, f *entity.SubscriptionFreeze) (bool, error)) {
		for _, id := range ids {
			changed := false
			err := s.tx.WithTx(ctx, func(ctx context.Context) error {
				f, err := s.repo.FindFreeze(ctx, id, true)
				if err != nil {
					return err
				}
				sub, err := s.repo.FindByID(ctx, f.SubscriptionID, true)
				if err != nil {
					return err
				}
				changed, err = fn(ctx, sub, f)
				return err
			})
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if changed {
				n++
			}
		}
	}

	// 1) Kết thúc bảo lưu đã hết hạn.
	endIDs, err := s.repo.FreezeIDsToEnd(ctx, today)
	if err != nil {
		return 0, err
	}
	process(endIDs, func(ctx context.Context, sub *entity.Subscription, f *entity.SubscriptionFreeze) (bool, error) {
		if f.Status != entity.FreezeStatusApproved || sub.Status != entity.SubStatusFrozen || !subNorm(f.EndDate).Before(today) {
			return false, nil
		}
		return true, s.unfreeze(ctx, sub, f)
	})

	// 2) Áp các bảo lưu đã duyệt đến ngày bắt đầu.
	startIDs, err := s.repo.FreezeIDsToStart(ctx, today)
	if err != nil {
		return n, errors.Join(firstErr, err)
	}
	process(startIDs, func(ctx context.Context, sub *entity.Subscription, f *entity.SubscriptionFreeze) (bool, error) {
		if f.Status != entity.FreezeStatusApproved || sub.Status != entity.SubStatusActive ||
			subNorm(f.StartDate).After(today) || subNorm(f.EndDate).Before(today) {
			return false, nil
		}
		return true, s.applyFreeze(ctx, sub, f)
	})
	return n, firstErr
}

func (s *subscriptionService) ListExpiring(ctx context.Context, days []int, today time.Time) ([]repository.ExpiringSubscription, error) {
	today = subDateOf(today) // chấp nhận cả time.Now() lẫn ngày 00:00 UTC
	dates := make([]time.Time, 0, len(days))
	for _, d := range days {
		if d >= 0 {
			dates = append(dates, subAddDays(today, d))
		}
	}
	rows, err := s.repo.ListExpiring(ctx, dates)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].DaysLeft = subDaysBetween(today, rows[i].EndDate)
	}
	return rows, nil
}
