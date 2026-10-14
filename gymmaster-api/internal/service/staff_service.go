package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/staff/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Chức vụ nhân viên (staff_profiles.position).
const (
	StaffPositionManager      = "MANAGER"
	StaffPositionReceptionist = "RECEPTIONIST"
	StaffPositionTrainer      = "TRAINER"
	StaffPositionOther        = "OTHER"
)

const (
	auditEntityStaff  = "staff"
	auditStaffLock    = "LOCK"
	auditStaffUnlock  = "UNLOCK"
	auditStaffRoles   = "ASSIGN_ROLES"
	auditStaffResetPw = "RESET_PASSWORD"
)

type (
	StaffService interface {
		List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error)
		Get(ctx context.Context, id uint) (*v1.Staff, error)
		Create(ctx context.Context, req *v1.CreateReq) (*v1.Staff, error)
		Update(ctx context.Context, id uint, in v1.StaffInput) (*v1.Staff, error)
		AssignRoles(ctx context.Context, id uint, roles []v1.RoleAssignment) (*v1.Staff, error)
		Lock(ctx context.Context, id uint) (*v1.Staff, error)
		Unlock(ctx context.Context, id uint) (*v1.Staff, error)
		ResetPassword(ctx context.Context, id uint, newPassword string) error
		Delete(ctx context.Context, id uint) error
		Options(ctx context.Context, branchID uint) ([]*v1.StaffOption, error)
		TrainerOptions(ctx context.Context, branchID uint) ([]*v1.TrainerOption, error)
	}

	staffService struct {
		staffRepo   repository.StaffRepository
		roleRepo    repository.RoleRepository
		refreshRepo repository.RefreshTokenRepository
		tx          postgres.TxManager
		audit       AuditService
		now         func() time.Time
	}
)

func NewStaffService(
	staffRepo repository.StaffRepository,
	roleRepo repository.RoleRepository,
	refreshRepo repository.RefreshTokenRepository,
	tx postgres.TxManager,
	audit AuditService,
) StaffService {
	return &staffService{staffRepo: staffRepo, roleRepo: roleRepo, refreshRepo: refreshRepo, tx: tx, audit: audit, now: time.Now}
}

// Cột được phép sắp xếp → cột thật (danh sách có JOIN staff_profiles nên cần tiền tố bảng).
var staffOrderColumns = map[string]string{
	"id":         "users.id",
	"full_name":  "users.full_name",
	"email":      "users.email",
	"status":     "users.status",
	"created_at": "users.created_at",
	"staff_code": "sp.staff_code",
	"position":   "sp.position",
	"hire_date":  "sp.hire_date",
}

func (s *staffService) List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem nhân viên của chi nhánh này")
	}
	orderable := make([]string, 0, len(staffOrderColumns))
	for k := range staffOrderColumns {
		orderable = append(orderable, k)
	}
	q := req.PageReq.ToQuery(orderable, "id")
	q.OrderBy = staffOrderColumns[q.OrderBy]

	items, total, err := s.staffRepo.List(ctx, q, repository.StaffFilter{
		BranchIDs: branchIDs, Position: req.Position, Status: req.Status,
	})
	if err != nil {
		return nil, err
	}
	names, err := s.branchNamesOf(ctx, items...)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Staff, 0, len(items))
	for i := range items {
		out = append(out, toStaffRes(&items[i], names))
	}
	return dto.NewPageRes(out, total, q), nil
}

