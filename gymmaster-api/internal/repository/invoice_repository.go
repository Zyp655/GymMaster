package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	// InvoiceRow là hóa đơn kèm thông tin hiển thị (hội viên, chi nhánh, người lập, mã KM).
	InvoiceRow struct {
		entity.Invoice
		MemberCode    string
		MemberName    string
		MemberPhone   string
		BranchName    string
		IssuedByName  string
		VoidedByName  string
		PromotionCode string
	}

	// InvoiceListFilter là bộ lọc danh sách hóa đơn đã chuẩn hóa ở service.
	InvoiceListFilter struct {
		BranchIDs []uint // nil = không giới hạn
		MemberID  uint
		Status    string
		Source    string
		From      *time.Time // issued_at >= From
		To        *time.Time // issued_at < To
	}

	// PaymentRow là một lần thu kèm hóa đơn/hội viên/người thu (sổ thu).
	PaymentRow struct {
		entity.Payment
		InvoiceNo      string
		BranchID       uint
		BranchName     string
		MemberID       *uint
		MemberCode     string
		MemberName     string
		ReceivedByName string
	}

	RefundRow struct {
		entity.Refund
		RefundedByName string
	}

	PaymentLedgerFilter struct {
		BranchIDs  []uint
		Method     string
		From       *time.Time
		To         *time.Time
		ReceivedBy uint
		InvoiceID  uint
	}

	PaymentMethodTotal struct {
		Method string
		Amount float64
		Count  int64
	}

	// InvoiceConsumedSub: gói tập của hóa đơn mà giá trị đã trả đã được khấu trừ sang gói khác.
	InvoiceConsumedSub struct {
		SubscriptionID   uint
		Status           string
		EventType        string // UPGRADED | TRANSFERRED
		ToSubscriptionID uint
	}

	InvoiceRepository interface {
		// Create ghi hóa đơn và các dòng (inv.Items), gán InvoiceID cho từng dòng.
		Create(ctx context.Context, inv *entity.Invoice) error
		// FindByID đọc hóa đơn; lock = true → SELECT ... FOR UPDATE (phải ở trong transaction).
		FindByID(ctx context.Context, id uint, lock bool) (*entity.Invoice, error)
		FindRow(ctx context.Context, id uint) (*InvoiceRow, error)
		Items(ctx context.Context, invoiceID uint) ([]entity.InvoiceItem, error)
		UpdateFields(ctx context.Context, id uint, fields map[string]any) error
		CreatePayment(ctx context.Context, p *entity.Payment) error
		CreateRefund(ctx context.Context, r *entity.Refund) error
		Payments(ctx context.Context, invoiceID uint) ([]PaymentRow, error)
		Refunds(ctx context.Context, invoiceID uint) ([]RefundRow, error)
		List(ctx context.Context, q dto.Query, f InvoiceListFilter) ([]InvoiceRow, int64, error)
		ListPayments(ctx context.Context, q dto.Query, f PaymentLedgerFilter) ([]PaymentRow, int64, error)
		SumPayments(ctx context.Context, f PaymentLedgerFilter) ([]PaymentMethodTotal, error)
		// ReleasePromotion trả lại 1 lượt dùng mã khuyến mãi (khi hủy hóa đơn).
		ReleasePromotion(ctx context.Context, promotionID uint) error
		FindBranch(ctx context.Context, id uint) (*entity.Branch, error)
		FindMember(ctx context.Context, id uint) (*entity.Member, error)
		UserName(ctx context.Context, id uint) (string, error)

		// InstallmentSchedules: các kỳ trả góp của hóa đơn (theo seq); lock = FOR UPDATE.
		InstallmentSchedules(ctx context.Context, invoiceID uint, lock bool) ([]entity.InstallmentSchedule, error)
		// SaveInstallmentSchedule ghi amount/paid_amount/status/paid_at/updated_at của một kỳ.
		SaveInstallmentSchedule(ctx context.Context, sc *entity.InstallmentSchedule) error
		// PendingOnlineTxn: giao dịch VNPay PENDING mới nhất của hóa đơn tạo từ thời điểm since (chưa hết hạn); nil nếu không có.
		PendingOnlineTxn(ctx context.Context, invoiceID uint, since time.Time) (*entity.OnlinePaymentTxn, error)
		// ConsumedSubscriptions: các gói tập của hóa đơn đã bị nâng cấp (CANCELLED) / chuyển nhượng (TRANSFERRED) và
		// chưa được khôi phục – giá trị đã trả đã được khấu trừ sang gói khác. Bỏ qua gói nhận chuyển nhượng
		// (carried_paid_base) vì hóa đơn của nó chỉ là phí chuyển nhượng, không phải gốc khấu trừ.
		ConsumedSubscriptions(ctx context.Context, invoiceID uint) ([]InvoiceConsumedSub, error)
	}

	invoiceRepository struct {
		db *gorm.DB
	}
)

