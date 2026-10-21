package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/auth/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
	"gymmaster-api/internal/pkg/mailer"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	passwordResetTTL   = 30 * time.Minute
	memberOTPTTL       = 10 * time.Minute
	forgotResetMessage = "Nếu email tồn tại trong hệ thống, hướng dẫn đặt lại mật khẩu đã được gửi"

	// Chống dò/spam OTP đăng ký hội viên.
	memberOTPMaxAttempts = 5                // nhập sai 5 lần → OTP bị vô hiệu
	memberOTPResendGap   = 60 * time.Second // tối thiểu 60 giây giữa 2 lần gửi OTP cho cùng hồ sơ
	memberOTPMaxPerHour  = 5                // tối đa 5 OTP/giờ cho cùng hồ sơ
	passwordResetGap     = 60 * time.Second // tối thiểu 60 giây giữa 2 email quên mật khẩu cho cùng tài khoản
	passwordResetMaxHour = 5                // tối đa 5 email quên mật khẩu/giờ cho cùng tài khoản
)

// LoginMFAChecker kiểm tra xác thực 2 lớp khi đăng nhập (do PrivacyService cài đặt).
//   - enabled=false: tài khoản chưa bật 2FA.
//   - enabled=true, ok=true: mã TOTP hoặc mã khôi phục hợp lệ (mã khôi phục bị tiêu hủy).
type LoginMFAChecker interface {
	CheckLoginMFA(ctx context.Context, userID uint, code string) (enabled bool, ok bool, err error)
}

// ClientInfo là thông tin thiết bị gửi request, lưu kèm refresh token để audit.
type ClientInfo struct {
	UserAgent string
	IP        string
}

// AuthSecurityDeps gom các phụ thuộc cho quên mật khẩu, chống brute-force và hội viên tự đăng ký.
type AuthSecurityDeps struct {
	PasswordResets repository.PasswordResetRepository
	Limiter        repository.LoginLimiter
	MemberRegister repository.MemberRegisterRepository
	MFA            LoginMFAChecker
	Audit          AuditService
	Mailer         mailer.Mailer
	WebURL         string
	AdminURL       string
}

type (
	AuthService interface {
		// Login: mfaCode chỉ cần khi tài khoản đã bật 2FA (thiếu → 40107 MFA_REQUIRED, sai → 40108).
		Login(ctx context.Context, email, password, mfaCode string, client ClientInfo) (*v1.TokenRes, error)
		Refresh(ctx context.Context, refreshToken string, client ClientInfo) (*v1.TokenRes, error)
		Logout(ctx context.Context, refreshToken string) error
		Me(ctx context.Context) (*v1.UserInfo, error)
		ChangePassword(ctx context.Context, oldPassword, newPassword string) error
		ForgotPassword(ctx context.Context, email, portal string, client ClientInfo) (*v1.ForgotPasswordRes, error)
		ResetPassword(ctx context.Context, rawToken, newPassword string) (*v1.ResetPasswordRes, error)
		RegisterMember(ctx context.Context, req *v1.RegisterMemberReq, client ClientInfo) (*v1.RegisterMemberRes, error)
		VerifyMemberOTP(ctx context.Context, req *v1.VerifyMemberOTPReq, client ClientInfo) (*v1.TokenRes, error)
	}

	authService struct {
		userRepo      repository.UserRepository
		refreshRepo   repository.RefreshTokenRepository
		resetRepo     repository.PasswordResetRepository
		limiter       repository.LoginLimiter
		memberRegRepo repository.MemberRegisterRepository
		mfa           LoginMFAChecker
		audit         AuditService
		mailer        mailer.Mailer
		webURL        string
		adminURL      string
		tokenMaker    token.Maker
		tx            postgres.TxManager
		accessTTL     time.Duration
		refreshTTL    time.Duration
		now           func() time.Time
	}
)

func NewAuthService(
	userRepo repository.UserRepository,
	refreshRepo repository.RefreshTokenRepository,
	tokenMaker token.Maker,
	tx postgres.TxManager,
	accessTTL, refreshTTL time.Duration,
) AuthService {
	return &authService{
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		tokenMaker:  tokenMaker,
		tx:          tx,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
		now:         time.Now,
	}
}

