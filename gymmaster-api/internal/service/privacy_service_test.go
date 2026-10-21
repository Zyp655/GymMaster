package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/privacy/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/cryptoext"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

type fakePrivacyRepo struct {
	nextReqID uint

	users    map[uint]*entity.User
	mfas     map[uint]*entity.UserMFA
	sessions map[uint]*entity.RefreshToken
	consents map[[2]any]*entity.Consent
	requests map[uint]*entity.DataRequest
	members  map[uint]*entity.Member

	mu       sync.Mutex
	lastStep map[uint]int64
	exports  map[uint][]byte
}

func newFakePrivacyRepo() *fakePrivacyRepo {
	return &fakePrivacyRepo{
		users:    map[uint]*entity.User{},
		mfas:     map[uint]*entity.UserMFA{},
		sessions: map[uint]*entity.RefreshToken{},
		consents: map[[2]any]*entity.Consent{},
		requests: map[uint]*entity.DataRequest{},
		members:  map[uint]*entity.Member{},
		lastStep: map[uint]int64{},
		exports:  map[uint][]byte{},
	}
}

// ConsumeTOTPStep / ConsumeRecoveryCode mô phỏng câu UPDATE có điều kiện (nguyên tử) của repository thật.
func (f *fakePrivacyRepo) ConsumeTOTPStep(_ context.Context, userID uint, step int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.mfas[userID]; !ok || f.lastStep[userID] >= step {
		return false, nil
	}
	f.lastStep[userID] = step
	return true, nil
}

func (f *fakePrivacyRepo) ConsumeRecoveryCode(_ context.Context, userID uint, codeHash string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	mfa, ok := f.mfas[userID]
	if !ok || mfa.EnabledAt == nil {
		return false, nil
	}
	hashes := parseRecoveryHashes(mfa.RecoveryCodesHash)
	idx := slices.Index(hashes, codeHash)
	if idx < 0 {
		return false, nil
	}
	hashes = slices.Delete(hashes, idx, idx+1)
	b, _ := json.Marshal(hashes)
	mfa.RecoveryCodesHash = datatypes.JSON(b)
	return true, nil
}

