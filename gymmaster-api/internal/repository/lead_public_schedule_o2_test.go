package repository

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Lịch lớp công khai (đăng ký học thử) hiển thị HLV thực dạy: COALESCE(substitute_trainer_id, trainer_id).
func TestListPublicSchedules_UsesActualTrainer(t *testing.T) {
	db := newDryRunDB(t)
	sqls := captureSQL(t, db)
	_, _ = NewLeadRepository(db).ListPublicSchedules(context.Background(), 1, time.Now(), time.Now().Add(time.Hour))
	if len(*sqls) == 0 {
		t.Fatal("không bắt được câu SQL")
	}
	q := (*sqls)[len(*sqls)-1]
	if !strings.Contains(q, "JOIN users u ON u.id = COALESCE(cs.substitute_trainer_id, cs.trainer_id)") ||
		!strings.Contains(q, "COALESCE(cs.substitute_trainer_id, cs.trainer_id) AS trainer_id") {
		t.Fatalf("SQL chưa dùng HLV thực dạy: %s", q)
	}
}