func NewAuthServiceWithSecurity(
	userRepo repository.UserRepository,
	refreshRepo repository.RefreshTokenRepository,
	tokenMaker token.Maker,
	tx postgres.TxManager,
	accessTTL, refreshTTL time.Duration,
	sec AuthSecurityDeps,
) AuthService {
	s := NewAuthService(userRepo, refreshRepo, tokenMaker, tx, accessTTL, refreshTTL).(*authService)
	s.SetSecurity(sec)
	return s
}

func (s *authService) SetSecurity(sec AuthSecurityDeps) {
	s.resetRepo = sec.PasswordResets
	s.limiter = sec.Limiter
	s.memberRegRepo = sec.MemberRegister
	s.mfa = sec.MFA
	s.audit = sec.Audit
	s.mailer = sec.Mailer
	s.webURL = strings.TrimRight(sec.WebURL, "/")
	s.adminURL = strings.TrimRight(sec.AdminURL, "/")
}

var errRefreshNotActive = errors.New("refresh token not active")

// Hành động audit của module xác thực (entity "auth", entity_id = user id). Không bao giờ ghi mật khẩu/mã OTP.
const (
	auditEntityAuth           = "auth"
	AuditLogin                = "LOGIN"
	AuditLoginFailed          = "LOGIN_FAILED"
	AuditChangePassword       = "CHANGE_PASSWORD"
	AuditResetPassword        = "RESET_PASSWORD"
	AuditResetPasswordRequest = "RESET_PASSWORD_REQUEST"
)

// auditAuth ghi audit cho các thao tác xác thực chưa có access token (đăng nhập, quên/đặt lại mật khẩu):
// actor_id = tài khoản liên quan (khi biết) thay vì NULL, để lọc nhật ký theo người dùng được.
func (s *authService) auditAuth(ctx context.Context, action string, userID uint, after any) {
	if userID > 0 && reqctx.Auth(ctx) == nil {
		ctx = reqctx.WithAuth(ctx, &token.Payload{UserID: userID})
	}
	auditLogSafe(s.audit, ctx, action, auditEntityAuth, userID, nil, after)
}

// auditLoginFailed ghi lần đăng nhập thất bại: chỉ email, IP và lý do (userID = 0 khi email không tồn tại).
func (s *authService) auditLoginFailed(ctx context.Context, userID uint, email, reason string, client ClientInfo) {
	s.auditAuth(ctx, AuditLoginFailed, userID, map[string]any{"email": email, "ip": client.IP, "reason": reason})
}

