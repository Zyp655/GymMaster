package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/plan/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

type (
	// MembershipPlanService quản lý danh mục gói tập.
	MembershipPlanService interface {
		List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error)
		// Options trả các gói đang bán (ACTIVE) cho dropdown bán gói.
		Options(ctx context.Context) ([]*v1.Plan, error)
		Get(ctx context.Context, id uint) (*v1.Plan, error)
		Create(ctx context.Context, in v1.PlanInput) (*v1.Plan, error)
		Update(ctx context.Context, id uint, in v1.PlanInput) (*v1.Plan, error)
		UpdateStatus(ctx context.Context, id uint, status string) (*v1.Plan, error)
		Delete(ctx context.Context, id uint) error
	}

	membershipPlanService struct {
		repo  repository.MembershipPlanRepository
		audit AuditService
	}
)

func NewMembershipPlanService(repo repository.MembershipPlanRepository, audit AuditService) MembershipPlanService {
	return &membershipPlanService{repo: repo, audit: audit}
}

const auditEntityPlan = "membership_plan"

var planOrderable = []string{"id", "code", "name", "type", "price", "duration_days", "status", "sort_order", "created_at"}

func (s *membershipPlanService) List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error) {
	q := req.PageReq.ToQuery(planOrderable, "sort_order")
	if req.OrderBy == "" { // mặc định theo thứ tự hiển thị tăng dần
		q.OrderDir = "ASC"
	}
	if req.Status != "" {
		q.Filters["status"] = req.Status
	}
	if req.Type != "" {
		q.Filters["type"] = req.Type
	}
	items, total, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(items, toPlanRes), total, q), nil
}

func (s *membershipPlanService) Options(ctx context.Context) ([]*v1.Plan, error) {
	items, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	return dto.Map(items, toPlanRes), nil
}

func (s *membershipPlanService) Get(ctx context.Context, id uint) (*v1.Plan, error) {
	p, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	return toPlanRes(*p), nil
}

func (s *membershipPlanService) Create(ctx context.Context, in v1.PlanInput) (*v1.Plan, error) {
	in, err := normalizePlanInput(in)
	if err != nil {
		return nil, err
	}
	if err := s.checkCode(ctx, in.Code, 0); err != nil {
		return nil, err
	}
	p := &entity.MembershipPlan{}
	applyPlanInput(p, in)
	p.CreatedBy = reqctx.UserID(ctx)
	p.UpdatedBy = p.CreatedBy
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, mapDuplicate(err, "Mã gói tập đã tồn tại")
	}
	res := toPlanRes(*p)
	auditCatalog(ctx, s.audit, AuditCreate, auditEntityPlan, p.ID, nil, res)
	return res, nil
}

func (s *membershipPlanService) Update(ctx context.Context, id uint, in v1.PlanInput) (*v1.Plan, error) {
	p, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	in, err = normalizePlanInput(in)
	if err != nil {
		return nil, err
	}
	if err := s.checkCode(ctx, in.Code, id); err != nil {
		return nil, err
	}
	before := toPlanRes(*p)
	applyPlanInput(p, in)
	p.UpdatedBy = reqctx.UserID(ctx)
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, mapDuplicate(err, "Mã gói tập đã tồn tại")
	}
	res, err := s.Get(ctx, id)
	if err == nil {
		auditCatalog(ctx, s.audit, AuditUpdate, auditEntityPlan, id, before, res)
	}
	return res, err
}

func (s *membershipPlanService) UpdateStatus(ctx context.Context, id uint, status string) (*v1.Plan, error) {
	p, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateStatus(ctx, id, status, reqctx.UserID(ctx)); err != nil {
		return nil, err
	}
	auditCatalog(ctx, s.audit, AuditUpdate, auditEntityPlan, id, map[string]any{"status": p.Status}, map[string]any{"status": status})
	return s.Get(ctx, id)
}

func (s *membershipPlanService) Delete(ctx context.Context, id uint) error {
	p, err := s.find(ctx, id)
	if err != nil {
		return err
	}
	n, err := s.repo.CountSubscriptions(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return rescode.New(rescode.Conflict, "Gói tập đã có hội viên đăng ký nên không thể xóa, hãy chuyển sang trạng thái Ngừng bán")
	}
	if err := s.repo.Delete(ctx, id, reqctx.UserID(ctx)); err != nil {
		return err
	}
	auditCatalog(ctx, s.audit, AuditDelete, auditEntityPlan, id, toPlanRes(*p), nil)
	return nil
}

func (s *membershipPlanService) find(ctx context.Context, id uint) (*entity.MembershipPlan, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy gói tập")
		}
		return nil, err
	}
	return p, nil
}

func (s *membershipPlanService) checkCode(ctx context.Context, code string, excludeID uint) error {
	exists, err := s.repo.ExistsCode(ctx, code, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return rescode.New(rescode.Conflict, "Mã gói tập đã tồn tại")
	}
	return nil
}

