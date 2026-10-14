package service

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"gymmaster-api/internal/pkg/rescode"
)

// mapDuplicate đổi lỗi vi phạm unique của DB (trường hợp 2 request đồng thời lọt qua bước kiểm tra trước) thành 409.
func mapDuplicate(err error, msg string) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return rescode.New(rescode.Conflict, msg)
	}
	return err
}

// isExclusionViolation nhận biết lỗi vi phạm EXCLUDE constraint (SQLSTATE 23P01),
// ví dụ trùng lịch phòng/HLV/ca làm.
func isExclusionViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23P01"
}

// isCheckViolation nhận biết lỗi vi phạm CHECK constraint (SQLSTATE 23514), ví dụ tồn kho âm.
func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}