func (s *authService) Login(ctx context.Context, email, password, mfaCode string, client ClientInfo) (*v1.TokenRes, error) {
	now := s.now()
	email = strings.ToLower(strings.TrimSpace(email))
	if s.limiter != nil {
		locked, err := s.limiter.IsLocked(ctx, email, client.IP, now)
		if err != nil {
			return nil, err
		}
		if locked {
			s.auditLoginFailed(ctx, 0, email, "TOO_MANY_ATTEMPTS", client)
			return nil, rescode.New(rescode.TooManyAttempts, "Tài khoản tạm khóa 15 phút do đăng nhập sai quá nhiều lần")
		}
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Vẫn so khớp bcrypt với chuỗi giả để thời gian phản hồi như nhau (chống dò email).
			hash.CheckPassword(dummyHash, password)
			if s.limiter != nil {
				_ = s.limiter.RecordAttempt(ctx, email, client.IP, false, now)
			}
			s.auditLoginFailed(ctx, 0, email, "UNKNOWN_EMAIL", client)
			return nil, rescode.New(rescode.InvalidCredentials, "Email hoặc mật khẩu không đúng")
		}
		return nil, err
	}
	if !hash.CheckPassword(user.PasswordHash, password) {
		if s.limiter != nil {
			_ = s.limiter.RecordAttempt(ctx, email, client.IP, false, now)
		}
		s.auditLoginFailed(ctx, user.ID, email, "WRONG_PASSWORD", client)
		return nil, rescode.New(rescode.InvalidCredentials, "Email hoặc mật khẩu không đúng")
	}
	if user.Status != consts.StatusActive {
		s.auditLoginFailed(ctx, user.ID, email, "ACCOUNT_LOCKED", client)
		return nil, rescode.New(rescode.AccountLocked, "Tài khoản đã bị khóa, vui lòng liên hệ quản trị viên")
	}

	// Bước 2: xác thực 2 lớp (TOTP hoặc mã khôi phục) nếu tài khoản đã bật 2FA.
	if s.mfa != nil {
		code := strings.TrimSpace(mfaCode)
		enabled, ok, err := s.mfa.CheckLoginMFA(ctx, user.ID, code)
		if err != nil {
			return nil, err
		}
		if enabled && code == "" {
			// Mật khẩu đúng nhưng chưa có mã 2FA: không tính là sai, cũng không tính là thành công.
			return nil, rescode.New(rescode.MFARequired, "Tài khoản đã bật xác thực 2 lớp, vui lòng nhập mã 6 số từ ứng dụng xác thực hoặc mã khôi phục")
		}
		if enabled && !ok {
			// Mã 2FA sai được tính vào bộ đếm khóa đăng nhập để chặn dò mã.
			if s.limiter != nil {
				_ = s.limiter.RecordAttempt(ctx, email, client.IP, false, now)
			}
			s.auditLoginFailed(ctx, user.ID, email, "MFA_INVALID_CODE", client)
			return nil, rescode.New(rescode.MFAInvalidCode, "Mã xác thực 2 lớp hoặc mã khôi phục không đúng")
		}
	}

	if s.limiter != nil {
		_ = s.limiter.RecordAttempt(ctx, email, client.IP, true, now)
	}
	res, err := s.issueTokens(ctx, user, client)
	if err != nil {
		return nil, err
	}
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID, now)
	s.auditAuth(ctx, AuditLogin, user.ID,
		map[string]any{"email": user.Email, "ip": client.IP, "user_agent": truncate(client.UserAgent, 255)})
	return res, nil
}

// Refresh xoay vòng refresh token: token cũ bị thu hồi, cấp cặp token mới.
// Nếu một token ĐÃ thu hồi bị dùng lại (dấu hiệu bị đánh cắp) thì thu hồi toàn bộ phiên của user.
func (s *authService) Refresh(ctx context.Context, refreshToken string, client ClientInfo) (*v1.TokenRes, error) {
	tokenHash := hash.SHA256(refreshToken)
	invalid := rescode.New(rescode.RefreshTokenInvalid, "Phiên đăng nhập đã hết hạn, vui lòng đăng nhập lại")

	var res *v1.TokenRes
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		now := s.now()
		old, err := s.refreshRepo.FindActiveForUpdate(ctx, tokenHash, now)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errRefreshNotActive
			}
			return err
		}
		if err := s.refreshRepo.Revoke(ctx, old.ID, now); err != nil {
			return err
		}
		user, err := s.userRepo.FindByID(ctx, old.UserID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return invalid
			}
			return err
		}
		if user.Status != consts.StatusActive {
			return rescode.New(rescode.AccountLocked, "Tài khoản đã bị khóa, vui lòng liên hệ quản trị viên")
		}
		res, err = s.issueTokens(ctx, user, client)
		return err
	})

	if errors.Is(err, errRefreshNotActive) {
		// Chỉ coi là tấn công dùng lại (token reuse) khi token đã từng được xoay vòng trong Refresh
		// (có cả RevokedAt và LastUsedAt). Phiên bị đăng xuất từ xa (RevokeSession / RevokeOtherSessions / Logout)
		// chỉ có RevokedAt nên khi thiết bị cũ tự refresh sẽ chỉ từ chối thiết bị đó, không làm văng phiên hiện tại.
		if t, findErr := s.refreshRepo.FindByHash(ctx, tokenHash); findErr == nil && t.RevokedAt != nil && t.LastUsedAt != nil {
			_ = s.refreshRepo.RevokeAllByUser(ctx, t.UserID, s.now())
		}
		return nil, invalid
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Logout thu hồi refresh token gửi lên. Không bắt buộc access token còn hạn: sở hữu refresh token
// (chuỗi ngẫu nhiên 256 bit) đã là bằng chứng đủ, nên kể cả khi access token hết hạn/vắng mặt
// thì phiên vẫn bị thu hồi phía server.
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil
	}
	t, err := s.refreshRepo.FindByHash(ctx, hash.SHA256(refreshToken))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if auth := reqctx.Auth(ctx); auth != nil && auth.UserID != t.UserID {
		// Token của người khác: không tiết lộ gì, coi như đã đăng xuất.
		return nil
	}
	if t.RevokedAt != nil {
		return nil
	}
	return s.refreshRepo.RevokeByHash(ctx, t.UserID, t.TokenHash, s.now())
}

