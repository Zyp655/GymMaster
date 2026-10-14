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
)

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
	Mailer         mailer.Mailer
	WebURL         string
	AdminURL       string
}

type (
	AuthService interface {
		Login(ctx context.Context, email, password string, client ClientInfo) (*v1.TokenRes, error)
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
	s.mailer = sec.Mailer
	s.webURL = strings.TrimRight(sec.WebURL, "/")
	s.adminURL = strings.TrimRight(sec.AdminURL, "/")
}

var errRefreshNotActive = errors.New("refresh token not active")

func (s *authService) Login(ctx context.Context, email, password string, client ClientInfo) (*v1.TokenRes, error) {
	now := s.now()
	if s.limiter != nil {
		locked, err := s.limiter.IsLocked(ctx, email, client.IP, now)
		if err != nil {
			return nil, err
		}
		if locked {
			return nil, rescode.New(rescode.TooManyAttempts, "Tài khoản tạm khóa 15 phút do đăng nhập sai quá 5 lần")
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
			return nil, rescode.New(rescode.InvalidCredentials, "Email hoặc mật khẩu không đúng")
		}
		return nil, err
	}
	if !hash.CheckPassword(user.PasswordHash, password) {
		if s.limiter != nil {
			_ = s.limiter.RecordAttempt(ctx, email, client.IP, false, now)
		}
		return nil, rescode.New(rescode.InvalidCredentials, "Email hoặc mật khẩu không đúng")
	}
	if user.Status != consts.StatusActive {
		return nil, rescode.New(rescode.AccountLocked, "Tài khoản đã bị khóa, vui lòng liên hệ quản trị viên")
	}

	if s.limiter != nil {
		_ = s.limiter.RecordAttempt(ctx, email, client.IP, true, now)
	}
	res, err := s.issueTokens(ctx, user, client)
	if err != nil {
		return nil, err
	}
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID, now)
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
		if t, findErr := s.refreshRepo.FindByHash(ctx, tokenHash); findErr == nil && t.RevokedAt != nil {
			_ = s.refreshRepo.RevokeAllByUser(ctx, t.UserID, s.now())
		}
		return nil, invalid
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	auth := reqctx.Auth(ctx)
	if auth == nil || refreshToken == "" {
		return nil
	}
	return s.refreshRepo.RevokeByHash(ctx, auth.UserID, hash.SHA256(refreshToken), s.now())
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
	hashed, err := hash.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.userRepo.UpdatePassword(ctx, user.ID, hashed); err != nil {
			return err
		}
		// Đăng xuất mọi thiết bị khác sau khi đổi mật khẩu.
		return s.refreshRepo.RevokeAllByUser(ctx, user.ID, s.now())
	})
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

	rawToken, err := hash.RandomToken(32)
	if err != nil {
		return nil, err
	}
	now := s.now()
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
	hashed, err := hash.HashPassword(newPassword)
	if err != nil {
		return nil, err
	}
	tokenHash := hash.SHA256(rawToken)
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		now := s.now()
		pr, err := s.resetRepo.FindActiveForUpdate(ctx, tokenHash, now)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return invalid
			}
			return err
		}
		if err := s.userRepo.UpdatePassword(ctx, pr.UserID, hashed); err != nil {
			return err
		}
		if err := s.resetRepo.MarkUsed(ctx, pr.ID, now); err != nil {
			return err
		}
		return s.refreshRepo.RevokeAllByUser(ctx, pr.UserID, now)
	})
	if err != nil {
		return nil, err
	}
	return &v1.ResetPasswordRes{Message: "Đặt lại mật khẩu thành công, vui lòng đăng nhập lại"}, nil
}

