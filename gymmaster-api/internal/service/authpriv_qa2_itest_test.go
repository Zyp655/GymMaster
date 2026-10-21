//go:build integration

package service

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	sysv1 "gymmaster-api/api/systemext/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

// Chạy: GYMMASTER_TEST_DSN=postgres://gymmaster:gymmaster@localhost:5432/gymmaster?sslmode=disable make itest
// Hồi quy QA vòng 2 (xác thực/quyền riêng tư/tiện ích hệ thống) trên PostgreSQL thật. Dữ liệu tạm có tiền tố FIX2-AUTH-.

type qa2Fixture struct {
	db       *gorm.DB
	branchA  uint
	branchB  uint
	ownerCtx context.Context
	sfx      string
}

func qa2Setup(t *testing.T) *qa2Fixture {
	t.Helper()
	dsn := os.Getenv("GYMMASTER_TEST_DSN")
	if dsn == "" {
		t.Skip("chưa đặt GYMMASTER_TEST_DSN")
	}
	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	fx := &qa2Fixture{db: db, sfx: fmt.Sprintf("%06d", time.Now().UnixNano()%1_000_000)}
	for i, p := range []*uint{&fx.branchA, &fx.branchB} {
		require.NoError(t, db.Raw(`INSERT INTO branches (code, name, open_time, close_time) VALUES (?, ?, '05:00', '22:00') RETURNING id`,
			fmt.Sprintf("FA%d%s", i, fx.sfx), fmt.Sprintf("FIX2-AUTH-CN%d-%s", i, fx.sfx)).Scan(p).Error)
	}
	var ownerID uint
	db.Raw(`SELECT u.id FROM users u JOIN user_roles ur ON ur.user_id = u.id JOIN roles r ON r.id = ur.role_id WHERE r.code = 'OWNER' ORDER BY u.id LIMIT 1`).Scan(&ownerID)
	fx.ownerCtx = reqctx.WithAuth(context.Background(), &token.Payload{UserID: ownerID, Roles: []string{consts.RoleOwner}})
	t.Cleanup(func() {
		b := []uint{fx.branchA, fx.branchB}
		db.Exec(`DELETE FROM audit_logs WHERE entity = 'legacy_import' AND entity_id IN (SELECT id FROM legacy_import_jobs WHERE branch_id IN ?)`, b)
		db.Exec(`DELETE FROM legacy_import_jobs WHERE branch_id IN ?`, b)
		db.Exec(`DELETE FROM invoices WHERE branch_id IN ?`, b)
		db.Exec(`DELETE FROM subscriptions WHERE branch_id IN ?`, b)
		db.Exec(`DELETE FROM data_requests WHERE member_id IN (SELECT id FROM members WHERE home_branch_id IN ?)`, b)
		db.Exec(`DELETE FROM members WHERE home_branch_id IN ?`, b)
		db.Exec(`DELETE FROM branches WHERE id IN ?`, b)
	})
	return fx
}

func qa2Phone() string { return fmt.Sprintf("09%08d", rand.IntN(100_000_000)) }

func (fx *qa2Fixture) member(t *testing.T, name, phone, email string, branch uint, status string) uint {
	t.Helper()
	var id uint
	require.NoError(t, fx.db.Raw(`INSERT INTO members (full_name, gender, phone, email, home_branch_id, source, status, created_at, updated_at)
		VALUES (?, 'OTHER', ?, ?, ?, 'OTHER', ?, now(), now()) RETURNING id`, name, phone, email, branch, status).Scan(&id).Error)
	return id
}

