package service

import (
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/member/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Tên entity / hành động trong audit log của module hội viên.
const (
	auditEntityMember     = "member"
	auditEntityBodyMetric = "body_metric"
	AuditGrantPortal      = "GRANT_PORTAL"
	AuditLockPortal       = "LOCK_PORTAL"
	AuditResetQR          = "RESET_QR"
	AuditImport           = "IMPORT"
)

// memberOptionsLimit: số hội viên tối đa trả về cho ô tìm kiếm.
const memberOptionsLimit = 20

type (
	MemberService interface {
		List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error)
		Options(ctx context.Context, keyword string, branchID uint) (v1.OptionsRes, error)
		Get(ctx context.Context, id uint) (*v1.Member, error)
		Create(ctx context.Context, in v1.MemberInput) (*v1.Member, error)
		Update(ctx context.Context, id uint, in v1.MemberInput) (*v1.Member, error)
		Delete(ctx context.Context, id uint) error

		CreatePortalAccount(ctx context.Context, id uint, email, password string) (*v1.PortalAccount, error)
		LockPortalAccount(ctx context.Context, id uint) error
		ResetQR(ctx context.Context, id uint) error

		ListBodyMetrics(ctx context.Context, memberID uint) (v1.ListBodyMetricsRes, error)
		CreateBodyMetric(ctx context.Context, req *v1.CreateBodyMetricReq) (*v1.BodyMetric, error)
		DeleteBodyMetric(ctx context.Context, memberID, metricID uint) error

		ImportTemplate() ([]byte, error)
		Import(ctx context.Context, branchID uint, file io.Reader) (*v1.ImportRes, error)
	}

	memberService struct {
		memberRepo repository.MemberRepository
		metricRepo repository.BodyMetricRepository
		userRepo   repository.UserRepository
		tx         postgres.TxManager
		audit      AuditService
		now        func() time.Time
	}
)

func NewMemberService(
	memberRepo repository.MemberRepository,
	metricRepo repository.BodyMetricRepository,
	userRepo repository.UserRepository,
	tx postgres.TxManager,
	audit AuditService,
) MemberService {
	return &memberService{
		memberRepo: memberRepo,
		metricRepo: metricRepo,
		userRepo:   userRepo,
		tx:         tx,
		audit:      audit,
		now:        time.Now,
	}
}

// ---- Quy tắc dùng chung -------------------------------------------------------

var (
	memberPhoneRegex    = regexp.MustCompile(`^(0(3|5|7|8|9)[0-9]{8}|02[0-9]{9})$`)
	memberPhoneStripper = strings.NewReplacer(" ", "", ".", "", "-", "", "(", "", ")", "")
	memberVNLoc         = memberLoadVNLoc()
)

func memberLoadVNLoc() *time.Location {
	loc, err := time.LoadLocation(consts.TimeZone)
	if err != nil {
		return time.FixedZone("ICT", 7*3600)
	}
	return loc
}

// NormalizeVNPhone chuẩn hóa SĐT Việt Nam về dạng 0xxxxxxxxx.
// Chấp nhận khoảng trắng/dấu chấm/gạch, tiền tố +84 hoặc 84. Trả ok=false nếu không hợp lệ.
func NormalizeVNPhone(raw string) (string, bool) {
	s := memberPhoneStripper.Replace(strings.TrimSpace(raw))
	switch {
	case strings.HasPrefix(s, "+84"):
		s = "0" + s[3:]
	case strings.HasPrefix(s, "84") && len(s) >= 11:
		s = "0" + s[2:]
	}
	if !memberPhoneRegex.MatchString(s) {
		return "", false
	}
	return s, true
}

// memberParseDob đọc ngày sinh "YYYY-MM-DD" (rỗng = không có). Ngày sinh không được ở tương lai.
func memberParseDob(s string, now time.Time) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	d, err := time.Parse(consts.DateLayout, s)
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Ngày sinh phải có dạng YYYY-MM-DD")
	}
	return memberValidateDob(d, now)
}

