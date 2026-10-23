package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"gymmaster-api/internal/entity"
)

// user_mfa: thiết lập lại MFA (struct CreatedAt rỗng) không được ghi đè created_at = 0001-01-01.
func TestSaveUserMFA_DoesNotOverwriteCreatedAt(t *testing.T) {
	db := newDryRunDB(t)
	sqls := captureSQL(t, db)
	repo := NewPrivacyRepository(db)
	_ = repo.SaveUserMFA(context.Background(), &entity.UserMFA{UserID: 7, TOTPSecretEnc: "x"})
	joined := strings.Join(*sqls, "\n")
	if !strings.Contains(joined, `INSERT INTO "user_mfa"`) || !strings.Contains(joined, `ON CONFLICT ("user_id") DO UPDATE SET`) {
		t.Fatalf("phải upsert theo user_id: %s", joined)
	}
	set := joined[strings.Index(joined, "DO UPDATE SET"):]
	if strings.Contains(set, "created_at") {
		t.Fatalf("khi trùng không được cập nhật created_at: %s", joined)
	}
	if strings.Contains(joined, "0001-01-01") {
		t.Fatalf("không được ghi thời điểm rỗng: %s", joined)
	}
}

// C11: HLV nhận bàn giao chỉ đếm "chưa đọc" từ thời điểm tham gia.
func TestCountUnreadMessages_SinceJoin(t *testing.T) {
	db := newDryRunDB(t)
	sqls := captureSQL(t, db)
	repo := NewCMSRepository(db)
	joined := time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)
	repo.CountUnreadMessages(context.Background(), 1, 2, "TRAINER", &joined)
	repo.CountUnreadMessages(context.Background(), 1, 3, "MEMBER", nil)
	if len(*sqls) != 2 || !strings.Contains((*sqls)[0], "created_at >=") || strings.Contains((*sqls)[1], "created_at >=") {
		t.Fatalf("SQL đếm chưa đọc sai: %v", *sqls)
	}

	// Đổi HLV → ghi trainer_joined_at; chỉ đổi hội viên → giữ nguyên.
	*sqls = (*sqls)[:0]
	at, err := repo.SyncConversationParticipants(context.Background(), &entity.Conversation{ID: 5, MemberID: 1, TrainerID: 10}, 1, 11)
	if err != nil || at == nil || len(*sqls) != 1 || !strings.Contains((*sqls)[0], "trainer_joined_at") {
		t.Fatalf("đổi HLV phải ghi trainer_joined_at: at=%v err=%v sql=%v", at, err, *sqls)
	}
	*sqls = (*sqls)[:0]
	at, _ = repo.SyncConversationParticipants(context.Background(), &entity.Conversation{ID: 5, MemberID: 1, TrainerID: 10, TrainerJoinedAt: &joined}, 2, 10)
	if at == nil || !at.Equal(joined) || strings.Contains((*sqls)[0], "trainer_joined_at") {
		t.Fatalf("không đổi HLV thì giữ trainer_joined_at: at=%v sql=%v", at, *sqls)
	}
}

// Dashboard/Mật độ/Check-in dùng chung định nghĩa "đang trong phòng tập".
func TestPresentDefinitionShared(t *testing.T) {
	now := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)
	if !PresentSince(now).Equal(now.Add(-3 * time.Hour)) {
		t.Fatal("cửa sổ đang trong phòng tập phải là 3 giờ")
	}
	db := newDryRunDB(t)
	sqls := captureSQL(t, db)
	_, _ = NewReportRepository(db).Overview(context.Background(), ReportOverviewParams{Now: now, TodayStart: now.Truncate(24 * time.Hour)})
	joined := strings.Join(*sqls, "\n")
	key := presentPersonKey("c")
	if !strings.Contains(joined, "count(DISTINCT "+key+")") {
		t.Fatalf("overview phải đếm theo người với cùng khóa của Mật độ: %s", joined)
	}
	if !strings.Contains(joined, PresentSince(now).Format("2006-01-02 15:04:05")) {
		t.Fatalf("phải dùng mốc PresentSince(now): %s", joined)
	}
}
