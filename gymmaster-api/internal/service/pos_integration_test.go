//go:build integration

package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	invv1 "gymmaster-api/api/inventory/v1"
	posv1 "gymmaster-api/api/pos/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Chạy: GYMMASTER_TEST_DSN=postgres://gymmaster:gymmaster@localhost:5432/gymmaster?sslmode=disable make itest

type posITFixture struct {
	db       *gorm.DB
	ctx      context.Context
	pos      PosService
	stock    StockService
	branchA  uint
	branchB  uint
	products []uint
}

func posITSetup(t *testing.T, nProducts int) *posITFixture {
	t.Helper()
	dsn := os.Getenv("GYMMASTER_TEST_DSN")
	if dsn == "" {
		t.Skip("chưa đặt GYMMASTER_TEST_DSN")
	}
	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(30)

	fx := &posITFixture{db: db}
	suffix := fmt.Sprintf("%06d", time.Now().UnixNano()%1_000_000)
	for i, p := range []*uint{&fx.branchA, &fx.branchB} {
		require.NoError(t, db.Raw(`INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '00:00', '23:59') RETURNING id`,
			fmt.Sprintf("IP%d%s", i, suffix), fmt.Sprintf("IT POS %d %s", i, suffix)).Scan(p).Error)
	}
	for i := 0; i < nProducts; i++ {
		var id uint
		// min_stock = 0 → không sinh thông báo tồn thấp trong test.
		require.NoError(t, db.Raw(`INSERT INTO products (sku, name, unit, cost_price, sell_price, min_stock)
			VALUES (?, ?, 'chai', 5000, 10000, 0) RETURNING id`, fmt.Sprintf("ITPOS%s%d", suffix, i), "IT POS sản phẩm "+suffix).Scan(&id).Error)
		fx.products = append(fx.products, id)
	}
	var adminID uint
	db.Raw(`SELECT id FROM users ORDER BY id LIMIT 1`).Scan(&adminID)
	fx.ctx = reqctx.WithAuth(context.Background(), &token.Payload{UserID: adminID, Roles: []string{consts.RoleOwner}})

	tx := postgres.NewTxManager(db)
	audit := NewAuditService(db)
	promos := NewPromotionService(repository.NewPromotionRepository(db), audit)
	invoices := NewInvoiceService(repository.NewInvoiceRepository(db), promos, tx, audit)
	productRepo, stockRepo := repository.NewProductRepository(db), repository.NewStockRepository(db)
	fx.pos = NewPosService(repository.NewPosRepository(db), productRepo, stockRepo, invoices, tx, audit, NewNotifier(db))
	fx.stock = NewStockService(stockRepo, productRepo, tx, audit, NewNotifier(db))

	t.Cleanup(func() {
		branches := []uint{fx.branchA, fx.branchB}
		invQ := `SELECT id FROM invoices WHERE branch_id IN ?`
		db.Exec(`DELETE FROM audit_logs WHERE (entity IN ('invoice', 'pos_order') AND entity_id IN (`+invQ+`))
			OR (entity = 'stock_voucher' AND after->>'voucher_no' IN (SELECT voucher_no FROM stock_transactions WHERE product_id IN ?))`,
			branches, fx.products)
		for _, q := range []string{
			`DELETE FROM stock_transactions WHERE product_id IN ?`,
			`DELETE FROM stock_levels WHERE product_id IN ?`,
		} {
			db.Exec(q, fx.products)
		}
		for _, q := range []string{
			`DELETE FROM payments WHERE invoice_id IN (` + invQ + `)`,
			`DELETE FROM refunds WHERE invoice_id IN (` + invQ + `)`,
			`DELETE FROM invoices WHERE branch_id IN ?`,
		} {
			db.Exec(q, branches)
		}
		db.Exec(`DELETE FROM products WHERE id IN ?`, fx.products)
		db.Exec(`DELETE FROM branches WHERE id IN ?`, branches)
	})
	return fx
}

func (fx *posITFixture) importStock(t *testing.T, branchID uint, qty int) {
	items := make([]*invv1.ImportItem, 0, len(fx.products))
	for _, id := range fx.products {
		items = append(items, &invv1.ImportItem{ProductID: id, Quantity: qty, UnitCost: 5000})
	}
	_, err := fx.stock.Import(fx.ctx, &invv1.ImportReq{BranchID: branchID, Items: items, Note: "IT"})
	require.NoError(t, err)
}

// 20 đơn POS đồng thời cùng mua 1 sản phẩm tồn 10 → đúng 10 đơn thành công, tồn 0, không âm, 10 dòng SALE.
func TestPosIT_ConcurrentSalesNoOversell(t *testing.T) {
	fx := posITSetup(t, 1)
	fx.importStock(t, fx.branchA, 10)
	pid := fx.products[0]

	const n = 20
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok, conflict := 0, 0
	var others []error
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := fx.pos.CreateOrder(fx.ctx, &posv1.CreateOrderReq{BranchID: fx.branchA,
				Items:    []*posv1.OrderItem{{ProductID: pid, Quantity: 1}},
				Payments: []*posv1.OrderPayment{{Amount: 20000, Method: entity.PaymentCash}}})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case gerror.Code(err) == rescode.Conflict:
				conflict++
			default:
				others = append(others, err)
			}
		}()
	}
	close(start)
	wg.Wait()

	require.Empty(t, others, "lỗi ngoài dự kiến (deadlock/khác)")
	assert.Equal(t, 10, ok)
	assert.Equal(t, 10, conflict)
	var qty int
	require.NoError(t, fx.db.Raw(`SELECT quantity FROM stock_levels WHERE product_id = ? AND branch_id = ?`, pid, fx.branchA).Scan(&qty).Error)
	assert.Equal(t, 0, qty)
	var sales, negative int64
	fx.db.Raw(`SELECT COUNT(*) FROM stock_transactions WHERE product_id = ? AND type = 'SALE'`, pid).Scan(&sales)
	fx.db.Raw(`SELECT COUNT(*) FROM stock_transactions WHERE product_id = ? AND balance_after < 0`, pid).Scan(&negative)
	assert.Equal(t, int64(10), sales)
	assert.Equal(t, int64(0), negative)
	var invoices int64
	fx.db.Raw(`SELECT COUNT(*) FROM invoices WHERE branch_id = ? AND source = 'POS' AND status = 'PAID' AND paid_amount = total`, fx.branchA).Scan(&invoices)
	assert.Equal(t, int64(10), invoices, "đơn lỗi không để lại hóa đơn")
}

