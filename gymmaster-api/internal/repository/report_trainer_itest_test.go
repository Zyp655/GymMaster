//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Lớp có người dạy thay: "số lớp đã dạy" (TrainerPerformance) và HLV trong thống kê lớp tính cho HLV THỰC DẠY
// (COALESCE(substitute_trainer_id, trainer_id)), không tính cho HLV được phân công.
func TestReportTrainerStats_UseActualTrainer_IT(t *testing.T) {
	db := openITDB(t)
	sfx := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	var branchID, roomID, typeID, assigned, substitute, schedID uint
	must := func(dest *uint, sql string, args ...any) {
		t.Helper()
		if err := db.Raw(sql, args...).Scan(dest).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, sql)
		}
	}
	must(&branchID, `INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '00:00', '23:59') RETURNING id`, "ITTR"+sfx, "IT HLV "+sfx)
	must(&roomID, `INSERT INTO rooms (branch_id, name, capacity) VALUES (?, 'IT phòng', 20) RETURNING id`, branchID)
	must(&typeID, `INSERT INTO class_types (code, name) VALUES (?, 'IT Yoga') RETURNING id`, "ITTR"+sfx)
	must(&assigned, `INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'IT HLV gốc') RETURNING id`, "ittr-a"+sfx+"@it.local")
	must(&substitute, `INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'IT HLV thay') RETURNING id`, "ittr-s"+sfx+"@it.local")
	must(&schedID, `INSERT INTO class_schedules (class_type_id, branch_id, room_id, trainer_id, substitute_trainer_id, start_at, end_at, capacity)
		VALUES (?, ?, ?, ?, ?, now() - interval '3 hour', now() - interval '2 hour', 10) RETURNING id`, typeID, branchID, roomID, assigned, substitute)
	t.Cleanup(func() {
		db.Exec(`DELETE FROM class_schedules WHERE id = ?`, schedID)
		db.Exec(`DELETE FROM class_types WHERE id = ?`, typeID)
		db.Exec(`DELETE FROM rooms WHERE id = ?`, roomID)
		db.Exec(`DELETE FROM users WHERE id IN ?`, []uint{assigned, substitute})
		db.Exec(`DELETE FROM branches WHERE id = ?`, branchID)
	})

	now := time.Now()
	sc := ReportScope{BranchIDs: []uint{branchID}, Start: now.Add(-24 * time.Hour), End: now.Add(time.Hour)}
	repo := NewReportRepository(db)
	rows, err := repo.TrainerPerformance(context.Background(), sc, now)
	if err != nil {
		t.Fatal(err)
	}
	taught := map[uint]int64{}
	for _, r := range rows {
		taught[r.TrainerID] = int64(r.ClassesTaught)
	}
	if taught[substitute] != 1 || taught[assigned] != 0 {
		t.Fatalf("classes_taught: thay=%d gốc=%d, muốn 1/0", taught[substitute], taught[assigned])
	}
	cls, err := repo.ClassScheduleStats(context.Background(), sc)
	if err != nil {
		t.Fatal(err)
	}
	if len(cls) != 1 || cls[0].TrainerID != substitute {
		t.Fatalf("ClassScheduleStats trainer = %+v, muốn HLV dạy thay %d", cls, substitute)
	}
}
