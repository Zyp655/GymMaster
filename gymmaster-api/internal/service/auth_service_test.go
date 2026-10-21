package service

import (
	"context"
	"testing"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
)

// ---- fakes (in-memory) ------------------------------------------------------

type fakeUserRepo struct {
	users map[uint]*entity.User
	perms map[uint][]string // roleID -> permission codes
}

func (f *fakeUserRepo) FindByEmail(_ context.Context, email string) (*entity.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeUserRepo) FindByID(_ context.Context, id uint) (*entity.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeUserRepo) Create(context.Context, *entity.User) error         { return nil }
func (f *fakeUserRepo) AssignRole(context.Context, *entity.UserRole) error { return nil }
func (f *fakeUserRepo) UpdateLastLogin(_ context.Context, id uint, at time.Time) error {
	f.users[id].LastLoginAt = &at
	return nil
}
func (f *fakeUserRepo) UpdatePassword(_ context.Context, id uint, h string) error {
	f.users[id].PasswordHash = h
	return nil
}
func (f *fakeUserRepo) PermissionCodes(_ context.Context, roleIDs []uint) ([]string, error) {
	out := []string{}
	for _, id := range roleIDs {
		out = append(out, f.perms[id]...)
	}
	return out, nil
}

type fakeRefreshRepo struct {
	nextID uint
	tokens map[string]*entity.RefreshToken // hash -> token
}

func (f *fakeRefreshRepo) Create(_ context.Context, t *entity.RefreshToken) error {
	f.nextID++
	t.ID = f.nextID
	f.tokens[t.TokenHash] = t
	return nil
}
func (f *fakeRefreshRepo) FindActiveForUpdate(_ context.Context, h string, now time.Time) (*entity.RefreshToken, error) {
	if t, ok := f.tokens[h]; ok && t.RevokedAt == nil && t.ExpiresAt.After(now) {
		return t, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeRefreshRepo) FindByHash(_ context.Context, h string) (*entity.RefreshToken, error) {
	if t, ok := f.tokens[h]; ok {
		return t, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeRefreshRepo) Revoke(_ context.Context, id uint, at time.Time) error {
	for _, t := range f.tokens {
		if t.ID == id {
			t.RevokedAt = &at
			t.LastUsedAt = &at
		}
	}
	return nil
}
func (f *fakeRefreshRepo) RevokeByHash(_ context.Context, userID uint, h string, at time.Time) error {
	if t, ok := f.tokens[h]; ok && t.UserID == userID && t.RevokedAt == nil {
		t.RevokedAt = &at
	}
	return nil
}
func (f *fakeRefreshRepo) RevokeAllByUser(_ context.Context, userID uint, at time.Time) error {
	for _, t := range f.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &at
		}
	}
	return nil
}
func (f *fakeRefreshRepo) activeCount(userID uint) int {
	n := 0
	for _, t := range f.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			n++
		}
	}
	return n
}

type noTx struct{}

func (noTx) WithTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

// ---- helpers ----------------------------------------------------------------

const testPassword = "Secret@123"

func newAuthFixture(t *testing.T) (*authService, *fakeUserRepo, *fakeRefreshRepo) {
	t.Helper()
	pw, err := hash.HashPassword(testPassword)
	require.NoError(t, err)

	branchID := uint(2)
	users := &fakeUserRepo{
		users: map[uint]*entity.User{
			1: {BaseEntity: entity.BaseEntity{ID: 1}, Email: "owner@gm.vn", PasswordHash: pw, FullName: "Owner", Status: consts.StatusActive,
				UserRoles: []entity.UserRole{{RoleID: 1, Role: entity.Role{Code: consts.RoleOwner}}}},
			2: {BaseEntity: entity.BaseEntity{ID: 2}, Email: "lt@gm.vn", PasswordHash: pw, FullName: "Lễ tân", Status: consts.StatusActive,
				UserRoles: []entity.UserRole{{RoleID: 3, BranchID: &branchID, Role: entity.Role{Code: consts.RoleReceptionist}}}},
			3: {BaseEntity: entity.BaseEntity{ID: 3}, Email: "locked@gm.vn", PasswordHash: pw, FullName: "Bị khóa", Status: consts.StatusLocked},
		},
		perms: map[uint][]string{1: {"branch.view", "branch.create"}, 3: {"branch.view"}},
	}
	refresh := &fakeRefreshRepo{tokens: map[string]*entity.RefreshToken{}}
	maker, err := token.NewJWTMaker("test-secret-test-secret-test-secret!", "test", time.Minute)
	require.NoError(t, err)

	svc := NewAuthService(users, refresh, maker, noTx{}, time.Minute, time.Hour).(*authService)
	return svc, users, refresh
}

func codeOf(err error) int {
	return gerror.Code(err).Code()
}

// ---- tests ------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	svc, users, refresh := newAuthFixture(t)

	res, err := svc.Login(context.Background(), "lt@gm.vn", testPassword, "", ClientInfo{IP: "127.0.0.1"})
	require.NoError(t, err)

	assert.NotEmpty(t, res.AccessToken)
	assert.Len(t, res.RefreshToken, 64)
	assert.Equal(t, 60, res.ExpiresIn)
	assert.Equal(t, []string{consts.RoleReceptionist}, res.User.Roles)
	assert.Equal(t, []string{"branch.view"}, res.User.Permissions)
	assert.Equal(t, []uint{2}, res.User.BranchIDs, "role gắn chi nhánh → chỉ làm việc ở chi nhánh đó")
	assert.Equal(t, 1, refresh.activeCount(2))
	assert.NotNil(t, users.users[2].LastLoginAt)

	// Refresh token chỉ lưu dạng băm.
	_, stored := refresh.tokens[res.RefreshToken]
	assert.False(t, stored)
}