// normalizePlanInput kiểm tra các ràng buộc nghiệp vụ (khớp CHECK của migration 000004)
// và chuẩn hóa dữ liệu: gói TIME không có số buổi, tiền làm tròn VND.
func normalizePlanInput(in v1.PlanInput) (v1.PlanInput, error) {
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	if in.Status == "" {
		in.Status = consts.StatusActive
	}
	if in.AccessScope == "" {
		in.AccessScope = entity.AccessSingleBranch
	}
	if in.DurationDays <= 0 {
		return in, rescode.New(rescode.BadRequest, "Thời hạn gói phải lớn hơn 0 ngày")
	}
	switch in.Type {
	case entity.PlanTypeSession:
		if in.SessionCount == nil || *in.SessionCount <= 0 {
			return in, rescode.New(rescode.BadRequest, "Gói theo buổi phải có số buổi lớn hơn 0")
		}
	case entity.PlanTypeTime:
		in.SessionCount = nil
	default:
		return in, rescode.New(rescode.BadRequest, "Loại gói không hợp lệ")
	}
	if in.Price < 0 {
		return in, rescode.New(rescode.BadRequest, "Giá gói không được âm")
	}
	if in.TransferFee < 0 {
		return in, rescode.New(rescode.BadRequest, "Phí chuyển nhượng không được âm")
	}
	if in.MaxFreezeDays < 0 || in.MaxFreezeTimes < 0 {
		return in, rescode.New(rescode.BadRequest, "Số ngày/số lần bảo lưu không được âm")
	}
	if (in.MaxFreezeDays == 0) != (in.MaxFreezeTimes == 0) {
		return in, rescode.New(rescode.BadRequest, "Số ngày bảo lưu và số lần bảo lưu phải cùng bằng 0 (không cho bảo lưu) hoặc cùng lớn hơn 0")
	}
	if in.MaxFreezeDays > in.DurationDays {
		return in, rescode.New(rescode.BadRequest, "Số ngày bảo lưu tối đa không được vượt quá thời hạn gói")
	}
	// Sao chép slice để chuẩn hóa không ghi đè dữ liệu của caller.
	in.AccessHours = append([]v1.AccessHoursSlot(nil), in.AccessHours...)
	for i, sl := range in.AccessHours {
		from, ok1 := parseHHMM(sl.From)
		to, ok2 := parseHHMM(sl.To)
		if !ok1 || !ok2 || from >= to {
			return in, rescode.New(rescode.BadRequest, "Khung giờ vào tập không hợp lệ (HH:MM, giờ bắt đầu phải trước giờ kết thúc)")
		}
		// Chuẩn hóa "8:00" → "08:00" (so khớp khung giờ khi check-in/hiển thị dùng dạng HH:MM).
		in.AccessHours[i].From = fmt.Sprintf("%02d:%02d", from/60, from%60)
		in.AccessHours[i].To = fmt.Sprintf("%02d:%02d", to/60, to%60)
		for _, d := range sl.Dow {
			if d < 0 || d > 7 {
				return in, rescode.New(rescode.BadRequest, "Thứ trong khung giờ vào tập không hợp lệ")
			}
		}
	}
	in.Price = roundMoneyVND(in.Price)
	in.TransferFee = roundMoneyVND(in.TransferFee)
	return in, nil
}

func applyPlanInput(p *entity.MembershipPlan, in v1.PlanInput) {
	p.Code = in.Code
	p.Name = in.Name
	p.Description = in.Description
	p.Type = in.Type
	p.DurationDays = in.DurationDays
	p.SessionCount = in.SessionCount
	p.Price = in.Price
	p.AccessScope = in.AccessScope
	p.MaxFreezeDays = in.MaxFreezeDays
	p.MaxFreezeTimes = in.MaxFreezeTimes
	p.IncludesClasses = in.IncludesClasses
	p.TransferFee = in.TransferFee
	p.GuestPassesPerMonth = in.GuestPassesPerMonth
	p.AccessHours = planAccessHoursJSON(in.AccessHours)
	p.Status = in.Status
	p.SortOrder = in.SortOrder
}

func toPlanRes(p entity.MembershipPlan) *v1.Plan {
	return &v1.Plan{
		ID:                  p.ID,
		Code:                p.Code,
		Name:                p.Name,
		Description:         p.Description,
		Type:                p.Type,
		DurationDays:        p.DurationDays,
		SessionCount:        p.SessionCount,
		Price:               p.Price,
		AccessScope:         p.AccessScope,
		MaxFreezeDays:       p.MaxFreezeDays,
		MaxFreezeTimes:      p.MaxFreezeTimes,
		IncludesClasses:     p.IncludesClasses,
		TransferFee:         p.TransferFee,
		GuestPassesPerMonth: p.GuestPassesPerMonth,
		AccessHours:         planAccessHours(p.AccessHours),
		Status:              p.Status,
		SortOrder:           p.SortOrder,
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
	}
}

func planAccessHoursJSON(slots []v1.AccessHoursSlot) datatypes.JSON {
	if len(slots) == 0 {
		return datatypes.JSON("[]")
	}
	b, _ := json.Marshal(slots)
	return datatypes.JSON(b)
}

// planAccessHours đọc access_hours; dữ liệu cũ dạng ["HH:MM-HH:MM"] được quy về khung mọi ngày.
func planAccessHours(raw datatypes.JSON) []v1.AccessHoursSlot {
	out := []v1.AccessHoursSlot{}
	if len(raw) == 0 {
		return out
	}
	if err := json.Unmarshal(raw, &out); err == nil {
		return out
	}
	var windows []string
	if err := json.Unmarshal(raw, &windows); err != nil {
		return []v1.AccessHoursSlot{}
	}
	for _, w := range windows {
		if from, to, ok := strings.Cut(w, "-"); ok {
			out = append(out, v1.AccessHoursSlot{From: strings.TrimSpace(from), To: strings.TrimSpace(to)})
		}
	}
	return out
}

// roundMoneyVND làm tròn số tiền về đồng (VND không có phần lẻ).
func roundMoneyVND(v float64) float64 {
	return math.Round(v)
}

// auditCatalog ghi audit nếu có AuditService (cho phép nil trong test).
func auditCatalog(ctx context.Context, audit AuditService, action, entityName string, id uint, before, after any) {
	if audit != nil {
		audit.Log(ctx, action, entityName, id, before, after)
	}
}
