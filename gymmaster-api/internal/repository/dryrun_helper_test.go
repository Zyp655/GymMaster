package repository

import (
	"fmt"
	"testing"

	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newDryRunDB: *gorm.DB không kết nối thật (DryRun) – dùng để kiểm tra câu SQL do repository sinh ra.
func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(pgdriver.New(pgdriver.Config{DSN: "host=127.0.0.1 port=1 user=x dbname=x sslmode=disable"}),
		&gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("open dry-run db: %v", err)
	}
	return db
}

// captureSQL ghi lại câu SQL cuối cùng của mọi thao tác (query/update/create/delete/raw).
func captureSQL(t *testing.T, db *gorm.DB) *[]string {
	t.Helper()
	out := &[]string{}
	fn := func(d *gorm.DB) {
		*out = append(*out, d.Statement.SQL.String()+fmt.Sprintf(" -- vars=%v", d.Statement.Vars))
	}
	_ = db.Callback().Query().After("gorm:query").Register("test:capture_q", fn)
	_ = db.Callback().Update().After("gorm:update").Register("test:capture_u", fn)
	_ = db.Callback().Create().After("gorm:create").Register("test:capture_c", fn)
	_ = db.Callback().Delete().After("gorm:delete").Register("test:capture_d", fn)
	_ = db.Callback().Raw().After("gorm:raw").Register("test:capture_r", fn)
	_ = db.Callback().Row().After("gorm:row").Register("test:capture_row", fn)
	return out
}