func NewInvoiceRepository(db *gorm.DB) InvoiceRepository {
	return &invoiceRepository{db: db}
}

const invoiceRowSelect = `i.*,
	COALESCE(m.member_code, '') AS member_code, COALESCE(m.full_name, '') AS member_name, COALESCE(m.phone, '') AS member_phone,
	b.name AS branch_name, COALESCE(u.full_name, '') AS issued_by_name, COALESCE(vu.full_name, '') AS voided_by_name,
	COALESCE(p.code, '') AS promotion_code`

func (r *invoiceRepository) rowQuery(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("invoices i").
		Joins("LEFT JOIN members m ON m.id = i.member_id").
		Joins("JOIN branches b ON b.id = i.branch_id").
		Joins("LEFT JOIN users u ON u.id = i.issued_by").
		Joins("LEFT JOIN users vu ON vu.id = i.voided_by").
		Joins("LEFT JOIN promotions p ON p.id = i.promotion_id")
}

func (r *invoiceRepository) Create(ctx context.Context, inv *entity.Invoice) error {
	db := postgres.Conn(ctx, r.db)
	if err := db.Create(inv).Error; err != nil {
		return err
	}
	if len(inv.Items) == 0 {
		return nil
	}
	for i := range inv.Items {
		inv.Items[i].InvoiceID = inv.ID
	}
	return db.Create(&inv.Items).Error
}