func (f *fakePrivacyRepo) HasPendingDataRequest(_ context.Context, memberID uint, reqType string) (bool, error) {
	for _, dr := range f.requests {
		if dr.MemberID == memberID && dr.Type == reqType && dr.Status == entity.DataRequestPending {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakePrivacyRepo) SaveDataRequestExport(_ context.Context, id uint, data []byte) error {
	f.exports[id] = data
	return nil
}

func (f *fakePrivacyRepo) FindDataRequestExport(_ context.Context, id uint) ([]byte, error) {
	if b, ok := f.exports[id]; ok {
		return b, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePrivacyRepo) FindUserByID(_ context.Context, userID uint) (*entity.User, error) {
	if u, ok := f.users[userID]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePrivacyRepo) FindUserMFA(_ context.Context, userID uint) (*entity.UserMFA, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if mfa, ok := f.mfas[userID]; ok {
		cp := *mfa
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePrivacyRepo) SaveUserMFA(_ context.Context, mfa *entity.UserMFA) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *mfa
	f.mfas[mfa.UserID] = &cp
	return nil
}

func (f *fakePrivacyRepo) DeleteUserMFA(_ context.Context, userID uint) error {
	delete(f.mfas, userID)
	return nil
}

func (f *fakePrivacyRepo) ListActiveSessions(_ context.Context, userID uint, now time.Time) ([]entity.RefreshToken, error) {
	var out []entity.RefreshToken
	for _, rt := range f.sessions {
		if rt.UserID == userID && rt.RevokedAt == nil && rt.ExpiresAt.After(now) {
			out = append(out, *rt)
		}
	}
	return out, nil
}

func (f *fakePrivacyRepo) RevokeSession(_ context.Context, userID, sessionID uint, now time.Time) (bool, error) {
	if rt, ok := f.sessions[sessionID]; ok && rt.UserID == userID && rt.RevokedAt == nil {
		rt.RevokedAt = &now
		return true, nil
	}
	return false, nil
}

func (f *fakePrivacyRepo) RevokeOtherSessions(_ context.Context, userID, excludeSessionID uint, excludeHash string, now time.Time) (int64, error) {
	var cnt int64
	for _, rt := range f.sessions {
		if rt.UserID == userID && rt.RevokedAt == nil && rt.ID != excludeSessionID && (excludeHash == "" || rt.TokenHash != excludeHash) {
			t := now
			rt.RevokedAt = &t
			cnt++
		}
	}
	return cnt, nil
}

func (f *fakePrivacyRepo) ListMemberConsents(_ context.Context, memberID uint) ([]entity.Consent, error) {
	var out []entity.Consent
	for k, c := range f.consents {
		if k[0] == memberID {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (f *fakePrivacyRepo) UpsertConsent(_ context.Context, c *entity.Consent) error {
	cp := *c
	f.consents[[2]any{c.MemberID, c.Type}] = &cp
	return nil
}

func (f *fakePrivacyRepo) ListDataRequests(_ context.Context, _ dto.Query, _ string, _ string, branchIDs []uint) ([]repository.DataRequestRow, int64, error) {
	all, _ := f.ListMemberDataRequests(context.Background(), 0)
	var rows []repository.DataRequestRow
	for _, r := range all {
		if len(branchIDs) == 0 || slices.Contains(branchIDs, r.MemberBranchID) {
			rows = append(rows, r)
		}
	}
	return rows, int64(len(rows)), nil
}

func (f *fakePrivacyRepo) rowOf(dr *entity.DataRequest) repository.DataRequestRow {
	row := repository.DataRequestRow{DataRequest: *dr}
	if m, ok := f.members[dr.MemberID]; ok {
		row.MemberBranchID = m.HomeBranchID
	}
	return row
}

func (f *fakePrivacyRepo) ListMemberDataRequests(_ context.Context, memberID uint) ([]repository.DataRequestRow, error) {
	var out []repository.DataRequestRow
	for _, dr := range f.requests {
		if memberID == 0 || dr.MemberID == memberID {
			out = append(out, f.rowOf(dr))
		}
	}
	return out, nil
}

func (f *fakePrivacyRepo) FindDataRequestByID(_ context.Context, id uint) (*repository.DataRequestRow, error) {
	if dr, ok := f.requests[id]; ok {
		row := f.rowOf(dr)
		return &row, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePrivacyRepo) CreateDataRequest(_ context.Context, dr *entity.DataRequest) error {
	f.nextReqID++
	dr.ID = f.nextReqID
	dr.CreatedAt = time.Now()
	cp := *dr
	f.requests[dr.ID] = &cp
	return nil
}

func (f *fakePrivacyRepo) UpdateDataRequest(_ context.Context, dr *entity.DataRequest) error {
	cp := *dr
	f.requests[dr.ID] = &cp
	return nil
}

func (f *fakePrivacyRepo) FindMemberByID(_ context.Context, id uint) (*entity.Member, error) {
	if m, ok := f.members[id]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePrivacyRepo) FindMemberByUserID(_ context.Context, userID uint) (*entity.Member, error) {
	for _, m := range f.members {
		if m.UserID != nil && *m.UserID == userID {
			cp := *m
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePrivacyRepo) AnonymizeMember(_ context.Context, memberID uint, _ *uint) error {
	m, ok := f.members[memberID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	m.FullName = "Hội viên đã ẩn danh"
	m.Phone = ""
	m.Email = ""
	m.IDCard = ""
	m.Address = ""
	m.Status = entity.MemberStatusInactive
	if m.UserID != nil {
		if u, ok := f.users[*m.UserID]; ok {
			u.Email = "deleted-user@anonymized.invalid"
			u.Status = consts.StatusLocked
		}
	}
	return nil
}

func TestPrivacyService_MFAAndAnonymize(t *testing.T) {
	repo := newFakePrivacyRepo()
	svc := NewPrivacyService(repo, nil, nil, "test-encryption-key-32-characters")

	uid := uint(50)
	u := &entity.User{Email: "member50@gymmaster.vn", FullName: "Lê Văn C"}
	u.ID = uid
	repo.users[uid] = u

	m := &entity.Member{
		UserID:     &uid,
		MemberCode: "GM000050",
		FullName:   "Lê Văn C",
		Phone:      "0909123456",
		Email:      "member50@gymmaster.vn",
		IDCard:     "001099001234",
		Address:    "Quận 1, TP.HCM",
		Status:     entity.MemberStatusActive,
	}
	m.ID = 50
	repo.members[50] = m

	memberCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: uid, Roles: []string{consts.RoleMember}})
	adminCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})

	// 1. Setup 2FA TOTP
	setupRes, err := svc.SetupMFA(memberCtx)
	require.NoError(t, err)
	assert.NotEmpty(t, setupRes.Secret)
	assert.Contains(t, setupRes.OTPAuthURL, "otpauth://totp/")

	// Enable with wrong code -> MFAInvalidCode
	_, err = svc.EnableMFA(memberCtx, &v1.EnableMFAReq{Code: "000000"})
	assert.Equal(t, rescode.MFAInvalidCode.Code(), codeOf(err))

	// Enable with valid TOTP code
	validCode, err := cryptoext.GenerateTOTPCode(setupRes.Secret, time.Now())
	require.NoError(t, err)
	enableRes, err := svc.EnableMFA(memberCtx, &v1.EnableMFAReq{Code: validCode})
	require.NoError(t, err)
	assert.True(t, enableRes.Enabled)
	require.Len(t, enableRes.RecoveryCodes, 6)

	// Đã bật 2FA → không cho setup lại (ghi đè secret/tắt 2FA không cần mã).
	_, err = svc.SetupMFA(memberCtx)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	enabled, okCode, err := svc.CheckLoginMFA(context.Background(), uid, "")
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.False(t, okCode)

	// Verify using a recovery code (one-time use)
	recCode := enableRes.RecoveryCodes[0]
	verRes, err := svc.VerifyMFA(memberCtx, &v1.VerifyMFAReq{Code: recCode})
	require.NoError(t, err)
	assert.True(t, verRes.Verified)
	assert.True(t, verRes.UsedRecovery)

	// Same recovery code cannot be reused
	_, err = svc.VerifyMFA(memberCtx, &v1.VerifyMFAReq{Code: recCode})
	assert.Equal(t, rescode.MFAInvalidCode.Code(), codeOf(err))

	// 2. Nghị định 13/2023: Yêu cầu DELETE và Admin APPROVE -> ẩn danh hóa hội viên
	delReq, err := svc.CreateDataRequest(memberCtx, &v1.CreateDataRequestReq{
		Type:   entity.DataRequestDelete,
		Reason: "Không còn nhu cầu tập luyện",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.DataRequestPending, delReq.Status)

	processed, err := svc.ProcessDataRequest(adminCtx, &v1.ProcessDataRequestReq{
		ID:     delReq.ID,
		Action: "APPROVE",
		Note:   "Đã ẩn danh hóa hồ sơ theo Nghị định 13/2023",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.DataRequestDone, processed.Status)

	anonUser := repo.users[uid]
	assert.Equal(t, consts.StatusLocked, anonUser.Status, "tài khoản đăng nhập phải bị khóa khi ẩn danh hóa")
	assert.NotEqual(t, "member50@gymmaster.vn", anonUser.Email)

	anonMember := repo.members[50]
	assert.Equal(t, "Hội viên đã ẩn danh", anonMember.FullName)
	assert.Empty(t, anonMember.Phone)
	assert.Empty(t, anonMember.Email)
	assert.Empty(t, anonMember.IDCard)
	assert.Empty(t, anonMember.Address)
}

func TestPrivacyService_DataRequestBranchScope(t *testing.T) {
	repo := newFakePrivacyRepo()
	svc := NewPrivacyService(repo, nil, nil, "test-encryption-key-32-characters")

	m := &entity.Member{FullName: "Chi nhánh 2", HomeBranchID: 2, Status: entity.MemberStatusActive}
	m.ID = 80
	repo.members[80] = m
	repo.requests[1] = &entity.DataRequest{ID: 1, MemberID: 80, Type: entity.DataRequestDelete, Status: entity.DataRequestPending}
	repo.nextReqID = 1

	mgrBranch1 := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 9, Roles: []string{consts.RoleManager}, BranchIDs: []uint{1}})
	list, err := svc.ListDataRequests(mgrBranch1, &v1.ListDataRequestsReq{})
	require.NoError(t, err)
	assert.Empty(t, list.Items)

	_, err = svc.ListDataRequests(mgrBranch1, &v1.ListDataRequestsReq{BranchID: 2})
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	_, err = svc.ProcessDataRequest(mgrBranch1, &v1.ProcessDataRequestReq{ID: 1, Action: "APPROVE"})
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
	assert.Equal(t, "Chi nhánh 2", repo.members[80].FullName)

	mgrBranch2 := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 9, Roles: []string{consts.RoleManager}, BranchIDs: []uint{2}})
	_, err = svc.ProcessDataRequest(mgrBranch2, &v1.ProcessDataRequestReq{ID: 1, Action: "APPROVE"})
	require.NoError(t, err)
}

// ---- Hồi quy QA vòng 2 ------------------------------------------------------------

func newMFAFixture(t *testing.T) (*privacyService, *fakePrivacyRepo, context.Context, string, []string) {
	t.Helper()
	repo := newFakePrivacyRepo()
	svc := NewPrivacyService(repo, nil, nil, "test-encryption-key-32-characters").(*privacyService)
	now := time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	uid := uint(60)
	repo.users[uid] = &entity.User{BaseEntity: entity.BaseEntity{ID: uid}, Email: "u60@gm.vn"}
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: uid, Roles: []string{consts.RoleManager}})
	setup, err := svc.SetupMFA(ctx)
	require.NoError(t, err)
	code, _ := cryptoext.GenerateTOTPCode(setup.Secret, now)
	en, err := svc.EnableMFA(ctx, &v1.EnableMFAReq{Code: code})
	require.NoError(t, err)
	return svc, repo, ctx, setup.Secret, en.RecoveryCodes
}

func TestPrivacyMFA_RecoveryCodeConsumedOnceUnderConcurrency(t *testing.T) {
	svc, repo, _, _, codes := newMFAFixture(t)
	const n = 20
	var wg sync.WaitGroup
	var mu sync.Mutex
	okCount := 0
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok, err := svc.CheckLoginMFA(context.Background(), 60, codes[0])
			require.NoError(t, err)
			if ok {
				mu.Lock()
				okCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, okCount, "mã khôi phục chỉ dùng được đúng 1 lần dù đăng nhập song song")

	// Không mất cập nhật giữa hai mã khác nhau dùng song song.
	wg.Add(2)
	for _, c := range codes[1:3] {
		go func() {
			defer wg.Done()
			_, ok, err := svc.CheckLoginMFA(context.Background(), 60, c)
			assert.NoError(t, err)
			assert.True(t, ok)
		}()
	}
	wg.Wait()
	assert.Len(t, parseRecoveryHashes(repo.mfas[60].RecoveryCodesHash), 3, "6 mã − 3 mã đã dùng")

	// Gõ thường/thiếu gạch trên điện thoại vẫn nhận.
	_, ok, err := svc.CheckLoginMFA(context.Background(), 60, strings.ToLower(strings.ReplaceAll(codes[3], "-", "")))
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestPrivacyMFA_TOTPReplayRejected(t *testing.T) {
	svc, _, ctx, secret, _ := newMFAFixture(t)
	now := svc.now()
	// Mã đã dùng để kích hoạt không dùng lại được.
	enableCode, _ := cryptoext.GenerateTOTPCode(secret, now)
	_, ok, err := svc.CheckLoginMFA(context.Background(), 60, enableCode)
	require.NoError(t, err)
	assert.False(t, ok, "mã dùng để bật 2FA không được dùng lại để đăng nhập")

	// Sang bước kế tiếp: mã mới dùng được một lần, lần hai bị từ chối; mã cũ hơn (bước trước) cũng bị từ chối.
	later := now.Add(30 * time.Second)
	svc.now = func() time.Time { return later }
	next, _ := cryptoext.GenerateTOTPCode(secret, later)
	_, ok, err = svc.CheckLoginMFA(context.Background(), 60, next)
	require.NoError(t, err)
	assert.True(t, ok)
	_, ok, _ = svc.CheckLoginMFA(context.Background(), 60, next)
	assert.False(t, ok, "phát lại cùng mã TOTP bị từ chối")
	_, err = svc.VerifyMFA(ctx, &v1.VerifyMFAReq{Code: next})
	assert.Equal(t, rescode.MFAInvalidCode.Code(), codeOf(err))
	_, ok, _ = svc.CheckLoginMFA(context.Background(), 60, enableCode)
	assert.False(t, ok, "mã của bước cũ hơn bị từ chối")
}

func TestPrivacySessions_RevokeOthersKeepsCurrent(t *testing.T) {
	repo := newFakePrivacyRepo()
	svc := NewPrivacyService(repo, nil, nil, "test-encryption-key-32-characters")
	future := time.Now().Add(time.Hour)
	for id := uint(1); id <= 3; id++ {
		repo.sessions[id] = &entity.RefreshToken{ID: id, UserID: 7, TokenHash: fmt.Sprintf("h%d", id), ExpiresAt: future}
	}
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 7, SessionID: 2, Roles: []string{consts.RoleMember}})

	list, err := svc.ListSessions(ctx)
	require.NoError(t, err)
	current := 0
	for _, it := range list {
		if it.IsCurrent {
			current++
			assert.Equal(t, uint(2), it.ID)
		}
	}
	assert.Equal(t, 1, current)

	res, err := svc.RevokeOtherSessions(ctx, &v1.RevokeOtherSessionsReq{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.RevokedCount)
	assert.Nil(t, repo.sessions[2].RevokedAt, "phiên hiện tại được giữ")
	assert.NotNil(t, repo.sessions[1].RevokedAt)

	// Token cũ không có sid và không gửi gì → 400, không thu hồi phiên nào.
	repo.sessions[4] = &entity.RefreshToken{ID: 4, UserID: 7, ExpiresAt: future}
	oldCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 7, Roles: []string{consts.RoleMember}})
	_, err = svc.RevokeOtherSessions(oldCtx, &v1.RevokeOtherSessionsReq{})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	assert.Nil(t, repo.sessions[4].RevokedAt)
	// Token cũ + current_session_id trong body → dùng được.
	_, err = svc.RevokeOtherSessions(oldCtx, &v1.RevokeOtherSessionsReq{CurrentSessionID: 4})
	require.NoError(t, err)
	assert.Nil(t, repo.sessions[4].RevokedAt)
}