func memberValidateDob(d time.Time, now time.Time) (*time.Time, error) {
	today := now.In(memberVNLoc).Format(consts.DateLayout)
	if d.Format(consts.DateLayout) > today {
		return nil, rescode.New(rescode.BadRequest, "Ngày sinh không được ở tương lai")
	}
	if d.Year() < 1900 {
		return nil, rescode.New(rescode.BadRequest, "Ngày sinh không hợp lệ")
	}
	return &d, nil
}

func (s *memberService) find(ctx context.Context, id uint) (*entity.Member, error) {
	m, err := s.memberRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy hội viên")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, m.HomeBranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Hội viên thuộc chi nhánh bạn không quản lý")
	}
	return m, nil
}

// checkBranch: chi nhánh phải tồn tại và nằm trong phạm vi của người dùng.
func (s *memberService) checkBranch(ctx context.Context, branchID uint) (*entity.Branch, error) {
	if !reqctx.CanAccessBranch(ctx, branchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được thao tác trên chi nhánh này")
	}
	b, err := s.memberRepo.FindBranch(ctx, branchID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.BadRequest, "Chi nhánh không tồn tại")
		}
		return nil, err
	}
	return b, nil
}

// prepareInput validate + chuẩn hóa input, trả entity đã áp dữ liệu.
func (s *memberService) prepareInput(ctx context.Context, m *entity.Member, in v1.MemberInput, excludeID uint) error {
	phone, ok := NormalizeVNPhone(in.Phone)
	if !ok {
		return rescode.New(rescode.BadRequest, "Số điện thoại không hợp lệ (SĐT Việt Nam 10 số)")
	}
	emergencyPhone := ""
	if strings.TrimSpace(in.EmergencyContactPhone) != "" {
		if emergencyPhone, ok = NormalizeVNPhone(in.EmergencyContactPhone); !ok {
			return rescode.New(rescode.BadRequest, "Số điện thoại người liên hệ không hợp lệ")
		}
	}
	dob, err := memberParseDob(in.Dob, s.now())
	if err != nil {
		return err
	}
	// Đổi chi nhánh (hoặc tạo mới) thì chi nhánh đích phải trong phạm vi.
	if excludeID == 0 || m.HomeBranchID != in.HomeBranchID {
		if _, err := s.checkBranch(ctx, in.HomeBranchID); err != nil {
			return err
		}
	}
	exists, err := s.memberRepo.ExistsPhone(ctx, phone, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return rescode.New(rescode.Conflict, "Số điện thoại đã được dùng cho hội viên khác")
	}

	m.FullName = strings.TrimSpace(in.FullName)
	m.Gender = memberDefault(in.Gender, entity.MemberGenderOther)
	m.Dob = dob
	m.Phone = phone
	m.Email = strings.ToLower(strings.TrimSpace(in.Email))
	m.IDCard = strings.TrimSpace(in.IDCard)
	m.Address = strings.TrimSpace(in.Address)
	if !validImageURLStrict(strings.TrimSpace(in.AvatarURL)) {
		return rescode.New(rescode.BadRequest, "URL ảnh đại diện không hợp lệ (chỉ chấp nhận http(s) hoặc ảnh đã tải lên)")
	}
	m.AvatarURL = strings.TrimSpace(in.AvatarURL)
	m.EmergencyContactName = strings.TrimSpace(in.EmergencyContactName)
	m.EmergencyContactPhone = emergencyPhone
	m.HomeBranchID = in.HomeBranchID
	m.Source = memberDefault(in.Source, entity.MemberSourceWalkIn)
	m.Status = memberDefault(in.Status, entity.MemberStatusActive)
	m.Note = strings.TrimSpace(in.Note)
	if m.FullName == "" {
		return rescode.New(rescode.BadRequest, "Vui lòng nhập họ tên")
	}
	return nil
}

func memberDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// ---- CRUD ----------------------------------------------------------------------

var memberOrderable = []string{"id", "member_code", "full_name", "status", "created_at"}

func (s *memberService) List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem hội viên của chi nhánh này")
	}
	q := req.PageReq.ToQuery(memberOrderable, "id")
	q.OrderBy = "m." + q.OrderBy
	rows, total, err := s.memberRepo.List(ctx, q, repository.MemberListFilter{
		BranchIDs:          branchIDs,
		Status:             req.Status,
		Gender:             req.Gender,
		Source:             req.Source,
		SubscriptionStatus: req.SubscriptionStatus,
	})
	if err != nil {
		return nil, err
	}
	now := s.now()
	return dto.NewPageRes(dto.Map(rows, func(r repository.MemberRow) *v1.MemberItem {
		return toMemberItem(r, now)
	}), total, q), nil
}

func (s *memberService) Options(ctx context.Context, keyword string, branchID uint) (v1.OptionsRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, branchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem hội viên của chi nhánh này")
	}
	items, err := s.memberRepo.Options(ctx, strings.TrimSpace(keyword), branchIDs, memberOptionsLimit)
	if err != nil {
		return nil, err
	}
	out := make(v1.OptionsRes, 0, len(items))
	for _, m := range items {
		out = append(out, &v1.MemberOption{
			ID: m.ID, MemberCode: m.MemberCode, FullName: m.FullName, Phone: m.Phone, AvatarURL: m.AvatarURL, Status: m.Status,
		})
	}
	return out, nil
}

func (s *memberService) Get(ctx context.Context, id uint) (*v1.Member, error) {
	m, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	res := toMemberRes(*m)
	if b, err := s.memberRepo.FindBranch(ctx, m.HomeBranchID); err == nil {
		res.HomeBranch = &v1.BranchRef{ID: b.ID, Code: b.Code, Name: b.Name}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if m.UserID != nil {
		u, err := s.userRepo.FindByID(ctx, *m.UserID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if u != nil {
			res.PortalAccount = toPortalAccount(u)
		}
	}
	latest, err := s.metricRepo.Latest(ctx, id)
	if err != nil {
		return nil, err
	}
	if latest != nil {
		res.LatestMetric = toBodyMetricRes(*latest)
	}
	sub, err := s.memberRepo.ActiveSubscription(ctx, id)
	if err != nil {
		return nil, err
	}
	res.ActiveSubscription = toSubscriptionSummary(sub, s.now())
	stats, err := s.memberRepo.CheckinStats(ctx, id)
	if err != nil {
		return nil, err
	}
	res.Stats = v1.MemberStats{TotalCheckins: stats.Total, LastCheckinAt: stats.LastAt}
	return res, nil
}

func (s *memberService) Create(ctx context.Context, in v1.MemberInput) (*v1.Member, error) {
	m := &entity.Member{}
	if err := s.prepareInput(ctx, m, in, 0); err != nil {
		return nil, err
	}
	m.CreatedBy = reqctx.UserID(ctx)
	m.UpdatedBy = m.CreatedBy
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.memberRepo.Create(ctx, m); err != nil {
			return mapDuplicate(err, "Số điện thoại đã được dùng cho hội viên khác")
		}
		s.audit.Log(ctx, AuditCreate, auditEntityMember, m.ID, nil, toMemberRes(*m))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, m.ID)
}

func (s *memberService) Update(ctx context.Context, id uint, in v1.MemberInput) (*v1.Member, error) {
	m, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	before := toMemberRes(*m)
	if err := s.prepareInput(ctx, m, in, id); err != nil {
		return nil, err
	}
	m.UpdatedBy = reqctx.UserID(ctx)
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.memberRepo.Update(ctx, m); err != nil {
			return mapDuplicate(err, "Số điện thoại đã được dùng cho hội viên khác")
		}
		s.audit.Log(ctx, AuditUpdate, auditEntityMember, id, before, toMemberRes(*m))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *memberService) Delete(ctx context.Context, id uint) error {
	m, err := s.find(ctx, id)
	if err != nil {
		return err
	}
	n, err := s.memberRepo.CountOpenSubscriptions(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return rescode.New(rescode.Conflict, "Không thể xóa hội viên đang có gói tập còn hiệu lực hoặc đang bảo lưu")
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		// Xóa hội viên thì khóa luôn tài khoản cổng hội viên.
		if m.UserID != nil {
			if err := s.lockUser(ctx, *m.UserID); err != nil {
				return err
			}
		}
		if err := s.memberRepo.Delete(ctx, id, reqctx.UserID(ctx)); err != nil {
			return err
		}
		s.audit.Log(ctx, AuditDelete, auditEntityMember, id, toMemberRes(*m), nil)
		return nil
	})
}

