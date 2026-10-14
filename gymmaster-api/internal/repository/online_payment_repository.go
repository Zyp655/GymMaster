package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	OnlineTxnRow struct {
		entity.OnlinePaymentTxn
		InvoiceNo  string
		MemberCode string
		MemberName string
	}

	OnlineTxnListFilter struct {
		BranchIDs []uint
		Status    string
		Search    string
	}

	OnlinePaymentRepository interface {
		Create(ctx context.Context, txn *entity.OnlinePaymentTxn) error
		FindByTxnRef(ctx context.Context, txnRef string, lock bool) (*entity.OnlinePaymentTxn, error)
		UpdateFields(ctx context.Context, id uint, fields map[string]any) error
		List(ctx context.Context, q dto.Query, f OnlineTxnListFilter) ([]OnlineTxnRow, int64, error)
		ExpirePendingBefore(ctx context.Context, before time.Time) (int64, error)

		FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error)
		FindMemberByID(ctx context.Context, id uint) (*entity.Member, error)
		FindPlanByID(ctx context.Context, id uint) (*entity.MembershipPlan, error)
		SetInvoiceSource(ctx context.Context, invoiceID uint, source string) error
	}

	onlinePaymentRepository struct {
		db *gorm.DB
	}
)

func NewOnlinePaymentRepository(db *gorm.DB) OnlinePaymentRepository {
	return &onlinePaymentRepository{db: db}
}

func (r *onlinePaymentRepository) Create(ctx context.Context, txn *entity.OnlinePaymentTxn) error {
	return postgres.Conn(ctx, r.db).Create(txn).Error
}

func (r *onlinePaymentRepository) FindByTxnRef(ctx context.Context, txnRef string, lock bool) (*entity.OnlinePaymentTxn, error) {
	db := postgres.Conn(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var txn entity.OnlinePaymentTxn
	if err := db.Where("txn_ref = ?", txnRef).First(&txn).Error; err != nil {
		return nil, err
	}
	return &txn, nil
}

func (r *onlinePaymentRepository) UpdateFields(ctx context.Context, id uint, fields map[string]any) error {
	return postgres.Conn(ctx, r.db).Model(&entity.OnlinePaymentTxn{}).Where("id = ?", id).Updates(fields).Error
}

const onlineTxnSelect = `t.*, COALESCE(i.invoice_no, '') AS invoice_no,
	COALESCE(m.member_code, '') AS member_code, COALESCE(m.full_name, '') AS member_name`

func (r *onlinePaymentRepository) List(ctx context.Context, q dto.Query, f OnlineTxnListFilter) ([]OnlineTxnRow, int64, error) {
	tx := postgres.Conn(ctx, r.db).Table("online_payment_txns t").
		Joins("JOIN invoices i ON i.id = t.invoice_id").
		Joins("JOIN members m ON m.id = t.member_id")
	if f.BranchIDs != nil {
		tx = tx.Where("i.branch_id IN ?", f.BranchIDs)
	}
	if status := strings.TrimSpace(f.Status); status != "" {
		tx = tx.Where("t.status = ?", status)
	}
	search := strings.TrimSpace(f.Search)
	if search == "" {
		search = q.Keyword
	}
	if search != "" {
		kw := "%" + search + "%"
		tx = tx.Where("t.txn_ref ILIKE ? OR t.provider_txn_no ILIKE ? OR i.invoice_no ILIKE ? OR m.full_name ILIKE ? OR m.member_code ILIKE ?",
			kw, kw, kw, kw, kw)
	}
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []OnlineTxnRow
	if err := tx.Select(onlineTxnSelect).Scopes(q.Paginate).Order("t.id DESC").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *onlinePaymentRepository) ExpirePendingBefore(ctx context.Context, before time.Time) (int64, error) {
	res := postgres.Conn(ctx, r.db).Model(&entity.OnlinePaymentTxn{}).
		Where("status = ? AND created_at < ?", entity.OnlineTxnPending, before).
		Update("status", entity.OnlineTxnExpired)
	return res.RowsAffected, res.Error
}

func (r *onlinePaymentRepository) FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).Where("user_id = ?", userID).Order("id").Take(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *onlinePaymentRepository) FindMemberByID(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *onlinePaymentRepository) FindPlanByID(ctx context.Context, id uint) (*entity.MembershipPlan, error) {
	var p entity.MembershipPlan
	if err := postgres.Conn(ctx, r.db).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *onlinePaymentRepository) SetInvoiceSource(ctx context.Context, invoiceID uint, source string) error {
	return postgres.Conn(ctx, r.db).Model(&entity.Invoice{}).Where("id = ?", invoiceID).Update("source", source).Error
}
