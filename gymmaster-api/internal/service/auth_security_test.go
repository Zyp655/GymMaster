package service

import (
	"context"
	"fmt"
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
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

// ---- In-memory fakes for Auth Security ------------------------------------------

type fakePasswordResetRepo struct {
	now    func() time.Time
	nextID uint
	items  map[uint]*entity.PasswordReset
}

func newFakePasswordResetRepo() *fakePasswordResetRepo {
	return &fakePasswordResetRepo{items: map[uint]*entity.PasswordReset{}}
}

func (f *fakePasswordResetRepo) Create(_ context.Context, pr *entity.PasswordReset) error {
	f.nextID++
	pr.ID = f.nextID
	if pr.CreatedAt.IsZero() && f.now != nil {
		pr.CreatedAt = f.now()
	}
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

func (f *fakePasswordResetRepo) InvalidateAllForUser(_ context.Context, userID uint, now time.Time) error {
	for _, pr := range f.items {
		if pr.UserID == userID && pr.UsedAt == nil {
			pr.UsedAt = &now
		}
	}
	return nil
}

func (f *fakePasswordResetRepo) CountSince(_ context.Context, userID uint, since time.Time) (int64, error) {
	var n int64
	for _, pr := range f.items {
		if pr.UserID == userID && !pr.CreatedAt.Before(since) {
			n++
		}
	}
	return n, nil
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
	if count >= repository.LoginMaxFailedAttempts {
		return true, nil
	}
	emailFails := 0
	for _, a := range l.attempts {
		if strings.EqualFold(a.Email, email) && !a.Success && !a.AttemptedAt.Before(windowStart) {
			emailFails++
		}
	}
	return emailFails >= repository.LoginMaxFailedPerEmail, nil
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
	now     func() time.Time
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

func (r *fakeMemberRegisterRepo) IsActiveBranch(_ context.Context, id uint) (bool, error) {
	return id == 1, nil
}

func (r *fakeMemberRegisterRepo) CountOTPsSince(_ context.Context, memberID uint, since time.Time) (int64, error) {
	var n int64
	for _, o := range r.otps {
		if o.MemberID == memberID && !o.CreatedAt.Before(since) {
			n++
		}
	}
	return n, nil
}

func (r *fakeMemberRegisterRepo) RecordOTPFailure(_ context.Context, id uint, maxAttempts int, now time.Time) (int, error) {
	o, ok := r.otps[id]
	if !ok {
		return 0, gorm.ErrRecordNotFound
	}
	o.Attempts++
	if o.Attempts >= maxAttempts {
		o.UsedAt = &now
	}
	return o.Attempts, nil
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
	if otp.CreatedAt.IsZero() && r.now != nil {
		otp.CreatedAt = r.now()
	}
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
	resets.now = func() time.Time { return svc.now() }
	memReg.now = func() time.Time { return svc.now() }
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
		_, err := svc.Login(ctx, "lt@gm.vn", "wrong-pass", "", client)
		assert.Equal(t, rescode.InvalidCredentials.Code(), codeOf(err))
	}

	// Lần thứ 6 (kể cả đúng mật khẩu) → bị khóa tạm 15 phút (40105 TooManyAttempts).
	_, err := svc.Login(ctx, "lt@gm.vn", testPassword, "", client)
	assert.Equal(t, rescode.TooManyAttempts.Code(), codeOf(err))

	// Sau 16 phút → hết khóa, đăng nhập thành công.
	now = now.Add(16 * time.Minute)
	res, err := svc.Login(ctx, "lt@gm.vn", testPassword, "", client)
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
}

func TestForgotAndResetPassword(t *testing.T) {
	svc, users, refresh, _, _, _, mail := newAuthSecurityFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	// User đăng nhập trước để có refresh token đang hoạt động.
	_, err := svc.Login(ctx, "lt@gm.vn", testPassword, "", ClientInfo{IP: "127.0.0.1"})
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
		BranchID: authIntP(1),
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
		Email:        "quay@gm.vn",
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
		Phone:    "0987654321",
		Email:    "quay@gm.vn",
		OTP:      "000000",
		Password: "PortalPass@123",
	}, ClientInfo{})
	if otpCode != "000000" {
		assert.Equal(t, rescode.ResetTokenInvalid.Code(), codeOf(err))
	}

	// Nhập đúng OTP → liên kết tài khoản và trả TokenRes.
	tokenRes, err := svc.VerifyMemberOTP(ctx, &v1.VerifyMemberOTPReq{
		Phone:    "0987654321",
		OTP:      otpCode,
		Password: "PortalPass@123",
	}, ClientInfo{IP: "127.0.0.1"})
	require.NoError(t, err)
	assert.NotEmpty(t, tokenRes.AccessToken)
	assert.Equal(t, "quay@gm.vn", tokenRes.User.Email)
	assert.NotNil(t, memReg.members[50].UserID)

	// Đăng nhập bằng mật khẩu vừa tạo.
	loginRes, err := svc.Login(ctx, "quay@gm.vn", "PortalPass@123", "", ClientInfo{IP: "127.0.0.1"})
	require.NoError(t, err)
	assert.Equal(t, tokenRes.User.ID, loginRes.User.ID)
}