// ---- Tài khoản cổng hội viên & QR ---------------------------------------------

func (s *memberService) CreatePortalAccount(ctx context.Context, id uint, email, password string) (*v1.PortalAccount, error) {
	m, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	// Cùng chính sách mật khẩu với nhân viên/đăng ký/đổi/đặt lại (≥ 8 ký tự, có cả chữ và số).
	if err := ValidateNewPassword(password); err != nil {
		return nil, err
	}
	pwHash, err := hash.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Email không được trùng với tài khoản khác.
	other, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if other != nil && (m.UserID == nil || other.ID != *m.UserID) {
		return nil, rescode.New(rescode.Conflict, "Email đã được sử dụng cho tài khoản khác")
	}

	// Đã có tài khoản: đang hoạt động → 409; đang bị khóa → mở lại với email/mật khẩu mới.
	if m.UserID != nil {
		u, err := s.userRepo.FindByID(ctx, *m.UserID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if u != nil {
			if u.Status == consts.StatusActive {
				return nil, rescode.New(rescode.Conflict, "Hội viên đã có tài khoản cổng hội viên")
			}
			err := s.tx.WithTx(ctx, func(ctx context.Context) error {
				if err := s.memberRepo.UpdatePortalUser(ctx, u.ID, map[string]any{
					"email": email, "password_hash": pwHash, "status": consts.StatusActive, "updated_by": reqctx.UserID(ctx),
				}); err != nil {
					return mapDuplicate(err, "Email đã được sử dụng cho tài khoản khác")
				}
				s.audit.Log(ctx, AuditGrantPortal, auditEntityMember, id, nil, map[string]any{"user_id": u.ID, "email": email, "reopened": true})
				return nil
			})
			if err != nil {
				return nil, err
			}
			u.Email, u.Status = email, consts.StatusActive
			return toPortalAccount(u), nil
		}
	}

	u := &entity.User{
		Email:        email,
		Phone:        m.Phone,
		PasswordHash: pwHash,
		FullName:     m.FullName,
		AvatarURL:    m.AvatarURL,
		Status:       consts.StatusActive,
	}
	u.CreatedBy = reqctx.UserID(ctx)
	u.UpdatedBy = u.CreatedBy
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		roleID, err := s.memberRepo.RoleIDByCode(ctx, consts.RoleMember)
		if err != nil {
			return err
		}
		if err := s.userRepo.Create(ctx, u); err != nil {
			return mapDuplicate(err, "Email đã được sử dụng cho tài khoản khác")
		}
		// Role MEMBER không gắn chi nhánh (branch_id NULL).
		if err := s.userRepo.AssignRole(ctx, &entity.UserRole{UserID: u.ID, RoleID: roleID}); err != nil {
			return err
		}
		if err := s.memberRepo.SetUserID(ctx, id, u.ID, reqctx.UserID(ctx)); err != nil {
			return mapDuplicate(err, "Hội viên đã có tài khoản cổng hội viên")
		}
		s.audit.Log(ctx, AuditGrantPortal, auditEntityMember, id, nil, map[string]any{"user_id": u.ID, "email": email})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toPortalAccount(u), nil
}

func (s *memberService) LockPortalAccount(ctx context.Context, id uint) error {
	m, err := s.find(ctx, id)
	if err != nil {
		return err
	}
	if m.UserID == nil {
		return rescode.New(rescode.NotFound, "Hội viên chưa có tài khoản cổng hội viên")
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.lockUser(ctx, *m.UserID); err != nil {
			return err
		}
		s.audit.Log(ctx, AuditLockPortal, auditEntityMember, id, nil, map[string]any{"user_id": *m.UserID})
		return nil
	})
}