func (s *authService) Me(ctx context.Context) (*v1.UserInfo, error) {
	auth := reqctx.Auth(ctx)
	if auth == nil {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	user, err := s.userRepo.FindByID(ctx, auth.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.Unauthorized, "Tài khoản không tồn tại")
		}
		return nil, err
	}
	roles, roleIDs, branchIDs := rolesOf(user)
	perms, err := s.userRepo.PermissionCodes(ctx, roleIDs)
	if err != nil {
		return nil, err
	}
	return toUserInfo(user, roles, perms, branchIDs), nil
}

func (s *authService) ChangePassword(ctx context.Context, oldPassword, newPassword string) error {
	auth := reqctx.Auth(ctx)
	if auth == nil {
		return rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	user, err := s.userRepo.FindByID(ctx, auth.UserID)
	if err != nil {
		return err
	}
	if !hash.CheckPassword(user.PasswordHash, oldPassword) {
		return rescode.New(rescode.BadRequest, "Mật khẩu hiện tại không đúng")
	}
	if err := ValidateNewPassword(newPassword); err != nil {
		return err
	}
	hashed, err := hash.HashPassword(newPassword)
	if err != nil {
		return err
	}
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.userRepo.UpdatePassword(ctx, user.ID, hashed); err != nil {
			return err
		}
		// Link quên mật khẩu đã gửi trước đó không còn dùng được.
		if s.resetRepo != nil {
			if err := s.resetRepo.InvalidateAllForUser(ctx, user.ID, s.now()); err != nil {
				return err
			}
		}
		// Đăng xuất mọi thiết bị khác sau khi đổi mật khẩu.
		return s.refreshRepo.RevokeAllByUser(ctx, user.ID, s.now())
	})
	if err != nil {
		return err
	}
	auditLogSafe(s.audit, ctx, AuditChangePassword, auditEntityAuth, user.ID, nil, map[string]any{"email": user.Email})
	return nil
}

func (s *authService) ForgotPassword(ctx context.Context, email, portal string, client ClientInfo) (*v1.ForgotPasswordRes, error) {
	res := &v1.ForgotPasswordRes{Message: forgotResetMessage}
	if s.resetRepo == nil {
		return res, nil
	}
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return res, nil
		}
		return nil, err
	}
	if user.Status != consts.StatusActive {
		return res, nil
	}

	// Throttle theo tài khoản: vẫn trả cùng thông điệp chung để không lộ email có tồn tại hay không.
	now := s.now()
	if n, err := s.resetRepo.CountSince(ctx, user.ID, now.Add(-passwordResetGap)); err != nil {
		return nil, err
	} else if n > 0 {
		return res, nil
	}
	if n, err := s.resetRepo.CountSince(ctx, user.ID, now.Add(-time.Hour)); err != nil {
		return nil, err
	} else if n >= passwordResetMaxHour {
		return res, nil
	}

	rawToken, err := hash.RandomToken(32)
	if err != nil {
		return nil, err
	}
	err = s.resetRepo.Create(ctx, &entity.PasswordReset{
		UserID:    user.ID,
		TokenHash: hash.SHA256(rawToken),
		ExpiresAt: now.Add(passwordResetTTL),
		IP:        truncate(client.IP, 64),
	})
	if err != nil {
		return nil, err
	}

	baseURL := s.webURL
	if strings.EqualFold(strings.TrimSpace(portal), "ADMIN") && s.adminURL != "" {
		baseURL = s.adminURL
	}
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	resetURL := strings.TrimRight(baseURL, "/") + "/reset-password?token=" + rawToken
	s.auditAuth(ctx, AuditResetPasswordRequest, user.ID,
		map[string]any{"email": user.Email, "ip": client.IP, "portal": strings.ToUpper(strings.TrimSpace(portal))})

	if s.mailer != nil {
		html, renderErr := mailer.Render(mailer.TplPasswordReset, mailer.PasswordResetData{
			FullName:   user.FullName,
			ResetURL:   resetURL,
			TTLMinutes: int(passwordResetTTL.Minutes()),
		})
		if renderErr == nil {
			s.mailer.SendAsync(mailer.Message{
				To:      []string{user.Email},
				Subject: "[GymMaster] Đặt lại mật khẩu",
				HTML:    html,
			})
		}
	}
	return res, nil
}