func TestQA2IT_LegacyImport(t *testing.T) {
	fx := qa2Setup(t)
	svc := NewSystemExtService(repository.NewSystemExtRepository(fx.db), nil, NewAuditService(fx.db))
	otherBranchPhone := qa2Phone()
	fx.member(t, "FIX2-AUTH-CN-khac", otherBranchPhone, "", fx.branchB, entity.MemberStatusActive)
	samePhone := qa2Phone()
	existingID := fx.member(t, "FIX2-AUTH-Sẵn có", samePhone, "", fx.branchA, entity.MemberStatusActive)

	newPhone := qa2Phone()
	rows := []sysv1.LegacyImportRow{
		{FullName: "FIX2-AUTH-Mới", Phone: newPhone[:4] + " " + newPhone[4:], DebtAmount: 150000},
		{FullName: "FIX2-AUTH-Sẵn có", Phone: "+84" + samePhone[1:], SubStartDate: "2026-01-01", SubEndDate: "2027-01-01"},
	}

	t.Run("không có quyền chi nhánh → 403", func(t *testing.T) {
		ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleManager}, BranchIDs: []uint{fx.branchB}})
		_, err := svc.RunLegacyImport(ctx, &sysv1.RunLegacyImportReq{BranchID: fx.branchA, Mode: "COMMIT", Rows: rows})
		assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
	})
	t.Run("chi nhánh ngừng hoạt động → 400", func(t *testing.T) {
		fx.db.Exec(`UPDATE branches SET status = 'INACTIVE' WHERE id = ?`, fx.branchB)
		defer fx.db.Exec(`UPDATE branches SET status = 'ACTIVE' WHERE id = ?`, fx.branchB)
		_, err := svc.RunLegacyImport(fx.ownerCtx, &sysv1.RunLegacyImportReq{BranchID: fx.branchB, Mode: "DRY_RUN", Rows: rows})
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	})
	t.Run("SĐT thuộc chi nhánh khác → lỗi dòng, COMMIT không ghi gì, FAILED", func(t *testing.T) {
		bad := append([]sysv1.LegacyImportRow{}, rows...)
		bad = append(bad, sysv1.LegacyImportRow{FullName: "FIX2-AUTH-Chéo", Phone: otherBranchPhone})
		dry, err := svc.RunLegacyImport(fx.ownerCtx, &sysv1.RunLegacyImportReq{BranchID: fx.branchA, Mode: "DRY_RUN", Rows: bad})
		require.NoError(t, err)
		require.Len(t, dry.Errors, 1)
		assert.Equal(t, 3, dry.Errors[0].Row)
		res, err := svc.RunLegacyImport(fx.ownerCtx, &sysv1.RunLegacyImportReq{BranchID: fx.branchA, Mode: "COMMIT", Rows: bad})
		require.NoError(t, err)
		assert.Equal(t, "FAILED", res.Status)
		assert.Equal(t, 0, res.OkRows)
		var n int64
		fx.db.Model(&entity.Member{}).Where("phone = ?", newPhone).Count(&n)
		assert.Zero(t, n, "không ghi dở dang")
	})
	t.Run("COMMIT thành công: gắn vào hội viên sẵn có theo SĐT chuẩn hóa, created_by, audit; chạy lại không nhân đôi", func(t *testing.T) {
		res, err := svc.RunLegacyImport(fx.ownerCtx, &sysv1.RunLegacyImportReq{BranchID: fx.branchA, Mode: "COMMIT", Rows: rows})
		require.NoError(t, err)
		assert.Equal(t, "COMPLETED", res.Status)
		assert.Equal(t, 2, res.OkRows)
		assert.Empty(t, res.Errors)

		var created entity.Member
		require.NoError(t, fx.db.Where("phone = ?", newPhone).First(&created).Error)
		assert.Equal(t, fx.branchA, created.HomeBranchID)
		require.NotNil(t, created.CreatedBy)
		var subs, invs, audits int64
		fx.db.Model(&entity.Subscription{}).Where("member_id = ?", existingID).Count(&subs)
		fx.db.Model(&entity.Invoice{}).Where("member_id = ?", created.ID).Count(&invs)
		fx.db.Table("audit_logs").Where("entity = 'legacy_import' AND entity_id = ? AND actor_id IS NOT NULL", res.ID).Count(&audits)
		assert.EqualValues(t, 1, subs, "gói tập gắn vào hội viên sẵn có, không tạo hội viên trùng")
		assert.EqualValues(t, 1, invs)
		assert.EqualValues(t, 1, audits)

		_, err = svc.RunLegacyImport(fx.ownerCtx, &sysv1.RunLegacyImportReq{BranchID: fx.branchA, Mode: "COMMIT", Rows: rows})
		require.NoError(t, err)
		fx.db.Model(&entity.Subscription{}).Where("member_id = ?", existingID).Count(&subs)
		fx.db.Model(&entity.Invoice{}).Where("member_id = ?", created.ID).Count(&invs)
		assert.EqualValues(t, 1, subs, "chạy lại không nhân đôi gói tập")
		assert.EqualValues(t, 1, invs, "chạy lại không nhân đôi công nợ")
	})
}