func TestPrivacyDataRequest_DuplicatePendingAndDownload(t *testing.T) {
	repo := newFakePrivacyRepo()
	svc := NewPrivacyService(repo, nil, nil, "test-encryption-key-32-characters")
	uidA, uidB := uint(71), uint(72)
	repo.members[1] = &entity.Member{BaseEntity: entity.BaseEntity{ID: 1}, UserID: &uidA, MemberCode: "GM000001", FullName: "A", Phone: "0901000001"}
	repo.members[2] = &entity.Member{BaseEntity: entity.BaseEntity{ID: 2}, UserID: &uidB, MemberCode: "GM000002", FullName: "B"}
	ctxA := reqctx.WithAuth(context.Background(), &token.Payload{UserID: uidA, Roles: []string{consts.RoleMember}})
	ctxB := reqctx.WithAuth(context.Background(), &token.Payload{UserID: uidB, Roles: []string{consts.RoleMember}})

	_, err := svc.CreateDataRequest(ctxA, &v1.CreateDataRequestReq{Type: "DELETE"})
	require.NoError(t, err)
	_, err = svc.CreateDataRequest(ctxA, &v1.CreateDataRequestReq{Type: "DELETE"})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err), "đã có yêu cầu xóa đang chờ")

	exp, err := svc.CreateDataRequest(ctxA, &v1.CreateDataRequestReq{Type: "EXPORT"})
	require.NoError(t, err)
	assert.Equal(t, entity.DataRequestDone, exp.Status)
	assert.Equal(t, fmt.Sprintf("/v1/member/data-requests/%d/download", exp.ID), exp.FileURL)

	data, name, err := svc.DownloadDataRequest(ctxA, exp.ID)
	require.NoError(t, err)
	assert.Contains(t, name, "GM000001")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Equal(t, "0901000001", parsed["phone"])

	// Người khác / yêu cầu không phải EXPORT đã xong → 404.
	_, _, err = svc.DownloadDataRequest(ctxB, exp.ID)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
	_, _, err = svc.DownloadDataRequest(ctxA, 1)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
	_, _, err = svc.DownloadDataRequest(ctxA, 999)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))

	// Danh sách của tôi có file_url cho EXPORT đã xong.
	list, err := svc.MemberDataRequests(ctxA)
	require.NoError(t, err)
	for _, it := range list {
		if it.Type == "EXPORT" {
			assert.NotEmpty(t, it.FileURL)
		}
	}
}
