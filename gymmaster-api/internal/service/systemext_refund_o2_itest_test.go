//go:build integration

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/systemext/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Hoàn VNPay hết số đã thu của HĐ PARTIAL phải chạy RefundHandler của InvoiceService dùng chung
// → gói PENDING gắn HĐ tự chuyển CANCELLED; hai lần hoàn cùng giây có gateway_ref khác nhau (FIN-VNP-19).
func TestIT_RefundOnlineTxn_UsesSharedInvoiceHooks(t *testing.T) {
	db := checkinITOpen(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	scan := func(dest *uint, sql string, args ...any) {
		t.Helper()
		require.NoError(t, db.Raw(sql, args...).Scan(dest).Error, sql)
	}
	var branchID, planID, memberID, subID, invID, txnID uint
	scan(&branchID, `SELECT id FROM branches WHERE status = 'ACTIVE' ORDER BY id LIMIT 1`)
	scan(&planID, `INSERT INTO membership_plans (code, name, type, duration_days, price) VALUES (?, 'IT O2 gói', 'TIME', 30, 1000000) RETURNING id`, "ITO2"+suffix)
	scan(&memberID, `INSERT INTO members (full_name, phone, home_branch_id) VALUES ('IT O2 HV', ?, ?) RETURNING id`,
		"09"+fmt.Sprintf("%08d", time.Now().UnixNano()%100_000_000), branchID)
	scan(&invID, `INSERT INTO invoices (member_id, branch_id, subtotal, total, paid_amount, status) VALUES (?, ?, 1000000, 1000000, 300000, 'PARTIAL') RETURNING id`, memberID, branchID)
	scan(&subID, `INSERT INTO subscriptions (member_id, plan_id, branch_id, start_date, end_date, price, status, invoice_id)
		VALUES (?, ?, ?, current_date + 1, current_date + 30, 1000000, 'PENDING', ?) RETURNING id`, memberID, planID, branchID, invID)
	require.NoError(t, db.Exec(`INSERT INTO invoice_items (invoice_id, item_type, description, unit_price, amount, ref_id) VALUES (?, 'MEMBERSHIP', 'IT O2', 1000000, 1000000, ?)`, invID, subID).Error)
	require.NoError(t, db.Exec(`INSERT INTO payments (invoice_id, amount, method, reference) VALUES (?, 300000, 'VNPAY', ?)`, invID, "ITO2"+suffix).Error)
	scan(&txnID, `INSERT INTO online_payment_txns (invoice_id, member_id, txn_ref, amount, status) VALUES (?, ?, ?, 300000, 'SUCCESS') RETURNING id`, invID, memberID, "ITO2"+suffix)
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM audit_logs WHERE entity_id = ? AND entity = 'online_payment_txn'`,
		} {
			db.Exec(q, txnID)
		}
		db.Exec(`DELETE FROM refunds WHERE invoice_id = ?`, invID)
		db.Exec(`DELETE FROM online_payment_txns WHERE id = ?`, txnID)
		db.Exec(`DELETE FROM payments WHERE invoice_id = ?`, invID)
		db.Exec(`DELETE FROM subscription_events WHERE subscription_id = ?`, subID)
		db.Exec(`DELETE FROM invoice_items WHERE invoice_id = ?`, invID)
		db.Exec(`DELETE FROM subscriptions WHERE id = ?`, subID)
		db.Exec(`DELETE FROM invoices WHERE id = ?`, invID)
		db.Exec(`DELETE FROM members WHERE id = ?`, memberID)
		db.Exec(`DELETE FROM membership_plans WHERE id = ?`, planID)
	})

	txm := postgres.NewTxManager(db)
	invoices := NewInvoiceService(repository.NewInvoiceRepository(db), nil, txm, nil)
	_ = NewSubscriptionService(repository.NewSubscriptionRepository(db), invoices, txm, nil, nil)
	svc := NewSystemExtService(repository.NewSystemExtRepository(db), nil)
	svc.SetInvoiceService(invoices)
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})

	r1, err := svc.RefundOnlineTxn(ctx, &v1.RefundOnlineTxnReq{ID: txnID, Amount: 100000, Reason: "IT hoàn lần 1"})
	require.NoError(t, err)
	r2, err := svc.RefundOnlineTxn(ctx, &v1.RefundOnlineTxnReq{ID: txnID, Reason: "IT hoàn phần còn lại"})
	require.NoError(t, err)
	assert.NotEqual(t, r1.GatewayRef, r2.GatewayRef)

	var invStatus, subStatus string
	require.NoError(t, db.Raw(`SELECT status FROM invoices WHERE id = ?`, invID).Scan(&invStatus).Error)
	require.NoError(t, db.Raw(`SELECT status FROM subscriptions WHERE id = ?`, subID).Scan(&subStatus).Error)
	assert.Equal(t, "REFUNDED", invStatus)
	assert.Equal(t, "CANCELLED", subStatus, "RefundHandler của gói tập phải chạy khi hoàn VNPay hết")
}