func (s *staffService) Get(ctx context.Context, id uint) (*v1.Staff, error) {
	u, err := s.findVisible(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toRes(ctx, u)
}

func (s *staffService) Create(ctx context.Context, req *v1.CreateReq) (*v1.Staff, error) {
	in := normalizeStaffInput(req.StaffInput)
	if err := s.checkEmail(ctx, in.Email, 0); err != nil {
		return nil, err
	}
	roles, err := s.resolveRoles(ctx, req.Roles)
	if err != nil {
		return nil, err
	}
	profile, err := buildStaffProfile(in)
	if err != nil {
		return nil, err
	}
	pwHash, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	actor := reqctx.UserID(ctx)
	var userID uint
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		u := &entity.User{Email: in.Email, Phone: in.Phone, FullName: in.FullName, PasswordHash: pwHash, Status: consts.StatusActive}
		u.CreatedBy, u.UpdatedBy = actor, actor
		if err := s.staffRepo.CreateUser(ctx, u); err != nil {
			return mapDuplicate(err, "Email đã được sử dụng")
		}
		userID = u.ID

		code, err := s.staffRepo.NextStaffCode(ctx)
		if err != nil {
			return err
		}
		profile.UserID, profile.StaffCode = u.ID, code
		if err := s.staffRepo.SaveProfile(ctx, profile); err != nil {
			return err
		}
		if in.Position == StaffPositionTrainer {
			if err := s.staffRepo.SaveTrainerProfile(ctx, buildTrainerProfile(u.ID, in.TrainerProfile)); err != nil {
				return err
			}
		}
		if err := s.staffRepo.ReplaceRoles(ctx, u.ID, roles); err != nil {
			return err
		}
		created, err := s.staffRepo.FindByID(ctx, u.ID)
		if err != nil {
			return err
		}
		res, err := s.toRes(ctx, created)
		if err != nil {
			return err
		}
		s.audit.Log(ctx, AuditCreate, auditEntityStaff, u.ID, nil, res)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, userID)
}

func (s *staffService) Update(ctx context.Context, id uint, raw v1.StaffInput) (*v1.Staff, error) {
	u, err := s.findManageable(ctx, id)
	if err != nil {
		return nil, err
	}
	before, err := s.toRes(ctx, u)
	if err != nil {
		return nil, err
	}
	in := normalizeStaffInput(raw)
	if err := s.checkEmail(ctx, in.Email, id); err != nil {
		return nil, err
	}
	profile, err := buildStaffProfile(in)
	if err != nil {
		return nil, err
	}
	profile.UserID = id
	if u.StaffProfile != nil {
		profile.StaffCode = u.StaffProfile.StaffCode
	}

	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		user := u.User
		user.Email, user.Phone, user.FullName = in.Email, in.Phone, in.FullName
		user.UpdatedBy = reqctx.UserID(ctx)
		if err := s.staffRepo.UpdateUser(ctx, &user); err != nil {
			return mapDuplicate(err, "Email đã được sử dụng")
		}
		if profile.StaffCode == "" { // tài khoản cũ chưa có hồ sơ → cấp mã mới
			code, err := s.staffRepo.NextStaffCode(ctx)
			if err != nil {
				return err
			}
			profile.StaffCode = code
		}
		if err := s.staffRepo.SaveProfile(ctx, profile); err != nil {
			return err
		}
		if in.Position == StaffPositionTrainer {
			if err := s.staffRepo.SaveTrainerProfile(ctx, buildTrainerProfile(id, in.TrainerProfile)); err != nil {
				return err
			}
		} else if u.TrainerProfile != nil {
			if err := s.staffRepo.DeleteTrainerProfile(ctx, id); err != nil {
				return err
			}
		}
		after, err := s.getRes(ctx, id)
		if err != nil {
			return err
		}
		s.audit.Log(ctx, AuditUpdate, auditEntityStaff, id, before, after)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *staffService) AssignRoles(ctx context.Context, id uint, in []v1.RoleAssignment) (*v1.Staff, error) {
	if staffIsSelf(ctx, id) {
		return nil, rescode.New(rescode.BadRequest, "Không thể tự thay đổi vai trò của chính mình")
	}
	u, err := s.findManageable(ctx, id)
	if err != nil {
		return nil, err
	}
	newRoles, err := s.resolveRoles(ctx, in)
	if err != nil {
		return nil, err
	}
	before, err := s.toRes(ctx, u)
	if err != nil {
		return nil, err
	}

	// Người không phải OWNER chỉ thay được vai trò trong phạm vi chi nhánh của mình:
	// vai trò toàn hệ thống và vai trò ở chi nhánh khác của nhân viên được giữ nguyên.
	final := []entity.UserRole{}
	if !reqctx.HasRole(ctx, consts.RoleOwner) {
		for _, ur := range u.UserRoles {
			if ur.BranchID == nil || !reqctx.CanAccessBranch(ctx, *ur.BranchID) {
				final = append(final, entity.UserRole{RoleID: ur.RoleID, BranchID: ur.BranchID})
			}
		}
	}
	for _, nr := range newRoles {
		if !staffContainsRole(final, nr) {
			final = append(final, nr)
		}
	}

	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.staffRepo.ReplaceRoles(ctx, id, final); err != nil {
			return err
		}
		after, err := s.getRes(ctx, id)
		if err != nil {
			return err
		}
		s.audit.Log(ctx, auditStaffRoles, auditEntityStaff, id, before.Roles, after.Roles)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *staffService) Lock(ctx context.Context, id uint) (*v1.Staff, error) {
	if staffIsSelf(ctx, id) {
		return nil, rescode.New(rescode.BadRequest, "Không thể tự khóa tài khoản của chính mình")
	}
	u, err := s.findManageable(ctx, id)
	if err != nil {
		return nil, err
	}
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.staffRepo.UpdateStatus(ctx, id, consts.StatusLocked, reqctx.UserID(ctx)); err != nil {
			return err
		}
		if err := s.refreshRepo.RevokeAllByUser(ctx, id, s.now()); err != nil {
			return err
		}
		s.audit.Log(ctx, auditStaffLock, auditEntityStaff, id, map[string]string{"status": u.Status}, map[string]string{"status": consts.StatusLocked})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *staffService) Unlock(ctx context.Context, id uint) (*v1.Staff, error) {
	u, err := s.findManageable(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.staffRepo.UpdateStatus(ctx, id, consts.StatusActive, reqctx.UserID(ctx)); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, auditStaffUnlock, auditEntityStaff, id, map[string]string{"status": u.Status}, map[string]string{"status": consts.StatusActive})
	return s.Get(ctx, id)
}