func TestQA2IT_GlobalSearchAndBulkActivate(t *testing.T) {
	fx := qa2Setup(t)
	svc := NewSystemExtService(repository.NewSystemExtRepository(fx.db), nil, NewAuditService(fx.db))
	phone := qa2Phone()
	name := "FIX2-AUTH-Nguyễn Đức Thắng " + fx.sfx
	id := fx.member(t, name, phone, "fix2auth"+fx.sfx+"@mail.vn", fx.branchA, entity.MemberStatusActive)
	ctxA := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleReceptionist},
		Permissions: []string{consts.PermMemberView}, BranchIDs: []uint{fx.branchA}})
	ctxB := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleReceptionist},
		Permissions: []string{consts.PermMemberView}, BranchIDs: []uint{fx.branchB}})

	found := func(ctx context.Context, q string) bool {
		res, err := svc.GlobalSearch(ctx, q)
		require.NoError(t, err)
		for _, h := range res.Items {
			if h.Category == "MEMBER" && h.ID == id {
				return true
			}
		}
		return false
	}
	assert.True(t, found(ctxA, "nguyen duc thang "+fx.sfx), "không dấu")
	assert.True(t, found(ctxA, "FIX2AUTH"+fx.sfx+"@MAIL"), "email")
	assert.True(t, found(ctxA, "+84 "+phone[1:4]+" "+phone[4:7]+" "+phone[7:]), "SĐT có khoảng trắng/+84")
	assert.False(t, found(ctxB, "nguyen duc thang "+fx.sfx), "ngoài chi nhánh")
	res, err := svc.GlobalSearch(ctxA, "%")
	require.NoError(t, err)
	for _, h := range res.Items {
		assert.NotEqual(t, id, h.ID, "ký tự % được thoát, không khớp mọi thứ")
	}

	// ACTIVATE không mở lại hội viên BANNED hoặc đã ẩn danh.
	banned := fx.member(t, "FIX2-AUTH-Cấm", qa2Phone(), "", fx.branchA, entity.MemberStatusBanned)
	anon := fx.member(t, "Hội viên đã ẩn danh", fmt.Sprintf("ANON-%s", fx.sfx), "", fx.branchA, entity.MemberStatusInactive)
	inactive := fx.member(t, "FIX2-AUTH-Ngừng", qa2Phone(), "", fx.branchA, entity.MemberStatusInactive)
	out, err := svc.BulkMemberAction(fx.ownerCtx, &sysv1.BulkMemberActionReq{MemberIDs: []uint{banned, anon, inactive}, Action: "ACTIVATE"})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Affected)
	status := func(id uint) string {
		var s string
		fx.db.Raw(`SELECT status FROM members WHERE id = ?`, id).Scan(&s)
		return s
	}
	assert.Equal(t, entity.MemberStatusBanned, status(banned))
	assert.Equal(t, entity.MemberStatusInactive, status(anon))
	assert.Equal(t, entity.MemberStatusActive, status(inactive))
	_, err = svc.BulkMemberAction(fx.ownerCtx, &sysv1.BulkMemberActionReq{MemberIDs: []uint{banned}, Action: "DEACTIVATE"})
	require.NoError(t, err)
	assert.Equal(t, entity.MemberStatusBanned, status(banned), "DEACTIVATE không gỡ trạng thái cấm")
}

func TestQA2IT_BranchDeleteBlockedAndMFAConsumeAtomic(t *testing.T) {
	fx := qa2Setup(t)
	branches := NewBranchService(repository.NewBranchRepository(fx.db))
	fx.member(t, "FIX2-AUTH-HV", qa2Phone(), "", fx.branchA, entity.MemberStatusActive)
	err := branches.Delete(fx.ownerCtx, fx.branchA)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	require.NoError(t, branches.Delete(fx.ownerCtx, fx.branchB), "chi nhánh trống xóa được")

	// Mã khôi phục: 10 lần tiêu thụ song song chỉ 1 lần thành công (UPDATE có điều kiện trên DB thật).
	var uid uint
	require.NoError(t, fx.db.Raw(`INSERT INTO users (email, password_hash, full_name, status) VALUES (?, 'x', 'FIX2-AUTH-MFA', 'ACTIVE') RETURNING id`,
		"fix2-auth-mfa-"+fx.sfx+"@gm.vn").Scan(&uid).Error)
	t.Cleanup(func() {
		fx.db.Exec(`DELETE FROM user_mfa WHERE user_id = ?`, uid)
		fx.db.Exec(`DELETE FROM users WHERE id = ?`, uid)
	})
	h1, h2 := hash.SHA256("AAAA-1111"), hash.SHA256("BBBB-2222")
	require.NoError(t, fx.db.Exec(`INSERT INTO user_mfa (user_id, totp_secret_enc, enabled_at, recovery_codes_hash) VALUES (?, 'x', now(), ?::jsonb)`,
		uid, fmt.Sprintf(`[%q,%q]`, h1, h2)).Error)
	repo := repository.NewPrivacyRepository(fx.db)
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := repo.ConsumeRecoveryCode(context.Background(), uid, h1)
			assert.NoError(t, err)
			if ok {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, wins)
	ok, err := repo.ConsumeRecoveryCode(context.Background(), uid, h2)
	require.NoError(t, err)
	assert.True(t, ok, "mã còn lại không bị mất do cập nhật chồng")
	mfa, err := repo.FindUserMFA(context.Background(), uid)
	require.NoError(t, err)
	assert.Empty(t, parseRecoveryHashes(mfa.RecoveryCodesHash))

	ok, err = repo.ConsumeTOTPStep(context.Background(), uid, 100)
	require.NoError(t, err)
	assert.True(t, ok)
	ok, _ = repo.ConsumeTOTPStep(context.Background(), uid, 100)
	assert.False(t, ok, "cùng bước TOTP bị từ chối")
	ok, _ = repo.ConsumeTOTPStep(context.Background(), uid, 99)
	assert.False(t, ok, "bước cũ hơn bị từ chối")
}