// ---- Hồi quy lỗi QA ---------------------------------------------------------------

func TestRegisterMember_ExistingProfile_OTPGoesToStoredEmail(t *testing.T) {
	svc, users, _, _, _, memReg, mail := newAuthSecurityFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	memReg.members[60] = &entity.Member{
		BaseEntity: entity.BaseEntity{ID: 60}, FullName: "Nạn Nhân", Phone: "0901111222",
		Email: "nannhan@gm.vn", HomeBranchID: 1, Status: entity.MemberStatusActive,
	}
	// Kẻ tấn công biết SĐT, nhập email của mình.
	res, err := svc.RegisterMember(ctx, &v1.RegisterMemberReq{
		FullName: "Hacker", Phone: "0901111222", Email: "hacker@evil.vn", Password: "Password@123",
	}, ClientInfo{})
	require.NoError(t, err)
	assert.True(t, res.RequiresOTP)
	require.Len(t, mail.sent, 1)
	assert.Equal(t, []string{"nannhan@gm.vn"}, mail.sent[0].To, "OTP phải gửi tới email trong hồ sơ, không phải email trong request")
	assert.NotContains(t, res.Message, "nannhan@gm.vn", "không lộ nguyên email hồ sơ")

	// Không có OTP → đoán sai 5 lần → OTP bị vô hiệu.
	for i := 0; i < 4; i++ {
		_, err = svc.VerifyMemberOTP(ctx, &v1.VerifyMemberOTPReq{Phone: "0901111222", Email: "hacker@evil.vn", OTP: "999999", Password: "Hacker123"}, ClientInfo{})
		assert.Equal(t, rescode.ResetTokenInvalid.Code(), codeOf(err))
	}
	_, err = svc.VerifyMemberOTP(ctx, &v1.VerifyMemberOTPReq{Phone: "0901111222", Email: "hacker@evil.vn", OTP: "999999", Password: "Hacker123"}, ClientInfo{})
	assert.Equal(t, rescode.TooManyAttempts.Code(), codeOf(err))

	// Mã đúng nhưng OTP đã bị vô hiệu → vẫn không hợp lệ.
	otp := regexp.MustCompile(`>(\d{6})<`).FindStringSubmatch(mail.sent[0].HTML)[1]
	_, err = svc.VerifyMemberOTP(ctx, &v1.VerifyMemberOTPReq{Phone: "0901111222", Email: "nannhan@gm.vn", OTP: otp, Password: "Owner1234"}, ClientInfo{})
	assert.Equal(t, rescode.ResetTokenInvalid.Code(), codeOf(err))
	assert.Nil(t, memReg.members[60].UserID)

	// Gửi lại ngay lập tức → bị throttle 60 giây.
	_, err = svc.RegisterMember(ctx, &v1.RegisterMemberReq{
		FullName: "Nạn Nhân", Phone: "0901111222", Email: "nannhan@gm.vn", Password: "Password@123",
	}, ClientInfo{})
	assert.Equal(t, rescode.TooManyRequests.Code(), codeOf(err))

	// Sau 61 giây: chủ hồ sơ nhận OTP mới, xác thực → tài khoản dùng email hồ sơ.
	now = now.Add(61 * time.Second)
	_, err = svc.RegisterMember(ctx, &v1.RegisterMemberReq{
		FullName: "Nạn Nhân", Phone: "0901111222", Email: "khac@gm.vn", Password: "Password@123",
	}, ClientInfo{})
	require.NoError(t, err)
	require.Len(t, mail.sent, 2)
	otp = regexp.MustCompile(`>(\d{6})<`).FindStringSubmatch(mail.sent[1].HTML)[1]
	tok, err := svc.VerifyMemberOTP(ctx, &v1.VerifyMemberOTPReq{Phone: "0901111222", Email: "khac@gm.vn", OTP: otp, Password: "Owner1234"}, ClientInfo{})
	require.NoError(t, err)
	assert.Equal(t, "nannhan@gm.vn", tok.User.Email)
	_, err = users.FindByEmail(ctx, "khac@gm.vn")
	assert.Error(t, err, "email trong request không được ghi đè email hồ sơ")
}