func (s *authService) ResetPassword(ctx context.Context, rawToken, newPassword string) (*v1.ResetPasswordRes, error) {
	invalid := rescode.New(rescode.ResetTokenInvalid, "Liên kết đặt lại mật khẩu không hợp lệ hoặc đã hết hạn")
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" || s.resetRepo == nil {
		return nil, invalid
	}
	if err := ValidateNewPassword(newPassword); err != nil {
		return nil, err
	}
	hashed, err := hash.HashPassword(newPassword)
	if err != nil {
		return nil, err
	}
	tokenHash := hash.SHA256(rawToken)
	var resetUserID uint
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		now := s.now()
		pr, err := s.resetRepo.FindActiveForUpdate(ctx, tokenHash, now)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return invalid
			}
			return err
		}
		resetUserID = pr.UserID
		if err := s.userRepo.UpdatePassword(ctx, pr.UserID, hashed); err != nil {
			return err
		}
		if err := s.resetRepo.MarkUsed(ctx, pr.ID, now); err != nil {
			return err
		}
		// Vô hiệu mọi link đặt lại mật khẩu khác còn hạn của user.
		if err := s.resetRepo.InvalidateAllForUser(ctx, pr.UserID, now); err != nil {
			return err
		}
		return s.refreshRepo.RevokeAllByUser(ctx, pr.UserID, now)
	})
	if err != nil {
		return nil, err
	}
	s.auditAuth(ctx, AuditResetPassword, resetUserID, map[string]any{"via": "reset_token"})
	return &v1.ResetPasswordRes{Message: "Đặt lại mật khẩu thành công, vui lòng đăng nhập lại"}, nil
}

func (s *authService) RegisterMember(ctx context.Context, req *v1.RegisterMemberReq, client ClientInfo) (*v1.RegisterMemberRes, error) {
	if s.memberRegRepo == nil {
		return nil, rescode.New(rescode.Internal, "Chức năng đăng ký hội viên chưa được cấu hình")
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	phone, ok := NormalizeVNPhone(req.Phone)
	if !ok {
		return nil, rescode.New(rescode.BadRequest, "Số điện thoại không hợp lệ (SĐT Việt Nam 10 số)")
	}
	fullName := strings.TrimSpace(req.FullName)
	if req.BranchID != nil && *req.BranchID <= 0 {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh không hợp lệ")
	}

	member, err := s.memberRegRepo.FindMemberByPhone(ctx, phone)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if member != nil {
		// Liên kết hồ sơ có sẵn: email/mật khẩu trong request bị bỏ qua – mật khẩu đặt ở bước xác thực OTP.
		return s.startMemberLinkOTP(ctx, member)
	}

	if email == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập email")
	}
	if err := ValidateNewPassword(req.Password); err != nil {
		return nil, err
	}

	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existingUser != nil {
		return nil, rescode.New(rescode.Conflict, "Email đã được sử dụng cho tài khoản khác")
	}

	var branchID uint
	if req.BranchID == nil {
		branchID, err = s.memberRegRepo.DefaultBranchID(ctx)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, rescode.New(rescode.BadRequest, "Hệ thống chưa có chi nhánh hoạt động để đăng ký")
			}
			return nil, err
		}
	} else {
		branchID = uint(*req.BranchID)
		active, err := s.memberRegRepo.IsActiveBranch(ctx, branchID)
		if err != nil {
			return nil, err
		}
		if !active {
			return nil, rescode.New(rescode.BadRequest, "Chi nhánh không tồn tại hoặc đã ngừng hoạt động")
		}
	}
	roleID, err := s.memberRegRepo.MemberRoleID(ctx)
	if err != nil {
		return nil, err
	}
	pwHash, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	var tokenRes *v1.TokenRes
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		user := &entity.User{
			Email:        email,
			Phone:        phone,
			PasswordHash: pwHash,
			FullName:     fullName,
			Status:       consts.StatusActive,
		}
		newMember := &entity.Member{
			FullName:     fullName,
			Gender:       entity.MemberGenderOther,
			Phone:        phone,
			Email:        email,
			HomeBranchID: branchID,
			Source:       entity.MemberSourceWebsite,
			Status:       entity.MemberStatusActive,
		}
		if err := s.memberRegRepo.CreateUserAndMember(ctx, user, newMember, roleID); err != nil {
			return mapDuplicate(err, "Email hoặc số điện thoại đã được sử dụng")
		}
		user.UserRoles = []entity.UserRole{{
			UserID: user.ID,
			RoleID: roleID,
			Role:   entity.Role{BaseEntity: entity.BaseEntity{ID: roleID}, Code: consts.RoleMember},
		}}
		tokenRes, err = s.issueTokens(ctx, user, client)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &v1.RegisterMemberRes{
		RequiresOTP: false,
		Message:     "Đăng ký tài khoản hội viên thành công",
		Token:       tokenRes,
	}, nil
}