func (s *staffService) ResetPassword(ctx context.Context, id uint, newPassword string) error {
	if len(newPassword) < 8 {
		return rescode.New(rescode.BadRequest, "Mật khẩu phải có ít nhất 8 ký tự")
	}
	if _, err := s.findManageable(ctx, id); err != nil {
		return err
	}
	pwHash, err := hash.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.staffRepo.UpdatePassword(ctx, id, pwHash, reqctx.UserID(ctx)); err != nil {
			return err
		}
		// Mật khẩu đổi → đăng xuất mọi phiên đang mở.
		if err := s.refreshRepo.RevokeAllByUser(ctx, id, s.now()); err != nil {
			return err
		}
		s.audit.Log(ctx, auditStaffResetPw, auditEntityStaff, id, nil, nil)
		return nil
	})
}

func (s *staffService) Delete(ctx context.Context, id uint) error {
	if staffIsSelf(ctx, id) {
		return rescode.New(rescode.BadRequest, "Không thể tự xóa tài khoản của chính mình")
	}
	u, err := s.findManageable(ctx, id)
	if err != nil {
		return err
	}
	now := s.now()
	classes, pts, err := s.staffRepo.CountFutureAssignments(ctx, id, now)
	if err != nil {
		return err
	}
	if classes > 0 || pts > 0 {
		return rescode.New(rescode.Conflict, fmt.Sprintf(
			"Không thể xóa: nhân viên còn %d lịch lớp và %d buổi PT sắp diễn ra. Hãy chuyển hoặc hủy lịch trước", classes, pts))
	}
	before, err := s.toRes(ctx, u)
	if err != nil {
		return err
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.staffRepo.DeleteFutureShifts(ctx, id, now); err != nil {
			return err
		}
		if err := s.refreshRepo.RevokeAllByUser(ctx, id, now); err != nil {
			return err
		}
		if err := s.staffRepo.SoftDelete(ctx, id, reqctx.UserID(ctx)); err != nil {
			return err
		}
		s.audit.Log(ctx, AuditDelete, auditEntityStaff, id, before, nil)
		return nil
	})
}