func (r *invoiceRepository) FindByID(ctx context.Context, id uint, lock bool) (*entity.Invoice, error) {
	db := postgres.Conn(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var inv entity.Invoice
	if err := db.First(&inv, id).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *invoiceRepository) FindRow(ctx context.Context, id uint) (*InvoiceRow, error) {
	var rows []InvoiceRow
	if err := r.rowQuery(ctx).Select(invoiceRowSelect).Where("i.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *invoiceRepository) Items(ctx context.Context, invoiceID uint) ([]entity.InvoiceItem, error) {
	var items []entity.InvoiceItem
	err := postgres.Conn(ctx, r.db).Where("invoice_id = ?", invoiceID).Order("id").Find(&items).Error
	return items, err
}

func (r *invoiceRepository) UpdateFields(ctx context.Context, id uint, fields map[string]any) error {
	return postgres.Conn(ctx, r.db).Model(&entity.Invoice{}).Where("id = ?", id).Updates(fields).Error
}

func (r *invoiceRepository) CreatePayment(ctx context.Context, p *entity.Payment) error {
	return postgres.Conn(ctx, r.db).Create(p).Error
}

func (r *invoiceRepository) CreateRefund(ctx context.Context, rf *entity.Refund) error {
	return postgres.Conn(ctx, r.db).Create(rf).Error
}

const paymentRowSelect = `pm.*, i.invoice_no, i.branch_id, b.name AS branch_name, i.member_id,
	COALESCE(m.member_code, '') AS member_code, COALESCE(m.full_name, '') AS member_name,
	COALESCE(u.full_name, '') AS received_by_name`

func (r *invoiceRepository) paymentQuery(ctx context.Context, f PaymentLedgerFilter) *gorm.DB {
	tx := postgres.Conn(ctx, r.db).Table("payments pm").
		Joins("JOIN invoices i ON i.id = pm.invoice_id").
		Joins("JOIN branches b ON b.id = i.branch_id").
		Joins("LEFT JOIN members m ON m.id = i.member_id").
		Joins("LEFT JOIN users u ON u.id = pm.received_by")
	if f.BranchIDs != nil {
		tx = tx.Where("i.branch_id IN ?", f.BranchIDs)
	}
	if f.InvoiceID > 0 {
		tx = tx.Where("pm.invoice_id = ?", f.InvoiceID)
	}
	if f.Method != "" {
		tx = tx.Where("pm.method = ?", f.Method)
	}
	if f.From != nil {
		tx = tx.Where("pm.paid_at >= ?", *f.From)
	}
	if f.To != nil {
		tx = tx.Where("pm.paid_at < ?", *f.To)
	}
	if f.ReceivedBy > 0 {
		tx = tx.Where("pm.received_by = ?", f.ReceivedBy)
	}
	return tx
}

func (r *invoiceRepository) Payments(ctx context.Context, invoiceID uint) ([]PaymentRow, error) {
	var rows []PaymentRow
	err := r.paymentQuery(ctx, PaymentLedgerFilter{InvoiceID: invoiceID}).Select(paymentRowSelect).
		Order("pm.paid_at, pm.id").Scan(&rows).Error
	return rows, err
}

func (r *invoiceRepository) Refunds(ctx context.Context, invoiceID uint) ([]RefundRow, error) {
	var rows []RefundRow
	err := postgres.Conn(ctx, r.db).Table("refunds rf").
		Select("rf.*, COALESCE(u.full_name, '') AS refunded_by_name").
		Joins("LEFT JOIN users u ON u.id = rf.refunded_by").
		Where("rf.invoice_id = ?", invoiceID).Order("rf.id").Scan(&rows).Error
	return rows, err
}

func (r *invoiceRepository) List(ctx context.Context, q dto.Query, f InvoiceListFilter) ([]InvoiceRow, int64, error) {
	tx := r.rowQuery(ctx)
	if f.BranchIDs != nil {
		tx = tx.Where("i.branch_id IN ?", f.BranchIDs)
	}
	if f.MemberID > 0 {
		tx = tx.Where("i.member_id = ?", f.MemberID)
	}
	if f.Status != "" {
		tx = tx.Where("i.status = ?", f.Status)
	}
	if f.Source != "" {
		tx = tx.Where("i.source = ?", f.Source)
	}
	if f.From != nil {
		tx = tx.Where("i.issued_at >= ?", *f.From)
	}
	if f.To != nil {
		tx = tx.Where("i.issued_at < ?", *f.To)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("i.invoice_no ILIKE ? OR m.full_name ILIKE ? OR m.phone LIKE ? OR m.member_code ILIKE ?", kw, kw, kw, kw)
	}
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []InvoiceRow
	if err := tx.Select(invoiceRowSelect).Scopes(q.Paginate).Order("i.id DESC").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *invoiceRepository) ListPayments(ctx context.Context, q dto.Query, f PaymentLedgerFilter) ([]PaymentRow, int64, error) {
	tx := r.paymentQuery(ctx, f)
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("i.invoice_no ILIKE ? OR m.full_name ILIKE ? OR m.phone LIKE ? OR pm.reference ILIKE ?", kw, kw, kw, kw)
	}
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []PaymentRow
	if err := tx.Select(paymentRowSelect).Scopes(q.Paginate).Order("pm.id DESC").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *invoiceRepository) SumPayments(ctx context.Context, f PaymentLedgerFilter) ([]PaymentMethodTotal, error) {
	var rows []PaymentMethodTotal
	err := r.paymentQuery(ctx, f).
		Select("pm.method AS method, COALESCE(SUM(pm.amount), 0) AS amount, COUNT(*) AS count").
		Group("pm.method").Scan(&rows).Error
	return rows, err
}

func (r *invoiceRepository) ReleasePromotion(ctx context.Context, promotionID uint) error {
	return postgres.Conn(ctx, r.db).Exec(
		`UPDATE promotions SET used_count = used_count - 1 WHERE id = ? AND used_count > 0`, promotionID).Error
}

func (r *invoiceRepository) FindBranch(ctx context.Context, id uint) (*entity.Branch, error) {
	var b entity.Branch
	if err := postgres.Conn(ctx, r.db).Unscoped().First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *invoiceRepository) FindMember(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).Unscoped().First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *invoiceRepository) UserName(ctx context.Context, id uint) (string, error) {
	var names []string
	err := postgres.Conn(ctx, r.db).Table("users").Where("id = ?", id).Limit(1).Pluck("full_name", &names).Error
	if err != nil || len(names) == 0 {
		return "", err
	}
	return names[0], nil
}

