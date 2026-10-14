package service

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/auth/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
	"gymmaster-api/internal/pkg/mailer"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

// ---- In-memory fakes for Auth Security ------------------------------------------

type fakePasswordResetRepo struct {
	nextID uint
	items  map[uint]*entity.PasswordReset
}

func newFakePasswordResetRepo() *fakePasswordResetRepo {
	return &fakePasswordResetRepo{items: map[uint]*entity.PasswordReset{}}
}

func (f *fakePasswordResetRepo) Create(_ context.Context, pr *entity.PasswordReset) error {
	f.nextID++
	pr.ID = f.nextID
	cp := *pr
	f.items[pr.ID] = &cp
	return nil
}

func (f *fakePasswordResetRepo) FindActiveForUpdate(_ context.Context, tokenHash string, now time.Time) (*entity.PasswordReset, error) {
	for _, pr := range f.items {
		if pr.TokenHash == tokenHash && pr.UsedAt == nil && pr.ExpiresAt.After(now) {
			cp := *pr
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePasswordResetRepo) MarkUsed(_ context.Context, id uint, now time.Time) error {
	if pr, ok := f.items[id]; ok {
		pr.UsedAt = &now
	}
	return nil
}

func (f *fakePasswordResetRepo) DeleteExpired(_ context.Context, before time.Time) (int64, error) {
	var deleted int64
	for id, pr := range f.items {
		if pr.ExpiresAt.Before(before) || (pr.UsedAt != nil && pr.UsedAt.Before(before)) {
			delete(f.items, id)
			deleted++
		}
	}
	return deleted, nil
}

type fakeLoginLimiter struct {
	attempts []entity.LoginAttempt
}

func (l *fakeLoginLimiter) IsLocked(_ context.Context, email, ip string, now time.Time) (bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	windowStart := now.Add(-repository.LoginLockWindow)
	cutoff := windowStart
	for _, a := range l.attempts {
		if strings.EqualFold(a.Email, email) && a.IP == ip && a.Success && !a.AttemptedAt.Before(windowStart) {
			if a.AttemptedAt.After(cutoff) {
				cutoff = a.AttemptedAt
			}
		}
	}
	count := 0
	for _, a := range l.attempts {
		if strings.EqualFold(a.Email, email) && a.IP == ip && !a.Success && !a.AttemptedAt.Before(windowStart) && !a.AttemptedAt.Before(cutoff) {
			if cutoff.Equal(windowStart) || a.AttemptedAt.After(cutoff) {
				count++
			}
		}
	}
	return count >= repository.LoginMaxFailedAttempts, nil
}

func (l *fakeLoginLimiter) RecordAttempt(_ context.Context, email, ip string, success bool, now time.Time) error {
	l.attempts = append(l.attempts, entity.LoginAttempt{
		Email:       strings.ToLower(strings.TrimSpace(email)),
		IP:          ip,
		AttemptedAt: now,
		Success:     success,
	})
	return nil
}

func (l *fakeLoginLimiter) CleanupBefore(_ context.Context, before time.Time) (int64, error) {
	var kept []entity.LoginAttempt
	var removed int64
	for _, a := range l.attempts {
		if a.AttemptedAt.Before(before) {
			removed++
		} else {
			kept = append(kept, a)
		}
	}
	l.attempts = kept
	return removed, nil
}

type fakeMemberRegisterRepo struct {
	users   *fakeUserRepo
	nextUID uint
	nextMID uint
	nextOID uint
	members map[uint]*entity.Member
	otps    map[uint]*entity.MemberRegisterOTP
}

func newFakeMemberRegisterRepo(users *fakeUserRepo) *fakeMemberRegisterRepo {
	return &fakeMemberRegisterRepo{
		users:   users,
		nextUID: 100,
		nextMID: 200,
		members: map[uint]*entity.Member{},
		otps:    map[uint]*entity.MemberRegisterOTP{},
	}
}

func (r *fakeMemberRegisterRepo) FindMemberByPhone(_ context.Context, phone string) (*entity.Member, error) {
	for _, m := range r.members {
		if m.Phone == phone {
			cp := *m
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeMemberRegisterRepo) DefaultBranchID(context.Context) (uint, error) {
	return 1, nil
}

func (r *fakeMemberRegisterRepo) MemberRoleID(context.Context) (uint, error) {
	return 5, nil
}

func (r *fakeMemberRegisterRepo) CreateUserAndMember(_ context.Context, user *entity.User, member *entity.Member, roleID uint) error {
	r.nextUID++
	user.ID = r.nextUID
	user.UserRoles = []entity.UserRole{{UserID: user.ID, RoleID: roleID, Role: entity.Role{Code: consts.RoleMember}}}
	cpUser := *user
	r.users.users[user.ID] = &cpUser

	r.nextMID++
	member.ID = r.nextMID
	member.UserID = &user.ID
	cpMem := *member
	r.members[member.ID] = &cpMem
	return nil
}

func (r *fakeMemberRegisterRepo) LinkUserToMember(_ context.Context, memberID uint, user *entity.User, roleID uint) error {
	r.nextUID++
	user.ID = r.nextUID
	user.UserRoles = []entity.UserRole{{UserID: user.ID, RoleID: roleID, Role: entity.Role{Code: consts.RoleMember}}}
	cpUser := *user
	r.users.users[user.ID] = &cpUser

	if m, ok := r.members[memberID]; ok {
		uid := user.ID
		m.UserID = &uid
		m.Email = user.Email
	}
	return nil
}

func (r *fakeMemberRegisterRepo) CreateOTP(_ context.Context, otp *entity.MemberRegisterOTP) error {
	r.nextOID++
	otp.ID = r.nextOID
	cp := *otp
	r.otps[otp.ID] = &cp
	return nil
}

func (r *fakeMemberRegisterRepo) FindLatestActiveOTP(_ context.Context, memberID uint, email string, now time.Time) (*entity.MemberRegisterOTP, error) {
	var best *entity.MemberRegisterOTP
	for _, o := range r.otps {
		if o.MemberID == memberID && strings.EqualFold(o.Email, email) && o.UsedAt == nil && o.ExpiresAt.After(now) {
			if best == nil || o.ID > best.ID {
				cp := *o
				best = &cp
			}
		}
	}
	if best == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return best, nil
}

func (r *fakeMemberRegisterRepo) MarkOTPUsed(_ context.Context, id uint, now time.Time) error {
	if o, ok := r.otps[id]; ok {
		o.UsedAt = &now
	}
	return nil
}

type fakeAuthMailer struct {
	sent []mailer.Message
}

func (m *fakeAuthMailer) Send(_ context.Context, msg mailer.Message) error {
	m.sent = append(m.sent, msg)
	return nil
}

func (m *fakeAuthMailer) SendAsync(msg mailer.Message) {
	m.sent = append(m.sent, msg)
}

func (m *fakeAuthMailer) Wait() {}

func newAuthSecurityFixture(t *testing.T) (*authService, *fakeUserRepo, *fakeRefreshRepo, *fakePasswordResetRepo, *fakeLoginLimiter, *fakeMemberRegisterRepo, *fakeAuthMailer) {
	t.Helper()
	svc, users, refresh := newAuthFixture(t)
	resets := newFakePasswordResetRepo()
	limiter := &fakeLoginLimiter{}
	memReg := newFakeMemberRegisterRepo(users)
	mail := &fakeAuthMailer{}
	svc.SetSecurity(AuthSecurityDeps{
		PasswordResets: resets,
		Limiter:        limiter,
		MemberRegister: memReg,
		Mailer:         mail,
		WebURL:         "http://localhost:3000",
		AdminURL:       "http://localhost:3001",
	})
	return svc, users, refresh, resets, limiter, memReg, mail
}

// ---- Tests ----------------------------------------------------------------------

func TestLogin_RateLimitBruteForce(t *testing.T) {
	svc, _, _, _, _, _, _ := newAuthSecurityFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	client := ClientInfo{IP: "10.0.0.1"}

	// 5 lần nhập sai liên tiếp → trả InvalidCredentials.
	for i := 0; i < 5; i++ {
		_, err := svc.Login(ctx, "lt@gm.vn", "wrong-pass", client)
		assert.Equal(t, rescode.InvalidCredentials.Code(), codeOf(err))
	}

	// Lần thứ 6 (kể cả đúng mật khẩu) → bị khóa tạm 15 phút (40105 TooManyAttempts).
	_, err := svc.Login(ctx, "lt@gm.vn", testPassword, client)
	assert.Equal(t, rescode.TooManyAttempts.Code(), codeOf(err))

	// Sau 16 phút → hết khóa, đăng nhập thành công.
	now = now.Add(16 * time.Minute)
	res, err := svc.Login(ctx, "lt@gm.vn", testPassword, client)
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
}

func TestForgotAndResetPassword(t *testing.T) {
	svc, users, refresh, _, _, _, mail := newAuthSecurityFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	// User đăng nhập trước để có refresh token đang hoạt động.
	_, err := svc.Login(ctx, "lt@gm.vn", testPassword, ClientInfo{IP: "127.0.0.1"})
	require.NoError(t, err)
	assert.Equal(t, 1, refresh.activeCount(2))

	// Email không tồn tại → vẫn trả thành công, không gửi email.
	forgotRes, err := svc.ForgotPassword(ctx, "khongtontai@gm.vn", "WEB", ClientInfo{IP: "127.0.0.1"})
	require.NoError(t, err)
	assert.Contains(t, forgotRes.Message, "Nếu email tồn tại")
	assert.Empty(t, mail.sent)

	// Email hợp lệ (portal ADMIN) → gửi email chứa link reset-password?token=...
	forgotRes, err = svc.ForgotPassword(ctx, "lt@gm.vn", "ADMIN", ClientInfo{IP: "127.0.0.1"})
	require.NoError(t, err)
	require.Len(t, mail.sent, 1)
	assert.Contains(t, mail.sent[0].HTML, "http://localhost:3001/reset-password?token=")

	re := regexp.MustCompile(`token=([a-f0-9]{64})`)
	matches := re.FindStringSubmatch(mail.sent[0].HTML)
	require.Len(t, matches, 2)
	rawToken := matches[1]

	// Token sai → 40104 ResetTokenInvalid.
	_, err = svc.ResetPassword(ctx, "token-khong-hop-le", "NewPass@123")
	assert.Equal(t, rescode.ResetTokenInvalid.Code(), codeOf(err))

	// Đặt lại mật khẩu thành công → đổi mật khẩu + thu hồi toàn bộ refresh token.
	resetRes, err := svc.ResetPassword(ctx, rawToken, "NewPass@123")
	require.NoError(t, err)
	assert.Contains(t, resetRes.Message, "thành công")
	assert.True(t, hash.CheckPassword(users.users[2].PasswordHash, "NewPass@123"))
	assert.Equal(t, 0, refresh.activeCount(2), "reset mật khẩu phải thu hồi toàn bộ refresh token")

	// Dùng lại token lần 2 → 40104 ResetTokenInvalid.
	_, err = svc.ResetPassword(ctx, rawToken, "AnotherPass@123")
	assert.Equal(t, rescode.ResetTokenInvalid.Code(), codeOf(err))
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	svc, _, _, _, _, _, mail := newAuthSecurityFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	_, err := svc.ForgotPassword(ctx, "lt@gm.vn", "WEB", ClientInfo{})
	require.NoError(t, err)
	re := regexp.MustCompile(`token=([a-f0-9]{64})`)
	rawToken := re.FindStringSubmatch(mail.sent[0].HTML)[1]

	// Quá 30 phút → hết hạn.
	now = now.Add(31 * time.Minute)
	_, err = svc.ResetPassword(ctx, rawToken, "NewPass@123")
	assert.Equal(t, rescode.ResetTokenInvalid.Code(), codeOf(err))
}

func TestRegisterMember_NewMemberAndExistingMemberOTP(t *testing.T) {
	svc, _, _, _, _, memReg, mail := newAuthSecurityFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	// 1. Số điện thoại mới hoàn toàn → tạo User + Member ngay, trả token.
	regRes, err := svc.RegisterMember(ctx, &v1.RegisterMemberReq{
		FullName: "Nguyễn Văn Mới",
		Phone:    "0912345678",
		Email:    "moi@gm.vn",
		Password: "Password@123",
		BranchID: 1,
	}, ClientInfo{IP: "127.0.0.1"})
	require.NoError(t, err)
	assert.False(t, regRes.RequiresOTP)
	require.NotNil(t, regRes.Token)
	assert.Equal(t, []string{consts.RoleMember}, regRes.Token.User.Roles)

	// Đăng ký lại cùng SĐT đã có tài khoản → 409 Conflict.
	_, err = svc.RegisterMember(ctx, &v1.RegisterMemberReq{
		FullName: "Nguyễn Văn Mới 2",
		Phone:    "0912345678",
		Email:    "moi2@gm.vn",
		Password: "Password@123",
	}, ClientInfo{})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	// 2. Hội viên đã có hồ sơ tại quầy (UserID == nil) → gửi OTP, RequiresOTP = true.
	memReg.members[50] = &entity.Member{
		BaseEntity:   entity.BaseEntity{ID: 50},
		FullName:     "Trần Thị Quầy",
		Phone:        "0987654321",
		HomeBranchID: 1,
		Status:       entity.MemberStatusActive,
	}
	otpRes, err := svc.RegisterMember(ctx, &v1.RegisterMemberReq{
		FullName: "Trần Thị Quầy",
		Phone:    "0987654321",
		Email:    "quay@gm.vn",
		Password: "PortalPass@123",
	}, ClientInfo{})
	require.NoError(t, err)
	assert.True(t, otpRes.RequiresOTP)
	assert.Nil(t, otpRes.Token)
	require.Len(t, mail.sent, 1)

	reOTP := regexp.MustCompile(`>(\d{6})<`)
	otpMatches := reOTP.FindStringSubmatch(mail.sent[0].HTML)
	require.Len(t, otpMatches, 2)
	otpCode := otpMatches[1]

	// Nhập sai OTP → 40104 ResetTokenInvalid.
	_, err = svc.VerifyMemberOTP(ctx, &v1.VerifyMemberOTPReq{
		Phone: "0987654321",
		Email: "quay@gm.vn",
		OTP:   "000000",
	}, ClientInfo{})
	if otpCode != "000000" {
		assert.Equal(t, rescode.ResetTokenInvalid.Code(), codeOf(err))
	}

	// Nhập đúng OTP → liên kết tài khoản và trả TokenRes.
	tokenRes, err := svc.VerifyMemberOTP(ctx, &v1.VerifyMemberOTPReq{
		Phone: "0987654321",
		Email: "quay@gm.vn",
		OTP:   otpCode,
	}, ClientInfo{IP: "127.0.0.1"})
	require.NoError(t, err)
	assert.NotEmpty(t, tokenRes.AccessToken)
	assert.Equal(t, "quay@gm.vn", tokenRes.User.Email)
	assert.NotNil(t, memReg.members[50].UserID)

	// Đăng nhập bằng mật khẩu vừa tạo.
	loginRes, err := svc.Login(ctx, "quay@gm.vn", "PortalPass@123", ClientInfo{IP: "127.0.0.1"})
	require.NoError(t, err)
	assert.Equal(t, tokenRes.User.ID, loginRes.User.ID)
}