func (s *staffService) Options(ctx context.Context, branchID uint) ([]*v1.StaffOption, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, branchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem nhân viên của chi nhánh này")
	}
	items, err := s.staffRepo.ActiveOptions(ctx, branchIDs, false)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.StaffOption, 0, len(items))
	for _, u := range items {
		o := &v1.StaffOption{ID: u.ID, FullName: u.FullName}
		if u.StaffProfile != nil {
			o.StaffCode, o.Position = u.StaffProfile.StaffCode, u.StaffProfile.Position
		}
		out = append(out, o)
	}
	return out, nil
}

func (s *staffService) TrainerOptions(ctx context.Context, branchID uint) ([]*v1.TrainerOption, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, branchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem HLV của chi nhánh này")
	}
	items, err := s.staffRepo.ActiveOptions(ctx, branchIDs, true)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.TrainerOption, 0, len(items))
	for _, u := range items {
		o := &v1.TrainerOption{ID: u.ID, FullName: u.FullName, Level: "JUNIOR", Specialties: []string{}}
		if u.TrainerProfile != nil {
			o.Level = u.TrainerProfile.Level
			o.Specialties = append(o.Specialties, u.TrainerProfile.Specialties...)
		}
		out = append(out, o)
	}
	return out, nil
}

// ---- helpers ----------------------------------------------------------------

func staffIsSelf(ctx context.Context, id uint) bool {
	uid := reqctx.UserID(ctx)
	return uid != nil && *uid == id
}

func (s *staffService) find(ctx context.Context, id uint) (*entity.StaffAccount, error) {
	u, err := s.staffRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy nhân viên")
		}
		return nil, err
	}
	return u, nil
}

// findVisible: người dùng gắn chi nhánh chỉ thấy nhân viên có vai trò tại chi nhánh của mình.
func (s *staffService) findVisible(ctx context.Context, id uint) (*entity.StaffAccount, error) {
	u, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	allowed := reqctx.AllowedBranches(ctx)
	if allowed == nil {
		return u, nil
	}
	for _, ur := range u.UserRoles {
		if ur.BranchID != nil && slices.Contains(allowed, *ur.BranchID) {
			return u, nil
		}
	}
	return nil, rescode.New(rescode.NotFound, "Không tìm thấy nhân viên")
}

// findManageable: ngoài phạm vi chi nhánh, chỉ OWNER mới thao tác được trên tài khoản OWNER.
func (s *staffService) findManageable(ctx context.Context, id uint) (*entity.StaffAccount, error) {
	u, err := s.findVisible(ctx, id)
	if err != nil {
		return nil, err
	}
	if !reqctx.HasRole(ctx, consts.RoleOwner) {
		for _, ur := range u.UserRoles {
			if ur.Role.Code == consts.RoleOwner {
				return nil, rescode.New(rescode.PermissionDenied, "Chỉ chủ trung tâm mới được thao tác trên tài khoản chủ trung tâm")
			}
		}
	}
	return u, nil
}

func (s *staffService) checkEmail(ctx context.Context, email string, excludeID uint) error {
	exists, err := s.staffRepo.EmailExists(ctx, email, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return rescode.New(rescode.Conflict, "Email đã được sử dụng")
	}
	return nil
}