func TestLogin_OwnerHasGlobalScope(t *testing.T) {
	svc, _, _ := newAuthFixture(t)
	res, err := svc.Login(context.Background(), "owner@gm.vn", testPassword, "", ClientInfo{})
	require.NoError(t, err)
	assert.Empty(t, res.User.BranchIDs)
}

func TestLogin_Failures(t *testing.T) {
	svc, _, _ := newAuthFixture(t)
	ctx := context.Background()

	_, err := svc.Login(ctx, "khongton@tai.vn", testPassword, "", ClientInfo{})
	assert.Equal(t, rescode.InvalidCredentials.Code(), codeOf(err))

	_, err = svc.Login(ctx, "lt@gm.vn", "sai-mat-khau", "", ClientInfo{})
	assert.Equal(t, rescode.InvalidCredentials.Code(), codeOf(err))

	_, err = svc.Login(ctx, "locked@gm.vn", testPassword, "", ClientInfo{})
	assert.Equal(t, rescode.AccountLocked.Code(), codeOf(err))
}

func TestRefresh_RotatesToken(t *testing.T) {
	svc, _, refresh := newAuthFixture(t)
	ctx := context.Background()

	first, err := svc.Login(ctx, "lt@gm.vn", testPassword, "", ClientInfo{})
	require.NoError(t, err)

	second, err := svc.Refresh(ctx, first.RefreshToken, ClientInfo{})
	require.NoError(t, err)
	assert.NotEqual(t, first.RefreshToken, second.RefreshToken)
	assert.Equal(t, 1, refresh.activeCount(2), "token cũ bị thu hồi, chỉ còn token mới")
}

func TestRefresh_ReuseRevokesAllSessions(t *testing.T) {
	svc, _, refresh := newAuthFixture(t)
	ctx := context.Background()

	first, err := svc.Login(ctx, "lt@gm.vn", testPassword, "", ClientInfo{})
	require.NoError(t, err)
	_, err = svc.Refresh(ctx, first.RefreshToken, ClientInfo{})
	require.NoError(t, err)

	// Kẻ gian dùng lại refresh token đã bị xoay vòng.
	_, err = svc.Refresh(ctx, first.RefreshToken, ClientInfo{})
	assert.Equal(t, rescode.RefreshTokenInvalid.Code(), codeOf(err))
	assert.Equal(t, 0, refresh.activeCount(2), "phát hiện dùng lại → thu hồi toàn bộ phiên")
}

func TestRefresh_RemotelyRevokedSessionDoesNotRevokeActiveSessions(t *testing.T) {
	svc, _, refresh := newAuthFixture(t)
	ctx := context.Background()

	devA, err := svc.Login(ctx, "lt@gm.vn", testPassword, "", ClientInfo{UserAgent: "Desktop"})
	require.NoError(t, err)
	devB, err := svc.Login(ctx, "lt@gm.vn", testPassword, "", ClientInfo{UserAgent: "Mobile"})
	require.NoError(t, err)
	require.Equal(t, 2, refresh.activeCount(2))

	// Thiết bị B bị thu hồi từ xa (hoặc đăng xuất) → chỉ có RevokedAt, không có LastUsedAt.
	require.NoError(t, svc.Logout(ctx, devB.RefreshToken))
	require.Equal(t, 1, refresh.activeCount(2))

	// Thiết bị B tự động gọi Refresh → bị từ chối nhưng KHÔNG làm văng phiên của Thiết bị A.
	_, err = svc.Refresh(ctx, devB.RefreshToken, ClientInfo{UserAgent: "Mobile"})
	assert.Equal(t, rescode.RefreshTokenInvalid.Code(), codeOf(err))
	assert.Equal(t, 1, refresh.activeCount(2), "phiên của Thiết bị A vẫn phải còn hiệu lực")

	rotatedA, err := svc.Refresh(ctx, devA.RefreshToken, ClientInfo{UserAgent: "Desktop"})
	require.NoError(t, err)
	assert.NotEmpty(t, rotatedA.AccessToken)
}

func TestRefresh_UnknownOrExpired(t *testing.T) {
	svc, _, _ := newAuthFixture(t)
	ctx := context.Background()

	_, err := svc.Refresh(ctx, "khong-ton-tai", ClientInfo{})
	assert.Equal(t, rescode.RefreshTokenInvalid.Code(), codeOf(err))

	res, err := svc.Login(ctx, "lt@gm.vn", testPassword, "", ClientInfo{})
	require.NoError(t, err)
	svc.now = func() time.Time { return time.Now().Add(2 * time.Hour) } // quá RefreshTTL
	_, err = svc.Refresh(ctx, res.RefreshToken, ClientInfo{})
	assert.Equal(t, rescode.RefreshTokenInvalid.Code(), codeOf(err))
}

func TestChangePassword(t *testing.T) {
	svc, users, refresh := newAuthFixture(t)
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 2})

	_, err := svc.Login(context.Background(), "lt@gm.vn", testPassword, "", ClientInfo{})
	require.NoError(t, err)

	err = svc.ChangePassword(ctx, "sai", "MatKhauMoi@1")
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	require.NoError(t, svc.ChangePassword(ctx, testPassword, "MatKhauMoi@1"))
	assert.True(t, hash.CheckPassword(users.users[2].PasswordHash, "MatKhauMoi@1"))
	assert.Equal(t, 0, refresh.activeCount(2), "đổi mật khẩu → đăng xuất mọi thiết bị")
}
