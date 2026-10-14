package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
}

func newFakePrivacyRepo() *fakePrivacyRepo {
	return &fakePrivacyRepo{
		users:    map[uint]*entity.User{},
		mfas:     map[uint]*entity.UserMFA{},
		sessions: map[uint]*entity.RefreshToken{},
		consents: map[[2]any]*entity.Consent{},
		requests: map[uint]*entity.DataRequest{},
		members:  map[uint]*entity.Member{},
	}
}

func (f *fakePrivacyRepo) FindUserByID(_ context.Context, userID uint) (*entity.User, error) {
	if u, ok := f.users[userID]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePrivacyRepo) FindUserMFA(_ context.Context, userID uint) (*entity.UserMFA, error) {
	if mfa, ok := f.mfas[userID]; ok {
		cp := *mfa
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePrivacyRepo) SaveUserMFA(_ context.Context, mfa *entity.UserMFA) error {
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

func (f *fakePrivacyRepo) RevokeOtherSessions(_ context.Context, userID, excludeSessionID uint, _ string, now time.Time) (int64, error) {
	var cnt int64
	for _, rt := range f.sessions {
		if rt.UserID == userID && rt.RevokedAt == nil && rt.ID != excludeSessionID {
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

func (f *fakePrivacyRepo) ListDataRequests(_ context.Context, _ dto.Query, _ string, _ string) ([]repository.DataRequestRow, int64, error) {
	rows, _ := f.ListMemberDataRequests(context.Background(), 0)
	return rows, int64(len(rows)), nil
}

func (f *fakePrivacyRepo) ListMemberDataRequests(_ context.Context, memberID uint) ([]repository.DataRequestRow, error) {
	var out []repository.DataRequestRow
	for _, dr := range f.requests {
		if memberID == 0 || dr.MemberID == memberID {
			out = append(out, repository.DataRequestRow{DataRequest: *dr})
		}
	}
	return out, nil
}

func (f *fakePrivacyRepo) FindDataRequestByID(_ context.Context, id uint) (*repository.DataRequestRow, error) {
	if dr, ok := f.requests[id]; ok {
		return &repository.DataRequestRow{DataRequest: *dr}, nil
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

	anonMember := repo.members[50]
	assert.Equal(t, "Hội viên đã ẩn danh", anonMember.FullName)
	assert.Empty(t, anonMember.Phone)
	assert.Empty(t, anonMember.Email)
	assert.Empty(t, anonMember.IDCard)
	assert.Empty(t, anonMember.Address)
}