// startMemberLinkOTP: SĐT trùng hồ sơ hội viên có sẵn (tạo tại quầy) → chỉ chủ hồ sơ mới được liên kết.
// OTP LUÔN gửi tới email đã lưu trong hồ sơ (không phải email trong request) và tài khoản tạo ra dùng chính
// email đó, nên kẻ biết SĐT của người khác không thể chiếm hồ sơ.
// Mật khẩu KHÔNG lưu ở bước này: nếu lưu, người thứ ba biết SĐT có thể yêu cầu lại OTP (ghi đè mật khẩu)
// ngay trước khi chủ hồ sơ nhập mã, và tài khoản tạo ra sẽ mang mật khẩu của kẻ đó.
func (s *authService) startMemberLinkOTP(ctx context.Context, member *entity.Member) (*v1.RegisterMemberRes, error) {
	if member.UserID != nil {
		return nil, rescode.New(rescode.Conflict, "Số điện thoại này đã có tài khoản hội viên. Vui lòng đăng nhập hoặc dùng chức năng quên mật khẩu")
	}
	ownerEmail := strings.ToLower(strings.TrimSpace(member.Email))
	if ownerEmail == "" {
		return nil, rescode.New(rescode.BadRequest, "Số điện thoại đã có hồ sơ tại phòng tập nhưng chưa đăng ký email. Vui lòng liên hệ quầy lễ tân để được tạo tài khoản")
	}
	if u, err := s.userRepo.FindByEmail(ctx, ownerEmail); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	} else if u != nil {
		return nil, rescode.New(rescode.Conflict, "Email của hồ sơ hội viên đã được dùng cho tài khoản khác. Vui lòng liên hệ quầy lễ tân")
	}

	now := s.now()
	if n, err := s.memberRegRepo.CountOTPsSince(ctx, member.ID, now.Add(-memberOTPResendGap)); err != nil {
		return nil, err
	} else if n > 0 {
		return nil, rescode.New(rescode.TooManyRequests, "Mã OTP vừa được gửi, vui lòng đợi 60 giây trước khi yêu cầu mã mới")
	}
	if n, err := s.memberRegRepo.CountOTPsSince(ctx, member.ID, now.Add(-time.Hour)); err != nil {
		return nil, err
	} else if n >= memberOTPMaxPerHour {
		return nil, rescode.New(rescode.TooManyRequests, "Bạn đã yêu cầu quá nhiều mã OTP, vui lòng thử lại sau 1 giờ")
	}

	otpCode, err := randomSixDigitOTP()
	if err != nil {
		return nil, err
	}
	if err := s.memberRegRepo.CreateOTP(ctx, &entity.MemberRegisterOTP{
		MemberID:     member.ID,
		Email:        ownerEmail,
		PasswordHash: "", // mật khẩu được gửi ở bước verify-member-otp
		OTPHash:      hash.SHA256(otpCode),
		ExpiresAt:    now.Add(memberOTPTTL),
	}); err != nil {
		return nil, err
	}
	if s.mailer != nil {
		html, renderErr := mailer.Render(mailer.TplMemberRegisterOTP, mailer.MemberOTPData{
			FullName:   member.FullName,
			OTP:        otpCode,
			TTLMinutes: int(memberOTPTTL.Minutes()),
		})
		if renderErr == nil {
			s.mailer.SendAsync(mailer.Message{
				To:      []string{ownerEmail},
				Subject: "[GymMaster] Mã xác thực tài khoản hội viên",
				HTML:    html,
			})
		}
	}
	return &v1.RegisterMemberRes{
		RequiresOTP: true,
		MaskedEmail: maskEmail(ownerEmail),
		Message: fmt.Sprintf("Số điện thoại đã có hồ sơ tại phòng tập. Mã OTP đã được gửi tới email đăng ký trong hồ sơ (%s); "+
			"tài khoản sẽ đăng nhập bằng email này. Nếu email không đúng, vui lòng liên hệ quầy lễ tân.", maskEmail(ownerEmail)),
	}, nil
}