// lockUser khóa tài khoản và thu hồi mọi phiên đăng nhập.
func (s *memberService) lockUser(ctx context.Context, userID uint) error {
	if err := s.memberRepo.UpdatePortalUser(ctx, userID, map[string]any{
		"status": consts.StatusLocked, "updated_by": reqctx.UserID(ctx),
	}); err != nil {
		return err
	}
	return s.memberRepo.RevokeUserTokens(ctx, userID, s.now())
}

func (s *memberService) ResetQR(ctx context.Context, id uint) error {
	if _, err := s.find(ctx, id); err != nil {
		return err
	}
	// 32 byte ngẫu nhiên (64 ký tự hex) – cùng độ dài với giá trị mặc định trong DB.
	secret, err := hash.RandomToken(32)
	if err != nil {
		return err
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.memberRepo.SetQRSecret(ctx, id, secret, reqctx.UserID(ctx)); err != nil {
			return err
		}
		s.audit.Log(ctx, AuditResetQR, auditEntityMember, id, nil, nil)
		return nil
	})
}

// ---- Chỉ số cơ thể -------------------------------------------------------------

func (s *memberService) ListBodyMetrics(ctx context.Context, memberID uint) (v1.ListBodyMetricsRes, error) {
	if _, err := s.find(ctx, memberID); err != nil {
		return nil, err
	}
	rows, err := s.metricRepo.ListByMember(ctx, memberID)
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toBodyMetricRes), nil
}

func (s *memberService) CreateBodyMetric(ctx context.Context, req *v1.CreateBodyMetricReq) (*v1.BodyMetric, error) {
	if _, err := s.find(ctx, req.ID); err != nil {
		return nil, err
	}
	now := s.now()
	measuredAt := now
	if req.MeasuredAt != nil && !req.MeasuredAt.IsZero() {
		measuredAt = *req.MeasuredAt
		if measuredAt.After(now.Add(time.Minute)) {
			return nil, rescode.New(rescode.BadRequest, "Thời điểm đo không được ở tương lai")
		}
	}
	m := &entity.BodyMetric{
		MemberID:     req.ID,
		MeasuredAt:   measuredAt,
		WeightKg:     req.WeightKg,
		HeightCm:     req.HeightCm,
		BodyFatPct:   req.BodyFatPct,
		MuscleMassKg: req.MuscleMassKg,
		WaistCm:      req.WaistCm,
		ChestCm:      req.ChestCm,
		HipCm:        req.HipCm,
		Note:         strings.TrimSpace(req.Note),
		MeasuredBy:   reqctx.UserID(ctx),
	}
	if err := s.metricRepo.Create(ctx, m); err != nil {
		if isCheckViolation(err) {
			return nil, rescode.New(rescode.BadRequest, "Chỉ số không hợp lệ")
		}
		return nil, err
	}
	// Đọc lại để có BMI (cột GENERATED) và tên người ghi.
	row, err := s.metricRepo.FindByID(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	return toBodyMetricRes(*row), nil
}

func (s *memberService) DeleteBodyMetric(ctx context.Context, memberID, metricID uint) error {
	if _, err := s.find(ctx, memberID); err != nil {
		return err
	}
	row, err := s.metricRepo.FindByID(ctx, metricID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy lần đo")
		}
		return err
	}
	if row.MemberID != memberID {
		return rescode.New(rescode.NotFound, "Không tìm thấy lần đo")
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.metricRepo.Delete(ctx, metricID); err != nil {
			return err
		}
		s.audit.Log(ctx, AuditDelete, auditEntityBodyMetric, metricID, toBodyMetricRes(*row), nil)
		return nil
	})
}

