//go:build integration

package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	cmsv1 "gymmaster-api/api/cms/v1"
	cpv1 "gymmaster-api/api/contractpass/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Chạy: GYMMASTER_TEST_DSN=postgres://gymmaster:gymmaster@localhost:5432/<db>?sslmode=disable go test -tags integration -run QA5RB ./internal/service
// (cần migration 000048). Hồi quy QA vòng 5 nhóm R-B.

func qa5rbDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("GYMMASTER_TEST_DSN")
	if dsn == "" {
		t.Skip("chưa đặt GYMMASTER_TEST_DSN")
	}
	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(20)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func qa5rbID(t *testing.T, db *gorm.DB, sql string, args ...any) uint {
	t.Helper()
	var id uint
	require.NoError(t, db.Raw(sql, args...).Scan(&id).Error, sql)
	return id
}

// Việc 1: hai người mời KHÁC NHAU tạo vé mời đồng thời cho cùng SĐT + cùng ngày → đúng một vé, lệnh kia 409.
func TestQA5RBIT_GuestPassConcurrentSamePhone(t *testing.T) {
	db := qa5rbDB(t)
	sfx := fmt.Sprintf("%06d", time.Now().UnixNano()%1_000_000)
	branchID := qa5rbID(t, db, `INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '00:00', '23:59') RETURNING id`, "Q5G"+sfx, "QA5-RB-CN-"+sfx)
	planID := qa5rbID(t, db, `INSERT INTO membership_plans (code, name, type, duration_days, price, access_scope, guest_passes_per_month)
		VALUES (?, 'QA5-RB gói mời', 'TIME', 90, 0, 'ALL_BRANCHES', 30) RETURNING id`, "Q5G"+sfx)
	const sponsors = 2
	var userIDs, memberIDs []uint
	for i := range sponsors {
		uid := qa5rbID(t, db, `INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'QA5-RB HV') RETURNING id`,
			fmt.Sprintf("qa5rb-g%d-%s@it.local", i, sfx))
		mid := qa5rbID(t, db, `INSERT INTO members (user_id, full_name, phone, home_branch_id) VALUES (?, 'QA5-RB HV', ?, ?) RETURNING id`,
			uid, fmt.Sprintf("070%s%d", sfx, i), branchID)
		require.NoError(t, db.Exec(`INSERT INTO subscriptions (member_id, plan_id, branch_id, start_date, end_date, price, status)
			VALUES (?, ?, ?, CURRENT_DATE - 1, CURRENT_DATE + 60, 0, 'ACTIVE')`, mid, planID, branchID).Error)
		userIDs, memberIDs = append(userIDs, uid), append(memberIDs, mid)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM day_passes WHERE branch_id = ?`, branchID)
		db.Exec(`DELETE FROM subscriptions WHERE branch_id = ?`, branchID)
		db.Exec(`DELETE FROM members WHERE id IN ?`, memberIDs)
		db.Exec(`DELETE FROM users WHERE id IN ?`, userIDs)
		db.Exec(`DELETE FROM membership_plans WHERE id = ?`, planID)
		db.Exec(`DELETE FROM branches WHERE id = ?`, branchID)
	})
	svc := NewContractPassService(repository.NewContractPassRepository(db), postgres.NewTxManager(db), nil)

	guestPhone := "0912" + sfx
	for day := 1; day <= 5; day++ {
		validDate := time.Now().In(memberVNLoc).AddDate(0, 0, day).Format(consts.DateLayout)
		var wg sync.WaitGroup
		errs := make([]error, sponsors)
		start := make(chan struct{})
		for i := range sponsors {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: userIDs[i], Roles: []string{consts.RoleMember}})
				<-start
				_, errs[i] = svc.CreateGuestPass(ctx, &cpv1.CreateGuestPassReq{BuyerName: "QA5-RB Khách", Phone: guestPhone, ValidDate: validDate, BranchID: branchID})
			}(i)
		}
		close(start)
		wg.Wait()
		ok, conflict := 0, 0
		for _, err := range errs {
			switch {
			case err == nil:
				ok++
			case codeOf(err) == rescode.Conflict.Code():
				conflict++
				assert.Contains(t, err.Error(), "đã có vé mời cho ngày")
			default:
				t.Errorf("lỗi không mong đợi: %v", err)
			}
		}
		assert.Equal(t, 1, ok, "ngày +%d: đúng một vé được tạo", day)
		assert.Equal(t, 1, conflict, "ngày +%d: lệnh còn lại 409", day)
		var n int64
		db.Raw(`SELECT count(*) FROM day_passes WHERE type = 'GUEST' AND phone = ? AND valid_date = ?`, guestPhone, validDate).Scan(&n)
		assert.EqualValues(t, 1, n)
	}
}

// Việc 5: thiết lập lại MFA không ghi created_at = 0001-01-01.
func TestQA5RBIT_UserMFACreatedAtPreserved(t *testing.T) {
	db := qa5rbDB(t)
	uid := qa5rbID(t, db, `INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'QA5-RB MFA') RETURNING id`,
		fmt.Sprintf("qa5rb-mfa-%d@it.local", time.Now().UnixNano()))
	t.Cleanup(func() {
		db.Exec(`DELETE FROM user_mfa WHERE user_id = ?`, uid)
		db.Exec(`DELETE FROM users WHERE id = ?`, uid)
	})
	repo := repository.NewPrivacyRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.SaveUserMFA(ctx, &entity.UserMFA{UserID: uid, TOTPSecretEnc: "s1", RecoveryCodesHash: []byte("[]")}))
	first, err := repo.FindUserMFA(ctx, uid)
	require.NoError(t, err)
	assert.True(t, first.CreatedAt.Year() >= 2000)

	// Thiết lập lại (struct mới, CreatedAt rỗng – đúng như SetupMFA).
	require.NoError(t, repo.SaveUserMFA(ctx, &entity.UserMFA{UserID: uid, TOTPSecretEnc: "s2", RecoveryCodesHash: []byte("[]")}))
	again, err := repo.FindUserMFA(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, "s2", again.TOTPSecretEnc)
	assert.True(t, again.CreatedAt.Equal(first.CreatedAt), "created_at giữ nguyên: %v → %v", first.CreatedAt, again.CreatedAt)
	assert.False(t, again.UpdatedAt.Before(first.UpdatedAt))

	// Bật MFA (struct đã đọc từ DB) vẫn giữ created_at.
	now := time.Now()
	again.EnabledAt = &now
	require.NoError(t, repo.SaveUserMFA(ctx, again))
	enabled, _ := repo.FindUserMFA(ctx, uid)
	assert.NotNil(t, enabled.EnabledAt)
	assert.True(t, enabled.CreatedAt.Equal(first.CreatedAt))
}

// Việc 4 (C11): đổi HLV hợp đồng PT – HLV mới không thấy tin cũ gửi HLV trước là "chưa đọc".
func TestQA5RBIT_ConversationUnreadAfterTrainerChange(t *testing.T) {
	f := ptITSetup(t)
	db := f.db
	sfx := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	newTrainer := qa5rbID(t, db, `INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'QA5-RB HLV mới') RETURNING id`, "qa5rb-t2-"+sfx+"@it.local")
	memberUser := qa5rbID(t, db, `INSERT INTO users (email, password_hash, full_name) VALUES (?, 'x', 'QA5-RB HV chat') RETURNING id`, "qa5rb-m-"+sfx+"@it.local")
	contractID, memberID := f.contracts[0], f.memberIDs[0]
	require.NoError(t, db.Exec(`UPDATE members SET user_id = ? WHERE id = ?`, memberUser, memberID).Error)
	t.Cleanup(func() {
		db.Exec(`DELETE FROM conversations WHERE pt_contract_id IN ?`, f.contracts)
		db.Exec(`UPDATE pt_contracts SET trainer_id = ? WHERE id = ?`, f.trainerID, contractID)
		db.Exec(`UPDATE members SET user_id = NULL WHERE id = ?`, memberID)
		db.Exec(`DELETE FROM users WHERE id IN ?`, []uint{newTrainer, memberUser})
	})
	svc := NewCMSService(repository.NewCMSRepository(db), nil)
	memberCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: memberUser, Roles: []string{consts.RoleMember}})
	oldCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: f.trainerID, Roles: []string{consts.RoleTrainer}})
	newCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: newTrainer, Roles: []string{consts.RoleTrainer}})

	unreadOf := func(ctx context.Context) int64 {
		list, err := svc.ListConversations(ctx, &cmsv1.ListConversationsReq{})
		require.NoError(t, err)
		for _, c := range list {
			if c.PTContractID == contractID {
				return c.UnreadCount
			}
		}
		return -1
	}
	convs, err := svc.ListConversations(memberCtx, &cmsv1.ListConversationsReq{})
	require.NoError(t, err)
	require.NotEmpty(t, convs)
	convID := convs[0].ID
	for i := range 3 {
		_, err := svc.SendMessage(memberCtx, &cmsv1.SendMessageReq{ID: convID, Content: fmt.Sprintf("Tin gửi HLV cũ %d", i)})
		require.NoError(t, err)
	}
	assert.EqualValues(t, 3, unreadOf(oldCtx), "HLV cũ thấy 3 tin chưa đọc")

	// Đổi HLV hợp đồng.
	require.NoError(t, db.Exec(`UPDATE pt_contracts SET trainer_id = ? WHERE id = ?`, newTrainer, contractID).Error)
	assert.EqualValues(t, 0, unreadOf(newCtx), "HLV mới không thấy tin cũ là chưa đọc")
	assert.EqualValues(t, -1, unreadOf(oldCtx), "HLV cũ mất hội thoại")

	time.Sleep(5 * time.Millisecond)
	_, err = svc.SendMessage(memberCtx, &cmsv1.SendMessageReq{ID: convID, Content: "Chào HLV mới"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, unreadOf(newCtx), "tin gửi sau khi bàn giao vẫn là chưa đọc")

	msgs, err := svc.ListMessages(newCtx, convID)
	require.NoError(t, err)
	assert.Len(t, msgs, 4, "HLV mới vẫn đọc được lịch sử")
	assert.EqualValues(t, 0, unreadOf(newCtx))
}

// Việc 6: dashboard (overview), Mật độ (occupancy) và check-in stats dùng cùng định nghĩa "đang trong phòng tập".
func TestQA5RBIT_PresentDefinitionConsistent(t *testing.T) {
	db := qa5rbDB(t)
	sfx := fmt.Sprintf("%06d", time.Now().UnixNano()%1_000_000)
	branchID := qa5rbID(t, db, `INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '00:00', '23:59') RETURNING id`, "Q5P"+sfx, "QA5-RB-P-"+sfx)
	var memberIDs []uint
	for i := range 3 {
		memberIDs = append(memberIDs, qa5rbID(t, db, `INSERT INTO members (full_name, phone, home_branch_id) VALUES ('QA5-RB P', ?, ?) RETURNING id`,
			fmt.Sprintf("071%s%d", sfx, i), branchID))
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM checkins WHERE branch_id = ?`, branchID)
		db.Exec(`DELETE FROM members WHERE id IN ?`, memberIDs)
		db.Exec(`DELETE FROM branches WHERE id = ?`, branchID)
	})
	now := time.Now()
	ins := func(member uint, at time.Time, out *time.Time) {
		require.NoError(t, db.Exec(`INSERT INTO checkins (member_id, branch_id, checked_in_at, checked_out_at, method, result) VALUES (?, ?, ?, ?, 'QR', 'ALLOWED')`,
			member, branchID, at, out).Error)
	}
	out := now.Add(-10 * time.Minute)
	ins(memberIDs[0], now.Add(-4*time.Hour), nil)    // quên checkout quá 3 giờ → không tính
	ins(memberIDs[1], now.Add(-90*time.Minute), nil) // đang tập
	ins(memberIDs[2], now.Add(-30*time.Minute), nil) // đang tập
	ins(memberIDs[2], now.Add(-2*time.Hour), &out)   // lượt đã checkout

	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	occ, err := repository.NewKPIRepository(db).ListBranchOccupancy(ctx, repository.PresentSince(now))
	require.NoError(t, err)
	occCount := -1
	for _, r := range occ {
		if r.BranchID == branchID {
			occCount = r.PresentCount
		}
	}
	ov, err := repository.NewReportRepository(db).Overview(ctx, repository.ReportOverviewParams{
		BranchIDs: []uint{branchID}, Now: now, TodayStart: now.Add(-24 * time.Hour), TomorrowStart: now.Add(time.Hour),
		MonthStart: now.AddDate(0, -1, 0), PrevMonthStart: now.AddDate(0, -2, 0), PrevMonthSame: now.AddDate(0, -1, 0),
		Today: now.Format(consts.DateLayout), Today7: now.AddDate(0, 0, 7).Format(consts.DateLayout),
	})
	require.NoError(t, err)
	ps := repository.PresentSince(now)
	st, err := repository.NewCheckinRepository(db).Stats(ctx, []uint{branchID}, now.Add(-24*time.Hour), now.Add(time.Hour), &ps)
	require.NoError(t, err)
	list, err := repository.NewCheckinRepository(db).Present(ctx, []uint{branchID}, ps)
	require.NoError(t, err)

	assert.Equal(t, 2, occCount, "Mật độ")
	assert.EqualValues(t, 2, ov.PresentNow, "dashboard")
	assert.EqualValues(t, 2, st.Present, "check-in stats")
	assert.Len(t, list, 2, "danh sách đang trong phòng tập")
}