func TestRegisterMember_ProfileWithoutEmailAndValidation(t *testing.T) {
	svc, _, _, _, _, memReg, mail := newAuthSecurityFixture(t)
	ctx := context.Background()

	memReg.members[70] = &entity.Member{
		BaseEntity: entity.BaseEntity{ID: 70}, FullName: "Không Email", Phone: "0902222333",
		HomeBranchID: 1, Status: entity.MemberStatusActive,
	}
	_, err := svc.RegisterMember(ctx, &v1.RegisterMemberReq{
		FullName: "X", Phone: "0902222333", Email: "x@gm.vn", Password: "Password@123",
	}, ClientInfo{})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	assert.Empty(t, mail.sent)

	// SĐT không hợp lệ.
	_, err = svc.RegisterMember(ctx, &v1.RegisterMemberReq{
		FullName: "Abc", Phone: "abcdefghij", Email: "abc@gm.vn", Password: "Password@123",
	}, ClientInfo{})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// Chi nhánh không tồn tại / ngừng hoạt động.
	_, err = svc.RegisterMember(ctx, &v1.RegisterMemberReq{
		FullName: "Abc", Phone: "0903333444", Email: "abc@gm.vn", Password: "Password@123", BranchID: authIntP(999999),
	}, ClientInfo{})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// Mật khẩu > 72 byte → 400 (không phải 500).
	_, err = svc.RegisterMember(ctx, &v1.RegisterMemberReq{
		FullName: "Abc", Phone: "0903333444", Email: "abc@gm.vn", Password: strings.Repeat("ậ", 30) + "1", BranchID: authIntP(1),
	}, ClientInfo{})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
}

func TestLogin_PerEmailLockoutAcrossIPs(t *testing.T) {
	svc, _, _, _, _, _, _ := newAuthSecurityFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	for i := 0; i < repository.LoginMaxFailedPerEmail; i++ {
		_, err := svc.Login(ctx, "lt@gm.vn", "wrong-pass", "", ClientInfo{IP: fmt.Sprintf("10.0.%d.1", i)})
		assert.Equal(t, rescode.InvalidCredentials.Code(), codeOf(err))
	}
	_, err := svc.Login(ctx, "lt@gm.vn", testPassword, "", ClientInfo{IP: "10.9.9.9"})
	assert.Equal(t, rescode.TooManyAttempts.Code(), codeOf(err))
}

type fakeLoginMFA struct {
	enabled map[uint]string // userID → mã hợp lệ
}

func (f *fakeLoginMFA) CheckLoginMFA(_ context.Context, userID uint, code string) (bool, bool, error) {
	valid, ok := f.enabled[userID]
	if !ok {
		return false, false, nil
	}
	return true, code != "" && code == valid, nil
}

func TestLogin_MFARequired(t *testing.T) {
	svc, _, _, _, _, _, _ := newAuthSecurityFixture(t)
	svc.mfa = &fakeLoginMFA{enabled: map[uint]string{2: "123456"}}
	ctx := context.Background()

	_, err := svc.Login(ctx, "lt@gm.vn", testPassword, "", ClientInfo{IP: "1.1.1.1"})
	assert.Equal(t, rescode.MFARequired.Code(), codeOf(err))

	_, err = svc.Login(ctx, "lt@gm.vn", testPassword, "000000", ClientInfo{IP: "1.1.1.1"})
	assert.Equal(t, rescode.MFAInvalidCode.Code(), codeOf(err))

	res, err := svc.Login(ctx, "lt@gm.vn", testPassword, "123456", ClientInfo{IP: "1.1.1.1"})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)

	// Tài khoản chưa bật 2FA đăng nhập bình thường.
	_, err = svc.Login(ctx, "owner@gm.vn", testPassword, "", ClientInfo{IP: "1.1.1.1"})
	require.NoError(t, err)
}