// Điều chuyển 2 chiều đồng thời (A→B và B→A, nhiều sản phẩm, thứ tự dòng ngược nhau) không deadlock và bảo toàn tổng tồn.
func TestPosIT_ConcurrentTransfersNoDeadlock(t *testing.T) {
	fx := posITSetup(t, 3)
	fx.importStock(t, fx.branchA, 100)
	fx.importStock(t, fx.branchB, 100)

	const n = 30
	var wg sync.WaitGroup
	errs := make(chan error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			from, to := fx.branchA, fx.branchB
			items := []*invv1.TransferItem{}
			for _, id := range fx.products {
				items = append(items, &invv1.TransferItem{ProductID: id, Quantity: 1})
			}
			if i%2 == 1 {
				from, to = to, from
				for l, r := 0, len(items)-1; l < r; l, r = l+1, r-1 {
					items[l], items[r] = items[r], items[l]
				}
			}
			_, err := fx.stock.Transfer(fx.ctx, &invv1.TransferReq{FromBranchID: from, ToBranchID: to, Items: items})
			errs <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	for _, id := range fx.products {
		var a, b int
		fx.db.Raw(`SELECT quantity FROM stock_levels WHERE product_id = ? AND branch_id = ?`, id, fx.branchA).Scan(&a)
		fx.db.Raw(`SELECT quantity FROM stock_levels WHERE product_id = ? AND branch_id = ?`, id, fx.branchB).Scan(&b)
		assert.Equal(t, 100, a, "15 đi − 15 về")
		assert.Equal(t, 100, b)
	}
	var lines int64
	fx.db.Raw(`SELECT COUNT(*) FROM stock_transactions WHERE product_id IN ? AND type IN ('TRANSFER_IN', 'TRANSFER_OUT')`, fx.products).Scan(&lines)
	assert.Equal(t, int64(n*2*len(fx.products)), lines)
}

// Luồng đầy đủ trên DB thật: nhập (giá vốn bình quân) → bán → trả hàng một phần → thẻ kho, phiếu.
func TestPosIT_SaleAndReturnFlow(t *testing.T) {
	fx := posITSetup(t, 1)
	pid := fx.products[0]
	fx.importStock(t, fx.branchA, 10)                                        // 10 × 5.000 (giá vốn đang 5.000)
	_, err := fx.stock.Import(fx.ctx, &invv1.ImportReq{BranchID: fx.branchB, // tổng tồn 10 → +10 × 8.000 → 6.500
		Items: []*invv1.ImportItem{{ProductID: pid, Quantity: 10, UnitCost: 8000}}})
	require.NoError(t, err)
	var cost float64
	fx.db.Raw(`SELECT cost_price FROM products WHERE id = ?`, pid).Scan(&cost)
	assert.Equal(t, 6500.0, cost)

	res, err := fx.pos.CreateOrder(fx.ctx, &posv1.CreateOrderReq{BranchID: fx.branchA,
		Items:    []*posv1.OrderItem{{ProductID: pid, Quantity: 4}},
		Payments: []*posv1.OrderPayment{{Amount: 10000, Method: entity.PaymentTransfer}, {Amount: 50000, Method: entity.PaymentCash}}})
	require.NoError(t, err)
	assert.Equal(t, 40000.0, res.Invoice.Total)
	assert.Equal(t, 20000.0, res.ChangeAmount)
	assert.Equal(t, entity.InvoiceStatusPaid, res.Invoice.Status)

	ret, err := fx.pos.Return(fx.ctx, &posv1.ReturnReq{InvoiceID: res.Invoice.ID, RefundMethod: entity.PaymentCash, Reason: "IT",
		Items: []*posv1.ReturnItem{{ProductID: pid, Quantity: 1}}})
	require.NoError(t, err)
	assert.Equal(t, 10000.0, ret.RefundAmount)
	_, err = fx.pos.Return(fx.ctx, &posv1.ReturnReq{InvoiceID: res.Invoice.ID, RefundMethod: entity.PaymentCash, Reason: "IT",
		Items: []*posv1.ReturnItem{{ProductID: pid, Quantity: 4}}})
	require.Error(t, err)
	var qty int
	fx.db.Raw(`SELECT quantity FROM stock_levels WHERE product_id = ? AND branch_id = ?`, pid, fx.branchA).Scan(&qty)
	assert.Equal(t, 7, qty)

	v, err := fx.stock.Voucher(fx.ctx, ret.VoucherNo)
	require.NoError(t, err)
	assert.Equal(t, entity.StockSaleReturn, v.Kind)
	assert.Equal(t, 6500.0, v.Lines[0].UnitCost)
	_, err = fx.stock.Voucher(fx.ctx, "KHONGCO")
	assert.Equal(t, rescode.NotFound, gerror.Code(err))
}