// resolveRoles kiểm tra danh sách vai trò theo quy tắc phân quyền và đổi sang entity.UserRole.
func (s *staffService) resolveRoles(ctx context.Context, in []v1.RoleAssignment) ([]entity.UserRole, error) {
	if len(in) == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng gán ít nhất một vai trò")
	}
	isOwner := reqctx.HasRole(ctx, consts.RoleOwner)
	out := make([]entity.UserRole, 0, len(in))
	var branchIDs []uint

	for _, a := range in {
		code := strings.ToUpper(strings.TrimSpace(a.RoleCode))
		if code == "" {
			return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn vai trò")
		}
		if code == consts.RoleMember {
			return nil, rescode.New(rescode.BadRequest, "Không thể gán vai trò hội viên cho nhân viên")
		}
		role, err := s.roleRepo.FindByCode(ctx, code)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, rescode.New(rescode.BadRequest, fmt.Sprintf("Vai trò %s không tồn tại", code))
			}
			return nil, err
		}
		branchID := a.BranchID
		if branchID != nil && *branchID == 0 {
			branchID = nil
		}
		switch {
		case code == consts.RoleOwner:
			if !isOwner {
				return nil, rescode.New(rescode.PermissionDenied, "Chỉ chủ trung tâm mới được gán vai trò chủ trung tâm")
			}
			branchID = nil // OWNER luôn là vai trò toàn hệ thống
		case branchID == nil:
			if !isOwner {
				return nil, rescode.New(rescode.BadRequest, fmt.Sprintf("Vui lòng chọn chi nhánh cho vai trò %s", role.Name))
			}
		default:
			if !reqctx.CanAccessBranch(ctx, *branchID) {
				return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền gán vai trò tại chi nhánh này")
			}
			if !slices.Contains(branchIDs, *branchID) {
				branchIDs = append(branchIDs, *branchID)
			}
		}
		ur := entity.UserRole{RoleID: role.ID, BranchID: branchID}
		if !staffContainsRole(out, ur) {
			out = append(out, ur)
		}
	}

	if len(branchIDs) > 0 {
		names, err := s.staffRepo.BranchNames(ctx, branchIDs)
		if err != nil {
			return nil, err
		}
		for _, id := range branchIDs {
			if _, ok := names[id]; !ok {
				return nil, rescode.New(rescode.BadRequest, "Chi nhánh không tồn tại")
			}
		}
	}
	return out, nil
}

func staffContainsRole(list []entity.UserRole, ur entity.UserRole) bool {
	for _, x := range list {
		if x.RoleID == ur.RoleID && staffSameBranch(x.BranchID, ur.BranchID) {
			return true
		}
	}
	return false
}

func staffSameBranch(a, b *uint) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func normalizeStaffInput(in v1.StaffInput) v1.StaffInput {
	in.FullName = strings.TrimSpace(in.FullName)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = strings.TrimSpace(in.Phone)
	in.IDCard = strings.TrimSpace(in.IDCard)
	in.Address = strings.TrimSpace(in.Address)
	if in.Gender == "" {
		in.Gender = "OTHER"
	}
	return in
}

// staffParseDate đọc ngày "YYYY-MM-DD" ở dạng UTC 00:00 (cột DATE không mang múi giờ).
func staffParseDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(consts.DateLayout, s)
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Ngày phải có dạng YYYY-MM-DD")
	}
	return &t, nil
}