func TestForgotPassword_ThrottleAndInvalidateOldTokens(t *testing.T) {
	svc, _, _, resets, _, _, mail := newAuthSecurityFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	_, err := svc.ForgotPassword(ctx, "lt@gm.vn", "WEB", ClientInfo{})
	require.NoError(t, err)
	// Gửi lại ngay → vẫn trả thông điệp chung nhưng không gửi thêm email.
	res, err := svc.ForgotPassword(ctx, "lt@gm.vn", "WEB", ClientInfo{})
	require.NoError(t, err)
	assert.Equal(t, forgotResetMessage, res.Message)
	require.Len(t, mail.sent, 1)

	now = now.Add(61 * time.Second)
	_, err = svc.ForgotPassword(ctx, "lt@gm.vn", "WEB", ClientInfo{})
	require.NoError(t, err)
	require.Len(t, mail.sent, 2)
	assert.Len(t, resets.items, 2)

	re := regexp.MustCompile(`token=([a-f0-9]{64})`)
	tok1 := re.FindStringSubmatch(mail.sent[0].HTML)[1]
	tok2 := re.FindStringSubmatch(mail.sent[1].HTML)[1]
	_, err = svc.ResetPassword(ctx, tok2, "NewPass@123")
	require.NoError(t, err)
	// Token cũ (chưa dùng) cũng bị vô hiệu sau khi reset thành công.
	_, err = svc.ResetPassword(ctx, tok1, "Hacked@123")
	assert.Equal(t, rescode.ResetTokenInvalid.Code(), codeOf(err))
}

func TestLogout_WithoutAccessToken(t *testing.T) {
	svc, _, refresh, _, _, _, _ := newAuthSecurityFixture(t)
	ctx := context.Background()
	res, err := svc.Login(ctx, "lt@gm.vn", testPassword, "", ClientInfo{})
	require.NoError(t, err)
	require.Equal(t, 1, refresh.activeCount(2))

	// Không có access token trong context (đã hết hạn) → vẫn thu hồi được refresh token.
	require.NoError(t, svc.Logout(ctx, res.RefreshToken))
	assert.Equal(t, 0, refresh.activeCount(2))
}

type authAuditEntry struct {
	action string
	id     uint
	actor  *uint
	after  any
}

type authAuditSpy struct {
	entries []authAuditEntry
}

func (a *authAuditSpy) Log(ctx context.Context, action, _ string, id uint, _, after any) {
	a.entries = append(a.entries, authAuditEntry{action, id, reqctx.UserID(ctx), after})
}

func (a *authAuditSpy) actions() []string {
	out := make([]string, 0, len(a.entries))
	for _, e := range a.entries {
		out = append(out, e.action)
	}
	return out
}

func TestAuth_AuditLog(t *testing.T) {
	svc, _, _, _, _, _, mail := newAuthSecurityFixture(t)
	spy := &authAuditSpy{}
	svc.audit = spy
	ctx := context.Background()

	_, err := svc.Login(ctx, "lt@gm.vn", "wrong-pass", "", ClientInfo{IP: "1.1.1.1"})
	require.Error(t, err)
	_, err = svc.Login(ctx, "lt@gm.vn", testPassword, "", ClientInfo{IP: "1.1.1.1"})
	require.NoError(t, err)
	_, err = svc.ForgotPassword(ctx, "lt@gm.vn", "WEB", ClientInfo{})
	require.NoError(t, err)
	rawToken := regexp.MustCompile(`token=([a-f0-9]{64})`).FindStringSubmatch(mail.sent[0].HTML)[1]
	_, err = svc.ResetPassword(ctx, rawToken, "NewPass@123")
	require.NoError(t, err)
	assert.Nil(t, reqctx.Auth(ctx), "context gốc không bị gắn payload")

	assert.Equal(t, []string{AuditLoginFailed, AuditLogin, AuditResetPasswordRequest, AuditResetPassword}, spy.actions())
	for _, e := range spy.entries {
		assert.Equal(t, uint(2), e.id)
		require.NotNil(t, e.actor, "%s: actor_id phải là user liên quan, không NULL", e.action)
		assert.Equal(t, uint(2), *e.actor)
		assert.NotContains(t, fmt.Sprint(e.after), "wrong-pass", "không được ghi mật khẩu vào audit")
	}
}

func authIntP(v int) *int { return &v }