func (r *invoiceRepository) InstallmentSchedules(ctx context.Context, invoiceID uint, lock bool) ([]entity.InstallmentSchedule, error) {
	db := postgres.Conn(ctx, r.db).Table("installment_schedules s").Select("s.*").
		Joins("JOIN installment_plans ip ON ip.id = s.plan_id").
		Where("ip.invoice_id = ?", invoiceID).Order("s.seq")
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "s"}})
	}
	var out []entity.InstallmentSchedule
	err := db.Find(&out).Error
	return out, err
}

func (r *invoiceRepository) SaveInstallmentSchedule(ctx context.Context, sc *entity.InstallmentSchedule) error {
	return postgres.Conn(ctx, r.db).Model(&entity.InstallmentSchedule{}).Where("id = ?", sc.ID).Updates(map[string]any{
		"amount": sc.Amount, "paid_amount": sc.PaidAmount, "status": sc.Status, "paid_at": sc.PaidAt, "updated_at": sc.UpdatedAt,
	}).Error
}

func (r *invoiceRepository) PendingOnlineTxn(ctx context.Context, invoiceID uint, since time.Time) (*entity.OnlinePaymentTxn, error) {
	var rows []entity.OnlinePaymentTxn
	err := postgres.Conn(ctx, r.db).Where("invoice_id = ? AND status = ? AND created_at >= ?", invoiceID, entity.OnlineTxnPending, since).
		Order("id DESC").Limit(1).Find(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

func (r *invoiceRepository) ConsumedSubscriptions(ctx context.Context, invoiceID uint) ([]InvoiceConsumedSub, error) {
	var rows []InvoiceConsumedSub
	// Sự kiện quyết định cuối cùng của gói: UPGRADED/TRANSFERRED (đã khấu trừ sang gói khác) hay ACTIVATED/EXPIRED/
	// CANCELLED (gói được khôi phục khi gói mới bị hủy, hoặc hủy thông thường).
	err := postgres.Conn(ctx, r.db).Raw(`SELECT s.id AS subscription_id, s.status, e.event_type,
	COALESCE(NULLIF(e.data->>'to_subscription_id', '')::bigint, 0) AS to_subscription_id
FROM subscriptions s
JOIN LATERAL (
	SELECT ev.event_type, ev.data FROM subscription_events ev
	WHERE ev.subscription_id = s.id AND ev.event_type IN (?)
	ORDER BY ev.id DESC LIMIT 1
) e ON TRUE
WHERE s.invoice_id = ? AND s.carried_paid_base IS NULL AND s.status IN (?) AND e.event_type IN (?)
ORDER BY s.id`,
		[]string{entity.SubEventUpgraded, entity.SubEventTransferred, entity.SubEventActivated, entity.SubEventExpired, entity.SubEventCancelled},
		invoiceID,
		[]string{entity.SubStatusCancelled, entity.SubStatusTransferred},
		[]string{entity.SubEventUpgraded, entity.SubEventTransferred}).Scan(&rows).Error
	return rows, err
}