func (s *authService) VerifyMemberOTP(ctx context.Context, req *v1.VerifyMemberOTPReq, client ClientInfo) (*v1.TokenRes, error) {
	if s.memberRegRepo == nil {
		return nil, rescode.New(rescode.Internal, "Chức năng đăng ký hội viên chưa được cấu hình")
	}
	invalidOTP := rescode.New(rescode.ResetTokenInvalid, "Mã OTP không đúng hoặc đã hết hạn")

	phone, ok := NormalizeVNPhone(req.Phone)
	if !ok {
		return nil, invalidOTP
	}
	// Mật khẩu do người nhập đúng OTP (chủ hộp thư trong hồ sơ) đặt – không lấy từ bước register-member.
	if err := ValidateNewPassword(req.Password); err != nil {
		return nil, err
	}
	pwHash, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	member, err := s.memberRegRepo.FindMemberByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, invalidOTP
		}
		return nil, err
	}
	if member.UserID != nil {
		return nil, rescode.New(rescode.Conflict, "Số điện thoại này đã có tài khoản hội viên")
	}
	// Email tài khoản = email trong hồ sơ hội viên (nơi đã nhận OTP), không lấy từ request.
	ownerEmail := strings.ToLower(strings.TrimSpace(member.Email))
	if ownerEmail == "" {
		return nil, invalidOTP
	}

	now := s.now()
	otpRecord, err := s.memberRegRepo.FindLatestActiveOTP(ctx, member.ID, ownerEmail, now)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, invalidOTP
		}
		return nil, err
	}
	if otpRecord.OTPHash != hash.SHA256(strings.TrimSpace(req.OTP)) {
		attempts, err := s.memberRegRepo.RecordOTPFailure(ctx, otpRecord.ID, memberOTPMaxAttempts, now)
		if err != nil {
			return nil, err
		}
		if attempts >= memberOTPMaxAttempts {
			return nil, rescode.New(rescode.TooManyAttempts, "Nhập sai OTP quá 5 lần, mã đã bị vô hiệu. Vui lòng yêu cầu mã mới")
		}
		return nil, invalidOTP
	}

	existingUser, err := s.userRepo.FindByEmail(ctx, ownerEmail)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existingUser != nil {
		return nil, rescode.New(rescode.Conflict, "Email đã được sử dụng cho tài khoản khác")
	}

	roleID, err := s.memberRegRepo.MemberRoleID(ctx)
	if err != nil {
		return nil, err
	}

	var tokenRes *v1.TokenRes
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		user := &entity.User{
			Email:        ownerEmail,
			Phone:        member.Phone,
			PasswordHash: pwHash,
			FullName:     member.FullName,
			AvatarURL:    member.AvatarURL,
			Status:       consts.StatusActive,
		}
		if err := s.memberRegRepo.LinkUserToMember(ctx, member.ID, user, roleID); err != nil {
			return mapDuplicate(err, "Email đã được sử dụng cho tài khoản khác")
		}
		if err := s.memberRegRepo.MarkOTPUsed(ctx, otpRecord.ID, now); err != nil {
			return err
		}
		user.UserRoles = []entity.UserRole{{
			UserID: user.ID,
			RoleID: roleID,
			Role:   entity.Role{BaseEntity: entity.BaseEntity{ID: roleID}, Code: consts.RoleMember},
		}}
		tokenRes, err = s.issueTokens(ctx, user, client)
		return err
	})
	if err != nil {
		return nil, err
	}
	return tokenRes, nil
}