// Hồi quy QA vòng 2: mật khẩu tài khoản liên kết do người nhập đúng OTP đặt (bước verify), không phải người
// gọi register-member gần nhất; email không bắt buộc khi SĐT trùng hồ sơ; branch_id ≤ 0 → 400.
func TestRegisterMember_LinkFlow_PasswordAtVerifyStep(t *testing.T) {
	svc, _, _, _, _, memReg, mail := newAuthSecurityFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	memReg.members[80] = &entity.Member{
		BaseEntity: entity.BaseEntity{ID: 80}, FullName: "Chủ Hồ Sơ", Phone: "0904444555",
		Email: "chuhoso@gm.vn", HomeBranchID: 1, Status: entity.MemberStatusActive,
	}

	// Không gửi email/mật khẩu vẫn nhận OTP; trả email đã che để hiển thị.
	res, err := svc.RegisterMember(ctx, &v1.RegisterMemberReq{FullName: "Chủ Hồ Sơ", Phone: "0904444555"}, ClientInfo{})
	require.NoError(t, err)
	assert.True(t, res.RequiresOTP)
	assert.Equal(t, maskEmail("chuhoso@gm.vn"), res.MaskedEmail)
	assert.NotContains(t, res.MaskedEmail, "chuhoso")

	// Kẻ thứ ba yêu cầu lại OTP (sau 61 giây) kèm mật khẩu của mình → mật khẩu này không được dùng.
	now = now.Add(61 * time.Second)
	_, err = svc.RegisterMember(ctx, &v1.RegisterMemberReq{FullName: "X", Phone: "0904444555", Password: "Attacker123"}, ClientInfo{})
	require.NoError(t, err)
	require.Len(t, mail.sent, 2)
	otp := regexp.MustCompile(`>(\d{6})<`).FindStringSubmatch(mail.sent[1].HTML)[1]

	// Thiếu mật khẩu / mật khẩu yếu ở bước verify → 400, không tốn lượt OTP.
	_, err = svc.VerifyMemberOTP(ctx, &v1.VerifyMemberOTPReq{Phone: "0904444555", OTP: otp}, ClientInfo{})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	_, err = svc.VerifyMemberOTP(ctx, &v1.VerifyMemberOTPReq{Phone: "0904444555", OTP: otp, Password: "chiluchu"}, ClientInfo{})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	tok, err := svc.VerifyMemberOTP(ctx, &v1.VerifyMemberOTPReq{Phone: "0904444555", OTP: otp, Password: "ChuHoSo123"}, ClientInfo{})
	require.NoError(t, err)
	assert.Equal(t, "chuhoso@gm.vn", tok.User.Email)
	_, err = svc.Login(ctx, "chuhoso@gm.vn", "Attacker123", "", ClientInfo{IP: "127.0.0.2"})
	assert.Equal(t, rescode.InvalidCredentials.Code(), codeOf(err), "mật khẩu của kẻ yêu cầu OTP không được dùng")
	_, err = svc.Login(ctx, "chuhoso@gm.vn", "ChuHoSo123", "", ClientInfo{IP: "127.0.0.3"})
	require.NoError(t, err)
}

func TestRegisterMember_NewMemberValidation(t *testing.T) {
	svc, _, _, _, _, _, _ := newAuthSecurityFixture(t)
	ctx := context.Background()
	cases := []struct {
		name string
		req  v1.RegisterMemberReq
	}{
		{"thiếu email khi tạo mới", v1.RegisterMemberReq{FullName: "Abc", Phone: "0905555666", Password: "Password1"}},
		{"thiếu mật khẩu khi tạo mới", v1.RegisterMemberReq{FullName: "Abc", Phone: "0905555666", Email: "abc@gm.vn"}},
		{"mật khẩu không có số", v1.RegisterMemberReq{FullName: "Abc", Phone: "0905555666", Email: "abc@gm.vn", Password: "Password"}},
		{"mật khẩu không có chữ", v1.RegisterMemberReq{FullName: "Abc", Phone: "0905555666", Email: "abc@gm.vn", Password: "12345678"}},
		{"branch_id = 0", v1.RegisterMemberReq{FullName: "Abc", Phone: "0905555666", Email: "abc@gm.vn", Password: "Password1", BranchID: authIntP(0)}},
		{"branch_id âm", v1.RegisterMemberReq{FullName: "Abc", Phone: "0905555666", Email: "abc@gm.vn", Password: "Password1", BranchID: authIntP(-1)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.RegisterMember(ctx, &tc.req, ClientInfo{})
			assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
		})
	}
	// Không gửi branch_id → chi nhánh mặc định.
	res, err := svc.RegisterMember(ctx, &v1.RegisterMemberReq{FullName: "Abc", Phone: "0905555666", Email: "abc@gm.vn", Password: "Password1"}, ClientInfo{})
	require.NoError(t, err)
	require.NotNil(t, res.Token)
}

func TestValidateNewPassword(t *testing.T) {
	for pw, ok := range map[string]bool{
		"Password1": true, "matkhau9": true, "Mậtkhẩu1": true, "1234567a": true,
		"short1": false, "password": false, "12345678": false, "": false,
		strings.Repeat("ậ", 30) + "1": false, // > 72 byte
	} {
		err := ValidateNewPassword(pw)
		assert.Equal(t, ok, err == nil, pw)
		if err != nil {
			assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), pw)
		}
	}
}