// staffToday trả ngày hôm nay (giờ Việt Nam) ở dạng UTC 00:00.
func staffToday(now time.Time) time.Time {
	if loc, err := time.LoadLocation(consts.TimeZone); err == nil {
		now = now.In(loc)
	}
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func buildStaffProfile(in v1.StaffInput) (*entity.StaffProfile, error) {
	dob, err := staffParseDate(in.Dob)
	if err != nil {
		return nil, err
	}
	hire, err := staffParseDate(in.HireDate)
	if err != nil {
		return nil, err
	}
	today := staffToday(time.Now())
	if hire == nil {
		hire = &today
	}
	if dob != nil && !dob.Before(today) {
		return nil, rescode.New(rescode.BadRequest, "Ngày sinh phải trước ngày hôm nay")
	}
	return &entity.StaffProfile{
		Position:       in.Position,
		Gender:         in.Gender,
		Dob:            dob,
		IDCard:         in.IDCard,
		Address:        in.Address,
		HireDate:       *hire,
		BaseSalary:     in.BaseSalary,
		CommissionRate: in.CommissionRate,
		Note:           in.Note,
	}, nil
}

func buildTrainerProfile(userID uint, in *v1.TrainerProfileInput) *entity.TrainerProfile {
	p := &entity.TrainerProfile{UserID: userID, Level: "JUNIOR", Specialties: entity.StaffTextArray{}}
	if in == nil {
		return p
	}
	if in.Level != "" {
		p.Level = in.Level
	}
	seen := map[string]bool{}
	for _, sp := range in.Specialties {
		sp = strings.TrimSpace(sp)
		key := strings.ToLower(sp)
		if sp == "" || seen[key] {
			continue
		}
		seen[key] = true
		p.Specialties = append(p.Specialties, sp)
	}
	p.Bio = strings.TrimSpace(in.Bio)
	p.Certifications = strings.TrimSpace(in.Certifications)
	p.YearsExperience = max(in.YearsExperience, 0)
	return p
}

func (s *staffService) branchNamesOf(ctx context.Context, users ...entity.StaffAccount) (map[uint]string, error) {
	var ids []uint
	for _, u := range users {
		for _, ur := range u.UserRoles {
			if ur.BranchID != nil && !slices.Contains(ids, *ur.BranchID) {
				ids = append(ids, *ur.BranchID)
			}
		}
	}
	return s.staffRepo.BranchNames(ctx, ids)
}

func (s *staffService) toRes(ctx context.Context, u *entity.StaffAccount) (*v1.Staff, error) {
	names, err := s.branchNamesOf(ctx, *u)
	if err != nil {
		return nil, err
	}
	return toStaffRes(u, names), nil
}

func (s *staffService) getRes(ctx context.Context, id uint) (*v1.Staff, error) {
	u, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toRes(ctx, u)
}

func toStaffRes(u *entity.StaffAccount, branchNames map[uint]string) *v1.Staff {
	res := &v1.Staff{
		ID:          u.ID,
		Email:       u.Email,
		Phone:       u.Phone,
		FullName:    u.FullName,
		AvatarURL:   u.AvatarURL,
		Status:      u.Status,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
		Roles:       []v1.StaffRole{},
	}
	for _, ur := range u.UserRoles {
		r := v1.StaffRole{RoleID: ur.RoleID, RoleCode: ur.Role.Code, RoleName: ur.Role.Name, BranchID: ur.BranchID}
		if ur.BranchID != nil {
			r.BranchName = branchNames[*ur.BranchID]
		}
		res.Roles = append(res.Roles, r)
	}
	if p := u.StaffProfile; p != nil {
		sp := &v1.StaffProfile{
			StaffCode:      p.StaffCode,
			Position:       p.Position,
			Gender:         p.Gender,
			IDCard:         p.IDCard,
			Address:        p.Address,
			HireDate:       p.HireDate.Format(consts.DateLayout),
			BaseSalary:     p.BaseSalary,
			CommissionRate: p.CommissionRate,
			Note:           p.Note,
		}
		if p.Dob != nil {
			d := p.Dob.Format(consts.DateLayout)
			sp.Dob = &d
		}
		res.StaffProfile = sp
		if p.Position == StaffPositionTrainer && u.TrainerProfile != nil {
			tp := u.TrainerProfile
			res.TrainerProfile = &v1.TrainerProfile{
				Level:           tp.Level,
				Specialties:     append([]string{}, tp.Specialties...),
				Bio:             tp.Bio,
				Certifications:  tp.Certifications,
				YearsExperience: tp.YearsExperience,
			}
		}
	}
	return res
}