// ---- Mapping -------------------------------------------------------------------

func memberDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(consts.DateLayout)
	return &s
}

func toSubscriptionSummary(sub *repository.MemberSubscription, now time.Time) *v1.SubscriptionSummary {
	if sub == nil {
		return nil
	}
	today, _ := time.Parse(consts.DateLayout, now.In(memberVNLoc).Format(consts.DateLayout))
	end, _ := time.Parse(consts.DateLayout, sub.EndDate.Format(consts.DateLayout))
	return &v1.SubscriptionSummary{
		ID:                sub.ID,
		PlanID:            sub.PlanID,
		PlanName:          sub.PlanName,
		Status:            sub.Status,
		StartDate:         sub.StartDate.Format(consts.DateLayout),
		EndDate:           sub.EndDate.Format(consts.DateLayout),
		RemainingSessions: sub.RemainingSessions,
		DaysLeft:          int(end.Sub(today).Hours() / 24),
	}
}

func toMemberItem(r repository.MemberRow, now time.Time) *v1.MemberItem {
	m := r.Member
	return &v1.MemberItem{
		ID:                 m.ID,
		MemberCode:         m.MemberCode,
		FullName:           m.FullName,
		Gender:             m.Gender,
		Dob:                memberDatePtr(m.Dob),
		Phone:              m.Phone,
		Email:              m.Email,
		AvatarURL:          m.AvatarURL,
		HomeBranchID:       m.HomeBranchID,
		HomeBranchName:     r.HomeBranchName,
		Source:             m.Source,
		Status:             m.Status,
		HasPortalAccount:   m.UserID != nil,
		ActiveSubscription: toSubscriptionSummary(r.Subscription, now),
		CreatedAt:          m.CreatedAt,
	}
}

func toMemberRes(m entity.Member) *v1.Member {
	return &v1.Member{
		ID:                    m.ID,
		MemberCode:            m.MemberCode,
		FullName:              m.FullName,
		Gender:                m.Gender,
		Dob:                   memberDatePtr(m.Dob),
		Phone:                 m.Phone,
		Email:                 m.Email,
		IDCard:                m.IDCard,
		Address:               m.Address,
		AvatarURL:             m.AvatarURL,
		EmergencyContactName:  m.EmergencyContactName,
		EmergencyContactPhone: m.EmergencyContactPhone,
		HomeBranchID:          m.HomeBranchID,
		Source:                m.Source,
		Status:                m.Status,
		Note:                  m.Note,
		BookingBlockedUntil:   m.BookingBlockedUntil,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}
}

func toPortalAccount(u *entity.User) *v1.PortalAccount {
	return &v1.PortalAccount{UserID: u.ID, Email: u.Email, Status: u.Status, LastLoginAt: u.LastLoginAt}
}

func toBodyMetricRes(r repository.BodyMetricRow) *v1.BodyMetric {
	return &v1.BodyMetric{
		ID:             r.ID,
		MemberID:       r.MemberID,
		MeasuredAt:     r.MeasuredAt,
		WeightKg:       r.WeightKg,
		HeightCm:       r.HeightCm,
		BodyFatPct:     r.BodyFatPct,
		MuscleMassKg:   r.MuscleMassKg,
		WaistCm:        r.WaistCm,
		ChestCm:        r.ChestCm,
		HipCm:          r.HipCm,
		BMI:            r.BMI,
		Note:           r.Note,
		MeasuredBy:     r.MeasuredBy,
		MeasuredByName: r.MeasuredByName,
		CreatedAt:      r.CreatedAt,
	}
}