// passwordPolicyMessage: chính sách cho mật khẩu MỚI (tạo/đặt lại/đổi/đăng ký). Mật khẩu cũ đang dùng
// không bị kiểm tra lại nên tài khoản hiện có vẫn đăng nhập bình thường.
const passwordPolicyMessage = "Mật khẩu phải có ít nhất 8 ký tự, gồm cả chữ và số"

// ValidateNewPassword: ≥ 8 ký tự, có ít nhất một chữ cái và một chữ số, tối đa 72 byte (giới hạn bcrypt).
func ValidateNewPassword(password string) error {
	if utf8.RuneCountInString(password) < 8 {
		return rescode.New(rescode.BadRequest, passwordPolicyMessage)
	}
	hasLetter, hasDigit := false, false
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return rescode.New(rescode.BadRequest, passwordPolicyMessage)
	}
	return hash.ValidatePasswordLength(password)
}

// maskEmail che bớt email để hiển thị: nguyenvana@gmail.com → ng*******@gmail.com.
func maskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "***"
	}
	local := []rune(email[:at])
	keep := 2
	if len(local) <= 2 {
		keep = 1
	}
	return string(local[:keep]) + strings.Repeat("*", min(max(len(local)-keep, 3), 6)) + email[at:]
}

func randomSixDigitOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s *authService) issueTokens(ctx context.Context, user *entity.User, client ClientInfo) (*v1.TokenRes, error) {
	roles, roleIDs, branchIDs := rolesOf(user)
	perms, err := s.userRepo.PermissionCodes(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	refresh, err := hash.RandomToken(32)
	if err != nil {
		return nil, err
	}
	// Tạo refresh token (phiên) trước để gắn id phiên vào access token (claim "sid").
	rt := &entity.RefreshToken{
		UserID:    user.ID,
		TokenHash: hash.SHA256(refresh),
		ExpiresAt: s.now().Add(s.refreshTTL),
		UserAgent: truncate(client.UserAgent, 255),
		IP:        truncate(client.IP, 64),
	}
	if err := s.refreshRepo.Create(ctx, rt); err != nil {
		return nil, err
	}
	access, _, err := s.tokenMaker.CreateSessionToken(user.ID, rt.ID, roles, perms, branchIDs)
	if err != nil {
		return nil, err
	}

	return &v1.TokenRes{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(s.accessTTL.Seconds()),
		User:         toUserInfo(user, roles, perms, branchIDs),
	}, nil
}

// rolesOf trả mã role, id role và danh sách chi nhánh (không trùng) của user.
// Chỉ cần một role toàn hệ thống (BranchID = nil) thì branchIDs rỗng = làm việc mọi chi nhánh.
func rolesOf(user *entity.User) (roles []string, roleIDs []uint, branchIDs []uint) {
	global := false
	for _, ur := range user.UserRoles {
		if !slices.Contains(roles, ur.Role.Code) {
			roles = append(roles, ur.Role.Code)
			roleIDs = append(roleIDs, ur.RoleID)
		}
		if ur.BranchID == nil {
			global = true
		} else if !slices.Contains(branchIDs, *ur.BranchID) {
			branchIDs = append(branchIDs, *ur.BranchID)
		}
	}
	if roles == nil {
		roles = []string{}
	}
	if global || branchIDs == nil {
		branchIDs = []uint{}
	}
	return roles, roleIDs, branchIDs
}

func toUserInfo(u *entity.User, roles, perms []string, branchIDs []uint) *v1.UserInfo {
	return &v1.UserInfo{
		ID:          u.ID,
		Email:       u.Email,
		FullName:    u.FullName,
		Phone:       u.Phone,
		AvatarURL:   u.AvatarURL,
		Roles:       roles,
		Permissions: perms,
		BranchIDs:   branchIDs,
	}
}

// truncate cắt theo ký tự (rune) để không làm vỡ chuỗi UTF-8 (cột varchar đếm theo ký tự).
func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// dummyHash là bcrypt hợp lệ của một chuỗi bất kỳ, chỉ dùng để cân bằng thời gian khi email không tồn tại.
var dummyHash, _ = hash.HashPassword("gymmaster-dummy-password")