func (s *authService) RegisterMember(ctx context.Context, req *v1.RegisterMemberReq, client ClientInfo) (*v1.RegisterMemberRes, error) {
	if s.memberRegRepo == nil {
		return nil, rescode.New(rescode.Internal, "Chức năng đăng ký hội viên chưa được cấu hình")
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	phone := strings.TrimSpace(req.Phone)
	if norm, ok := NormalizeVNPhone(phone); ok {
		phone = norm
	}
	fullName := strings.TrimSpace(req.FullName)

	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existingUser != nil {
		return nil, rescode.New(rescode.Conflict, "Email đã được sử dụng cho tài khoản khác")
	}

	pwHash, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	member, err := s.memberRegRepo.FindMemberByPhone(ctx, phone)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if member != nil {
		if member.UserID != nil {
			return nil, rescode.New(rescode.Conflict, "Số điện thoại này đã có tài khoản hội viên")
		}
		otpCode, err := randomSixDigitOTP()
		if err != nil {
			return nil, err
		}
		now := s.now()
		if err := s.memberRegRepo.CreateOTP(ctx, &entity.MemberRegisterOTP{
			MemberID:     member.ID,
			Email:        email,
			PasswordHash: pwHash,
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
					To:      []string{email},
					Subject: "[GymMaster] Mã xác thực tài khoản hội viên",
					HTML:    html,
				})
			}
		}
		return &v1.RegisterMemberRes{
			RequiresOTP: true,
			Message:     "Số điện thoại đã có hồ sơ tại phòng tập. Vui lòng nhập mã OTP vừa gửi tới email để liên kết tài khoản.",
		}, nil
	}

	branchID := req.BranchID
	if branchID == 0 {
		branchID, err = s.memberRegRepo.DefaultBranchID(ctx)
		if err != nil {
			return nil, err
		}
	}
	roleID, err := s.memberRegRepo.MemberRoleID(ctx)
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

func (s *authService) VerifyMemberOTP(ctx context.Context, req *v1.VerifyMemberOTPReq, client ClientInfo) (*v1.TokenRes, error) {
	if s.memberRegRepo == nil {
		return nil, rescode.New(rescode.Internal, "Chức năng đăng ký hội viên chưa được cấu hình")
	}
	invalidOTP := rescode.New(rescode.ResetTokenInvalid, "Mã OTP không hợp lệ hoặc đã hết hạn")

	email := strings.ToLower(strings.TrimSpace(req.Email))
	phone := strings.TrimSpace(req.Phone)
	if norm, ok := NormalizeVNPhone(phone); ok {
		phone = norm
	}

	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existingUser != nil {
		return nil, rescode.New(rescode.Conflict, "Email đã được sử dụng cho tài khoản khác")
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

	now := s.now()
	otpRecord, err := s.memberRegRepo.FindLatestActiveOTP(ctx, member.ID, email, now)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, invalidOTP
		}
		return nil, err
	}
	if otpRecord.OTPHash != hash.SHA256(strings.TrimSpace(req.OTP)) {
		return nil, invalidOTP
	}

	roleID, err := s.memberRegRepo.MemberRoleID(ctx)
	if err != nil {
		return nil, err
	}

	var tokenRes *v1.TokenRes
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		user := &entity.User{
			Email:        email,
			Phone:        member.Phone,
			PasswordHash: otpRecord.PasswordHash,
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

	access, _, err := s.tokenMaker.CreateToken(user.ID, roles, perms, branchIDs)
	if err != nil {
		return nil, err
	}
	refresh, err := hash.RandomToken(32)
	if err != nil {
		return nil, err
	}
	err = s.refreshRepo.Create(ctx, &entity.RefreshToken{
		UserID:    user.ID,
		TokenHash: hash.SHA256(refresh),
		ExpiresAt: s.now().Add(s.refreshTTL),
		UserAgent: truncate(client.UserAgent, 255),
		IP:        truncate(client.IP, 64),
	})
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

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// dummyHash là bcrypt hợp lệ của một chuỗi bất kỳ, chỉ dùng để cân bằng thời gian khi email không tồn tại.
var dummyHash, _ = hash.HashPassword("gymmaster-dummy-password")
