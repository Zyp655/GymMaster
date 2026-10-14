package service

import (
	"context"
	"errors"
	"slices"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/auth/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// ClientInfo là thông tin thiết bị gửi request, lưu kèm refresh token để audit.
type ClientInfo struct {
	UserAgent string
	IP        string
}

type (
	AuthService interface {
		Login(ctx context.Context, email, password string, client ClientInfo) (*v1.TokenRes, error)
		Refresh(ctx context.Context, refreshToken string, client ClientInfo) (*v1.TokenRes, error)
		Logout(ctx context.Context, refreshToken string) error
		Me(ctx context.Context) (*v1.UserInfo, error)
		ChangePassword(ctx context.Context, oldPassword, newPassword string) error
	}

	authService struct {
		userRepo    repository.UserRepository
		refreshRepo repository.RefreshTokenRepository
		tokenMaker  token.Maker
		tx          postgres.TxManager
		accessTTL   time.Duration
		refreshTTL  time.Duration
		now         func() time.Time
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

var errRefreshNotActive = errors.New("refresh token not active")

func (s *authService) Login(ctx context.Context, email, password string, client ClientInfo) (*v1.TokenRes, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Vẫn so khớp bcrypt với chuỗi giả để thời gian phản hồi như nhau (chống dò email).
			hash.CheckPassword(dummyHash, password)
			return nil, rescode.New(rescode.InvalidCredentials, "Email hoặc mật khẩu không đúng")
		}
		return nil, err
	}
	if !hash.CheckPassword(user.PasswordHash, password) {
		return nil, rescode.New(rescode.InvalidCredentials, "Email hoặc mật khẩu không đúng")
	}
	if user.Status != consts.StatusActive {
		return nil, rescode.New(rescode.AccountLocked, "Tài khoản đã bị khóa, vui lòng liên hệ quản trị viên")
	}

	res, err := s.issueTokens(ctx, user, client)
	if err != nil {
		return nil, err
	}
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID, s.now())
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
