package dberr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"gymmaster-api/internal/pkg/rescode"
)

func TestTranslate(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code int
		msg  string
	}{
		{"unique via gorm translate + gerror wrap",
			gerror.Wrap(fmt.Errorf("%w: %w", gorm.ErrDuplicatedKey, &pgconn.PgError{Code: "23505", Detail: "Key (slug)=(abc) already exists."}), "create post"),
			40900, "Dữ liệu đã tồn tại (trùng đường dẫn (slug))"},
		{"unique composite drops scope column",
			&pgconn.PgError{Code: "23505", Detail: "Key (branch_id, lower(code::text))=(1, x) already exists."},
			40900, "Dữ liệu đã tồn tại (trùng mã)"},
		{"fk insert", &pgconn.PgError{Code: "23503", Message: `insert or update on table "x" violates foreign key constraint "x_branch_id_fkey"`, Detail: `Key (branch_id)=(9) is not present in table "branches".`},
			40000, "Dữ liệu tham chiếu không tồn tại (chi nhánh)"},
		{"fk delete", &pgconn.PgError{Code: "23503", Message: `update or delete on table "branches" violates foreign key constraint`, Detail: `Key (id)=(1) is still referenced from table "rooms".`},
			40900, MsgFKInUse},
		{"check", &pgconn.PgError{Code: "23514"}, 40000, MsgCheck},
		{"too long", &pgconn.PgError{Code: "22001"}, 40000, MsgTooLong},
		{"numeric", &pgconn.PgError{Code: "22003"}, 40000, MsgNumericRange},
		{"null byte", &pgconn.PgError{Code: "22021"}, 40000, MsgInvalidChar},
		{"not null", &pgconn.PgError{Code: "23502", ColumnName: "spent_at"}, 40000, "Thiếu giá trị bắt buộc (ngày chi)"},
		{"pgx encode overflow",
			errors.New("failed to encode args[1]: unable to encode 0xffffffffffffffff into binary format for int8 (OID 20): 18446744073709551615 is greater than maximum value for int8"),
			40000, MsgNumericRange},
		{"record not found", gerror.Wrap(gorm.ErrRecordNotFound, "x"), 40400, MsgNotFound},
	}
	for _, c := range cases {
		got, ok := Translate(c.err)
		if !ok || got.Code.Code() != c.code || got.Message != c.msg {
			t.Errorf("%s: got (%v,%d,%q) want (%d,%q)", c.name, ok, got.Code.Code(), got.Message, c.code, c.msg)
		}
	}
	for _, e := range []error{errors.New("boom"), &pgconn.PgError{Code: "22012"}, &pgconn.PgError{Code: "42P01"},
		errors.New("unable to encode x: cannot find encode plan")} {
		if _, ok := Translate(e); ok {
			t.Errorf("%v must stay 500", e)
		}
	}
	_ = rescode.Conflict
}
