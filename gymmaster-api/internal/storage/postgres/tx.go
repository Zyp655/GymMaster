package postgres

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

// TxManager chạy một nhóm thao tác của nhiều repository trong cùng một transaction.
// Transaction được truyền qua context, repository lấy kết nối bằng Conn(ctx, db).
//
//	err := tx.WithTx(ctx, func(ctx context.Context) error {
//	    if err := invoiceRepo.Create(ctx, inv); err != nil { return err }
//	    return subscriptionRepo.Create(ctx, sub)
//	})
type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type txManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) TxManager {
	return &txManager{db: db}
}

func (m *txManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	// Đã ở trong transaction thì dùng lại (transaction lồng nhau).
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// Conn trả transaction trong ctx nếu có, ngược lại trả db gốc.
func Conn(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}
