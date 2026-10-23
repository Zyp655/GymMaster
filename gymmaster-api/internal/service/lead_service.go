package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/lead/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/mailer"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	JobBirthdayGreetings = "birthday_greetings"
	JobChurnRiskScan     = "churn_risk_scan"

	auditEntityLead = "lead"

	publicLeadRateLimitMax    = 5
	publicLeadRateLimitWindow = 15 * time.Minute
)

var leadOrderable = []string{"id", "created_at", "full_name", "status"}

type (
	LeadService interface {
		PublicCreateLead(ctx context.Context, req *v1.PublicCreateLeadReq, clientIP string) (*v1.Lead, error)
		PublicClassSchedules(ctx context.Context, req *v1.PublicClassSchedulesReq) (v1.PublicClassSchedulesRes, error)

		ListLeads(ctx context.Context, req *v1.ListLeadsReq) (*v1.ListLeadsRes, error)
		GetLead(ctx context.Context, id uint) (*v1.Lead, error)
		CreateLead(ctx context.Context, req *v1.CreateLeadReq) (*v1.Lead, error)
		UpdateLead(ctx context.Context, req *v1.UpdateLeadReq) (*v1.Lead, error)
		// UpdateLeadStatus chỉ đổi trạng thái (Kanban) – kiểm tra luật chuyển trạng thái.
		UpdateLeadStatus(ctx context.Context, id uint, status string) (*v1.Lead, error)
		DeleteLead(ctx context.Context, id uint) error

		ListActivities(ctx context.Context, leadID uint) (v1.ListLeadActivitiesRes, error)
		AddActivity(ctx context.Context, req *v1.AddLeadActivityReq) (*v1.LeadActivity, error)
		IssueTrialPass(ctx context.Context, req *v1.IssueTrialPassReq) (*v1.TrialPass, error)
		ConvertLead(ctx context.Context, req *v1.ConvertLeadReq) (*v1.Lead, error)

		AtRiskMembers(ctx context.Context, req *v1.AtRiskMembersReq) (v1.AtRiskMembersRes, error)
		LeadsReport(ctx context.Context, req *v1.LeadsReportReq) (*v1.LeadsReportRes, error)

		SendBirthdayGreetings(ctx context.Context, now time.Time) (int, error)
		ScanChurnRisk(ctx context.Context, now time.Time) (int, error)
	}

	leadService struct {
		repo     repository.LeadRepository
		tx       postgres.TxManager
		audit    AuditService
		notifier Notifier
		mailer   mailer.Mailer
		now      func() time.Time

		rateMu sync.Mutex
		ipHits map[string][]time.Time
	}
)

func NewLeadService(
	repo repository.LeadRepository,
	tx postgres.TxManager,
	audit AuditService,
	notifier Notifier,
	m mailer.Mailer,
) LeadService {
	return &leadService{
		repo:     repo,
		tx:       tx,
		audit:    audit,
		notifier: notifier,
		mailer:   m,
		now:      time.Now,
		ipHits:   make(map[string][]time.Time),
	}
}

func (s *leadService) withinTx(ctx context.Context, fn func(context.Context) error) error {
	if s.tx == nil {
		return fn(ctx)
	}
	return s.tx.WithTx(ctx, fn)
}

func leadParseDateOpt(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(consts.DateLayout, raw)
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Ngày phải có định dạng YYYY-MM-DD")
	}
	return &t, nil
}

// leadNormalizePhone chuẩn hóa + kiểm tra SĐT theo đúng quy tắc của module hội viên (NormalizeVNPhone).
// leadEmail chuẩn hóa + kiểm tra email (rỗng = không có).
func leadEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if !validEmailStrict(email) {
		return "", rescode.New(rescode.BadRequest, "Email không hợp lệ")
	}
	return email, nil
}

func leadNormalizePhone(raw string) (string, error) {
	phone, ok := NormalizeVNPhone(raw)
	if !ok {
		return "", rescode.New(rescode.BadRequest, "Số điện thoại không hợp lệ")
	}
	return phone, nil
}

// leadCheckStatusChange kiểm tra luật chuyển trạng thái khách tiềm năng:
//   - CONVERTED chỉ đạt được qua chức năng chuyển đổi thành hội viên (có member_id);
//   - CONVERTED là trạng thái cuối, không chuyển sang trạng thái khác.
func leadCheckStatusChange(from, to string) error {
	if to == "" || to == from {
		return nil
	}
	if from == entity.LeadStatusConverted {
		return rescode.New(rescode.BadRequest, "Khách đã chuyển đổi thành hội viên, không thể đổi sang trạng thái khác")
	}
	if to == entity.LeadStatusConverted {
		return rescode.New(rescode.BadRequest, "Vui lòng dùng chức năng \"Chuyển đổi thành hội viên\" để chuyển sang trạng thái Đã chuyển đổi")
	}
	return nil
}

// leadToday: ngày hôm nay theo giờ Việt Nam (00:00).
func leadToday(now time.Time) time.Time {
	t := now.In(memberVNLoc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, memberVNLoc)
}

func (s *leadService) checkIPRateLimit(ip string, now time.Time) bool {
	key := strings.TrimSpace(ip)
	if key == "" {
		key = "unknown"
	}
	s.rateMu.Lock()
	defer s.rateMu.Unlock()

	cutoff := now.Add(-publicLeadRateLimitWindow)
	prev := s.ipHits[key]
	kept := prev[:0]
	for _, t := range prev {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= publicLeadRateLimitMax {
		s.ipHits[key] = kept
		return false
	}
	s.ipHits[key] = append(kept, now)
	return true
}

func (s *leadService) PublicCreateLead(ctx context.Context, req *v1.PublicCreateLeadReq, clientIP string) (*v1.Lead, error) {
	now := s.now()
	if !s.checkIPRateLimit(clientIP, now) {
		// 429 (không phải 401/40105) để client không hiểu nhầm là hết phiên đăng nhập (LD-08).
		return nil, rescode.New(rescode.TooManyRequests, "Bạn đã gửi quá nhiều yêu cầu. Vui lòng thử lại sau 15 phút")
	}

	if err := s.ensureLeadBranchActive(ctx, req.BranchID); err != nil {
		return nil, err
	}
	phone, err := leadNormalizePhone(req.Phone)
	if err != nil {
		return nil, err
	}
	email, err := leadEmail(req.Email)
	if err != nil {
		return nil, err
	}

	// Gửi lại form cùng SĐT khi đã có lead đang mở ở chi nhánh → ghi chú vào lead cũ, không tạo lead trùng (FA-04).
	existing, err := s.repo.FindOpenLeadByPhone(ctx, req.BranchID, phone)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return s.mergePublicLead(ctx, existing, req, email, now)
	}

	lead := &entity.Lead{
		BranchID:       req.BranchID,
		FullName:       strings.TrimSpace(req.FullName),
		Phone:          phone,
		Email:          email,
		Source:         entity.LeadSourceWebsite,
		InterestPlanID: req.InterestPlanID,
		Status:         entity.LeadStatusNew,
		Note:           strings.TrimSpace(req.Note),
	}
	if err := s.repo.CreateLead(ctx, lead); err != nil {
		return nil, err
	}

	if s.notifier != nil {
		s.notifier.NotifyPermission(
			ctx,
			consts.PermLeadManage,
			&lead.BranchID,
			NotifyLeadNew,
			"Khách đăng ký tập thử mới",
			fmt.Sprintf("%s (%s) vừa đăng ký tập thử từ trang chủ.", lead.FullName, lead.Phone),
			fmt.Sprintf("/leads?id=%d", lead.ID),
		)
	}

	row, err := s.repo.FindLeadByID(ctx, lead.ID)
	if err != nil {
		return &v1.Lead{
			ID:        lead.ID,
			BranchID:  lead.BranchID,
			FullName:  lead.FullName,
			Phone:     lead.Phone,
			Email:     lead.Email,
			Source:    lead.Source,
			Status:    lead.Status,
			Note:      lead.Note,
			CreatedAt: lead.CreatedAt,
			UpdatedAt: lead.UpdatedAt,
		}, nil
	}
	return toLeadDTO(row, nil, nil), nil
}

// ensureLeadBranchActive: chi nhánh phải tồn tại (404) và đang hoạt động (400, LD-02).
func (s *leadService) ensureLeadBranchActive(ctx context.Context, branchID uint) error {
	b, err := s.repo.FindBranch(ctx, branchID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy chi nhánh")
		}
		return err
	}
	if b.Status != consts.StatusActive {
		return rescode.New(rescode.BadRequest, "Chi nhánh đang ngừng hoạt động, vui lòng chọn chi nhánh khác")
	}
	return nil
}

// mergePublicLead ghi nhận lần đăng ký lặp lại vào lead đang mở (hoạt động NOTE), bổ sung email nếu trước đó trống.
func (s *leadService) mergePublicLead(ctx context.Context, l *entity.Lead, req *v1.PublicCreateLeadReq, email string, now time.Time) (*v1.Lead, error) {
	content := "Khách gửi lại form đăng ký tập thử từ website"
	if note := strings.TrimSpace(req.Note); note != "" {
		content += ": " + note
	}
	err := s.withinTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.CreateLeadActivity(txCtx, &entity.LeadActivity{
			LeadID: l.ID, Type: "NOTE", Content: content, CreatedAt: now,
		}); err != nil {
			return err
		}
		changed := false
		if l.Email == "" && email != "" {
			l.Email, changed = email, true
		}
		if l.InterestPlanID == nil && req.InterestPlanID != nil {
			l.InterestPlanID, changed = req.InterestPlanID, true
		}
		if changed {
			return s.repo.UpdateLead(txCtx, l)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Trang công khai: chỉ trả lại dữ liệu người gửi vừa nhập, không lộ thông tin đã lưu của lead cũ.
	return &v1.Lead{
		ID: l.ID, BranchID: l.BranchID, FullName: strings.TrimSpace(req.FullName), Phone: l.Phone, Email: email,
		Source: entity.LeadSourceWebsite, Status: entity.LeadStatusNew, Note: strings.TrimSpace(req.Note),
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// publicAvailableSlots tính số chỗ còn trống của lớp, không bao giờ âm.
func publicAvailableSlots(capacity, booked int) int {
	if capacity-booked < 0 {
		return 0
	}
	return capacity - booked
}

func (s *leadService) PublicClassSchedules(ctx context.Context, req *v1.PublicClassSchedulesReq) (v1.PublicClassSchedulesRes, error) {
	now := s.now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	from := today
	to := today.AddDate(0, 0, 7)

	if strings.TrimSpace(req.From) != "" {
		if t, err := time.ParseInLocation(consts.DateLayout, strings.TrimSpace(req.From), now.Location()); err == nil {
			from = t
		}
	}
	if strings.TrimSpace(req.To) != "" {
		if t, err := time.ParseInLocation(consts.DateLayout, strings.TrimSpace(req.To), now.Location()); err == nil {
			to = t.AddDate(0, 0, 1)
		}
	}

	rows, err := s.repo.ListPublicSchedules(ctx, req.BranchID, from, to)
	if err != nil {
		return nil, err
	}
	out := make(v1.PublicClassSchedulesRes, 0, len(rows))
	for _, r := range rows {
		out = append(out, &v1.PublicClassSchedule{
			ID:             r.ID,
			ClassTypeID:    r.ClassTypeID,
			ClassTypeCode:  r.ClassTypeCode,
			ClassTypeName:  r.ClassTypeName,
			ClassTypeColor: r.ClassTypeColor,
			Level:          r.Level,
			BranchID:       r.BranchID,
			BranchName:     r.BranchName,
			RoomID:         r.RoomID,
			RoomName:       r.RoomName,
			TrainerID:      r.TrainerID,
			TrainerName:    r.TrainerName,
			StartAt:        r.StartAt,
			EndAt:          r.EndAt,
			Capacity:       r.Capacity,
			BookedCount:    r.BookedCount,
			AvailableSlots: publicAvailableSlots(r.Capacity, r.BookedCount),
			Status:         r.Status,
		})
	}
	return out, nil
}

func (s *leadService) ListLeads(ctx context.Context, req *v1.ListLeadsReq) (*v1.ListLeadsRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem khách tiềm năng của chi nhánh này")
	}
	q := req.PageReq.ToQuery(leadOrderable, "id")
	q.OrderBy = "l." + q.OrderBy

	rows, total, err := s.repo.ListLeads(ctx, q, repository.LeadFilter{
		BranchIDs:  branchIDs,
		Status:     strings.TrimSpace(req.Status),
		Source:     strings.TrimSpace(req.Source),
		AssignedTo: req.AssignedTo,
		Search:     strings.TrimSpace(req.Search),
	})
	if err != nil {
		return nil, err
	}
	items := make([]*v1.Lead, 0, len(rows))
	for i := range rows {
		items = append(items, toLeadDTO(&rows[i], nil, nil))
	}
	return dto.NewPageRes(items, total, q), nil
}

func (s *leadService) GetLead(ctx context.Context, id uint) (*v1.Lead, error) {
	row, err := s.repo.FindLeadByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy khách tiềm năng")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền truy cập khách tiềm năng của chi nhánh này")
	}
	acts, err := s.repo.ListLeadActivities(ctx, id)
	if err != nil {
		return nil, err
	}
	passes, err := s.repo.ListTrialPasses(ctx, id)
	if err != nil {
		return nil, err
	}
	return toLeadDTO(row, acts, passes), nil
}

func (s *leadService) CreateLead(ctx context.Context, req *v1.CreateLeadReq) (*v1.Lead, error) {
	if !reqctx.CanAccessBranch(ctx, req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền thêm khách tiềm năng cho chi nhánh này")
	}
	if err := s.ensureLeadBranchActive(ctx, req.BranchID); err != nil {
		return nil, err
	}
	trialDate, err := leadParseDateOpt(req.TrialDate)
	if err != nil {
		return nil, err
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = entity.LeadSourceWalkIn
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = entity.LeadStatusNew
	}
	if err := leadCheckStatusChange(entity.LeadStatusNew, status); err != nil {
		return nil, err
	}
	phone, err := leadNormalizePhone(req.Phone)
	if err != nil {
		return nil, err
	}
	email, err := leadEmail(req.Email)
	if err != nil {
		return nil, err
	}
	actorID := reqctx.UserID(ctx)

	lead := &entity.Lead{
		BaseEntity:     entity.BaseEntity{CreatedBy: actorID, UpdatedBy: actorID},
		BranchID:       req.BranchID,
		FullName:       strings.TrimSpace(req.FullName),
		Phone:          phone,
		Email:          email,
		Source:         source,
		InterestPlanID: req.InterestPlanID,
		Status:         status,
		AssignedTo:     req.AssignedTo,
		TrialDate:      trialDate,
		Note:           strings.TrimSpace(req.Note),
	}

	if err := s.repo.CreateLead(ctx, lead); err != nil {
		return nil, err
	}
	res, err := s.GetLead(ctx, lead.ID)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityLead, lead.ID, nil, res)
	}
	return res, nil
}

func (s *leadService) UpdateLead(ctx context.Context, req *v1.UpdateLeadReq) (*v1.Lead, error) {
	row, err := s.repo.FindLeadByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy khách tiềm năng")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền cập nhật khách tiềm năng của chi nhánh này")
	}
	if req.BranchID > 0 && req.BranchID != row.BranchID {
		if !reqctx.CanAccessBranch(ctx, req.BranchID) {
			return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền chuyển khách sang chi nhánh này")
		}
		row.BranchID = req.BranchID
	}
	trialDate, err := leadParseDateOpt(req.TrialDate)
	if err != nil {
		return nil, err
	}
	phone, err := leadNormalizePhone(req.Phone)
	if err != nil {
		return nil, err
	}
	email, err := leadEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if err := leadCheckStatusChange(row.Status, strings.TrimSpace(req.Status)); err != nil {
		return nil, err
	}

	before := toLeadDTO(row, nil, nil)
	row.FullName = strings.TrimSpace(req.FullName)
	row.Phone = phone
	row.Email = email
	if strings.TrimSpace(req.Source) != "" {
		row.Source = strings.TrimSpace(req.Source)
	}
	row.InterestPlanID = req.InterestPlanID
	if strings.TrimSpace(req.Status) != "" {
		row.Status = strings.TrimSpace(req.Status)
	}
	row.AssignedTo = req.AssignedTo
	if req.TrialDate != "" {
		row.TrialDate = trialDate
	}
	row.Note = strings.TrimSpace(req.Note)
	row.UpdatedBy = reqctx.UserID(ctx)

	if err := s.repo.UpdateLead(ctx, &row.Lead); err != nil {
		return nil, err
	}
	res, err := s.GetLead(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityLead, row.ID, before, res)
	}
	return res, nil
}

func (s *leadService) UpdateLeadStatus(ctx context.Context, id uint, status string) (*v1.Lead, error) {
	status = strings.TrimSpace(status)
	row, err := s.repo.FindLeadByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy khách tiềm năng")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền cập nhật khách tiềm năng của chi nhánh này")
	}
	if err := leadCheckStatusChange(row.Status, status); err != nil {
		return nil, err
	}
	if row.Status == status {
		return s.GetLead(ctx, row.ID)
	}
	before := toLeadDTO(row, nil, nil)
	row.Status = status
	row.UpdatedBy = reqctx.UserID(ctx)
	if err := s.repo.UpdateLead(ctx, &row.Lead); err != nil {
		return nil, err
	}
	res, err := s.GetLead(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityLead, row.ID, before, res)
	}
	return res, nil
}

func (s *leadService) DeleteLead(ctx context.Context, id uint) error {
	row, err := s.repo.FindLeadByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy khách tiềm năng")
		}
		return err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return rescode.New(rescode.PermissionDenied, "Bạn không có quyền xóa khách tiềm năng của chi nhánh này")
	}
	if err := s.repo.DeleteLead(ctx, id, reqctx.UserID(ctx)); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityLead, id, toLeadDTO(row, nil, nil), nil)
	}
	return nil
}

func (s *leadService) ListActivities(ctx context.Context, leadID uint) (v1.ListLeadActivitiesRes, error) {
	row, err := s.repo.FindLeadByID(ctx, leadID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy khách tiềm năng")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền truy cập khách tiềm năng của chi nhánh này")
	}
	rows, err := s.repo.ListLeadActivities(ctx, leadID)
	if err != nil {
		return nil, err
	}
	out := make(v1.ListLeadActivitiesRes, 0, len(rows))
	for i := range rows {
		out = append(out, toLeadActivityDTO(&rows[i]))
	}
	return out, nil
}

func (s *leadService) AddActivity(ctx context.Context, req *v1.AddLeadActivityReq) (*v1.LeadActivity, error) {
	row, err := s.repo.FindLeadByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy khách tiềm năng")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền cập nhật khách tiềm năng của chi nhánh này")
	}

	actorID := reqctx.UserID(ctx)

	act := &entity.LeadActivity{
		LeadID:    row.ID,
		Type:      strings.TrimSpace(req.Type),
		Content:   strings.TrimSpace(req.Content),
		CreatedBy: actorID,
		CreatedAt: s.now(),
	}

	err = s.withinTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.CreateLeadActivity(txCtx, act); err != nil {
			return err
		}
		if row.Status == entity.LeadStatusNew {
			row.Status = entity.LeadStatusContacted
			row.UpdatedBy = actorID
			if err := s.repo.UpdateLead(txCtx, &row.Lead); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &v1.LeadActivity{
		ID:        act.ID,
		LeadID:    act.LeadID,
		Type:      act.Type,
		Content:   act.Content,
		CreatedBy: act.CreatedBy,
		CreatedAt: act.CreatedAt,
	}, nil
}

func (s *leadService) IssueTrialPass(ctx context.Context, req *v1.IssueTrialPassReq) (*v1.TrialPass, error) {
	row, err := s.repo.FindLeadByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy khách tiềm năng")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền cấp vé tập thử cho chi nhánh này")
	}

	if row.Status == entity.LeadStatusConverted {
		return nil, rescode.New(rescode.BadRequest, "Khách đã chuyển đổi thành hội viên, không cần cấp vé tập thử")
	}
	if err := s.ensureLeadBranchActive(ctx, row.BranchID); err != nil {
		return nil, err
	}
	now := s.now()
	today := leadToday(now)
	validDate := today
	if strings.TrimSpace(req.ValidDate) != "" {
		t, err := time.ParseInLocation(consts.DateLayout, strings.TrimSpace(req.ValidDate), memberVNLoc)
		if err != nil {
			return nil, rescode.New(rescode.BadRequest, "Ngày tập thử phải có dạng YYYY-MM-DD")
		}
		if t.Before(today) {
			return nil, rescode.New(rescode.BadRequest, "Ngày tập thử không được ở quá khứ")
		}
		validDate = t
	}
	if err := checkPassDayOpen(ctx, row.BranchID, validDate, "cấp vé tập thử"); err != nil {
		return nil, err
	}
	// Mỗi khách chỉ có tối đa 1 vé tập thử còn hiệu lực (chưa dùng và chưa quá ngày).
	passes, err := s.repo.ListTrialPasses(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	for _, p := range passes {
		vd := time.Date(p.ValidDate.Year(), p.ValidDate.Month(), p.ValidDate.Day(), 0, 0, 0, 0, memberVNLoc)
		if p.UsedAt == nil && !vd.Before(today) {
			return nil, rescode.New(rescode.Conflict, "Khách đã có vé tập thử còn hiệu lực (ngày "+vd.Format("02/01/2006")+"), không thể cấp thêm")
		}
	}

	rawBytes := make([]byte, 8)
	_, _ = rand.Read(rawBytes)
	qrToken := "TP-" + strings.ToUpper(hex.EncodeToString(rawBytes))

	actorID := reqctx.UserID(ctx)
	tp := &entity.TrialPass{
		LeadID:    row.ID,
		BranchID:  row.BranchID,
		ValidDate: validDate,
		QRToken:   qrToken,
		CreatedBy: actorID,
		CreatedAt: now,
	}

	err = s.withinTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.CreateTrialPass(txCtx, tp); err != nil {
			return err
		}
		if row.Status == entity.LeadStatusNew || row.Status == entity.LeadStatusContacted {
			row.Status = entity.LeadStatusTrial
		}
		row.TrialDate = &validDate
		row.UpdatedBy = actorID
		if err := s.repo.UpdateLead(txCtx, &row.Lead); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &v1.TrialPass{
		ID:         tp.ID,
		LeadID:     tp.LeadID,
		BranchID:   tp.BranchID,
		BranchName: row.BranchName,
		ValidDate:  tp.ValidDate.Format(consts.DateLayout),
		QRToken:    tp.QRToken,
		UsedAt:     tp.UsedAt,
		CreatedBy:  tp.CreatedBy,
		CreatedAt:  tp.CreatedAt,
	}, nil
}

func (s *leadService) ConvertLead(ctx context.Context, req *v1.ConvertLeadReq) (*v1.Lead, error) {
	row, err := s.repo.FindLeadByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy khách tiềm năng")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền chuyển đổi khách tiềm năng của chi nhánh này")
	}
	if row.Status == entity.LeadStatusConverted && row.MemberID != nil {
		return s.GetLead(ctx, row.ID)
	}

	dob, err := leadParseDateOpt(req.Dob)
	if err != nil {
		return nil, err
	}
	gender := strings.TrimSpace(req.Gender)
	if gender == "" {
		gender = "OTHER"
	}
	// Chuẩn hóa SĐT (lead cũ có thể lưu dạng chưa chuẩn) để khớp đúng hội viên đã có, tránh tạo trùng.
	phone, err := leadNormalizePhone(row.Phone)
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Số điện thoại của khách tiềm năng không hợp lệ, vui lòng cập nhật trước khi chuyển đổi")
	}
	row.Phone = phone
	actorID := reqctx.UserID(ctx)
	before := toLeadDTO(row, nil, nil)

	err = s.withinTx(ctx, func(txCtx context.Context) error {
		mem, findErr := s.repo.FindMemberByPhone(txCtx, phone)
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		if mem == nil {
			mem = &entity.Member{
				BaseEntity:   entity.BaseEntity{CreatedBy: actorID, UpdatedBy: actorID},
				FullName:     row.FullName,
				Gender:       gender,
				Dob:          dob,
				Phone:        row.Phone,
				Email:        row.Email,
				Address:      strings.TrimSpace(req.Address),
				HomeBranchID: row.BranchID,
				Source:       row.Source,
				Status:       entity.MemberStatusActive,
				Note:         row.Note,
			}
			if err := s.repo.CreateMember(txCtx, mem); err != nil {
				return err
			}
		}

		row.Status = entity.LeadStatusConverted
		row.MemberID = &mem.ID
		row.UpdatedBy = actorID
		if err := s.repo.UpdateLead(txCtx, &row.Lead); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	res, err := s.GetLead(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityLead, row.ID, before, res)
	}
	return res, nil
}

func (s *leadService) AtRiskMembers(ctx context.Context, req *v1.AtRiskMembersReq) (v1.AtRiskMembersRes, error) {
	days := req.DaysInactive
	if req.Days > 0 {
		days = req.Days
	}
	if days <= 0 {
		days = 14
	}
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem hội viên của chi nhánh này")
	}

	rows, err := s.repo.ListAtRiskMembers(ctx, branchIDs, s.now(), days)
	if err != nil {
		return nil, err
	}
	out := make(v1.AtRiskMembersRes, 0, len(rows))
	for _, r := range rows {
		out = append(out, &v1.AtRiskMember{
			ID:            r.MemberID,
			MemberID:      r.MemberID,
			MemberCode:    r.MemberCode,
			FullName:      r.FullName,
			Phone:         r.Phone,
			Email:         r.Email,
			BranchID:      r.BranchID,
			BranchName:    r.BranchName,
			PlanName:      r.PlanName,
			EndDate:       r.EndDate.Format(consts.DateLayout),
			LastCheckinAt: r.LastCheckinAt,
			DaysInactive:  r.DaysInactive,
			AbsentDays:    r.DaysInactive,
		})
	}
	return out, nil
}

func (s *leadService) LeadsReport(ctx context.Context, req *v1.LeadsReportReq) (*v1.LeadsReportRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem báo cáo của chi nhánh này")
	}
	from, err := leadParseDateOpt(req.From)
	if err != nil {
		return nil, err
	}
	to, err := leadParseDateOpt(req.To)
	if err != nil {
		return nil, err
	}
	if to != nil {
		endExclusive := to.AddDate(0, 0, 1)
		to = &endExclusive
	}

	data, err := s.repo.LeadsReport(ctx, branchIDs, from, to)
	if err != nil {
		return nil, err
	}
	var rate float64
	if data.Total > 0 {
		rate = math.Round(float64(data.Converted)*10000/float64(data.Total)) / 100
	}
	byStatus := make([]*v1.LeadCountItem, 0, len(data.ByStatus))
	for _, item := range data.ByStatus {
		byStatus = append(byStatus, &v1.LeadCountItem{
			Key:    item.Key,
			Status: item.Key,
			Count:  item.Count,
		})
	}
	bySource := make([]*v1.LeadCountItem, 0, len(data.BySource))
	for _, item := range data.BySource {
		bySource = append(bySource, &v1.LeadCountItem{
			Key:    item.Key,
			Source: item.Key,
			Count:  item.Count,
		})
	}
	return &v1.LeadsReportRes{
		Total:          data.Total,
		Converted:      data.Converted,
		ConversionRate: rate,
		ByStatus:       byStatus,
		BySource:       bySource,
	}, nil
}

func (s *leadService) SendBirthdayGreetings(ctx context.Context, now time.Time) (int, error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	members, err := s.repo.ListBirthdayMembers(ctx, today)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, m := range members {
		code := fmt.Sprintf("BDAY-%s-%d", m.MemberCode, today.Year())
		endAt := today.AddDate(0, 0, 7).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		usageLimit := 1
		promo := &entity.Promotion{
			Code:       code,
			Name:       fmt.Sprintf("Ưu đãi sinh nhật hội viên %s (%d)", m.FullName, today.Year()),
			Type:       entity.PromotionPercent,
			Value:      10,
			AppliesTo:  "ALL",
			StartAt:    today,
			EndAt:      endAt,
			UsageLimit: &usageLimit,
			Status:     consts.StatusActive,
		}
		_ = s.repo.EnsureBirthdayPromotion(ctx, promo)

		if s.notifier != nil && m.UserID != nil && *m.UserID > 0 {
			_ = s.notifier.Notify(
				ctx,
				[]uint{*m.UserID},
				NotifyBirthday,
				"Chúc mừng sinh nhật bạn!",
				fmt.Sprintf("GymMaster gửi tặng bạn mã ưu đãi %s giảm 10%% khi gia hạn gói tập trong 7 ngày tới.", code),
				"/me/subscriptions/buy",
			)
		}
		if s.mailer != nil && strings.TrimSpace(m.Email) != "" {
			if body, renderErr := mailer.Render(mailer.TplBirthdayGreeting, mailer.BirthdayData{
				MemberName:   m.FullName,
				VoucherCode:  code,
				DiscountText: "Giảm 10%",
				EndDate:      endAt.Format("02/01/2006"),
			}); renderErr == nil {
				s.mailer.SendAsync(mailer.Message{
					To:      []string{m.Email},
					Subject: "Chúc mừng sinh nhật từ GymMaster!",
					HTML:    body,
				})
			}
		}
		count++
	}
	return count, nil
}

func (s *leadService) ScanChurnRisk(ctx context.Context, now time.Time) (int, error) {
	rows, err := s.repo.ListAtRiskMembers(ctx, nil, now, 14)
	if err != nil {
		return 0, err
	}
	byBranch := map[uint]int{}
	for _, r := range rows {
		byBranch[r.BranchID]++
	}
	if s.notifier != nil {
		for branchID, cnt := range byBranch {
			bid := branchID
			s.notifier.NotifyPermission(
				ctx,
				consts.PermMemberView,
				&bid,
				NotifySubscriptionExpiring,
				"Danh sách hội viên cần chăm sóc (vắng mặt > 14 ngày)",
				fmt.Sprintf("Chi nhánh hiện có %d hội viên có gói đang hoạt động nhưng hơn 14 ngày chưa check-in.", cnt),
				"/leads?tab=at-risk",
			)
		}
	}
	return len(rows), nil
}

func toLeadActivityDTO(r *repository.LeadActivityRow) *v1.LeadActivity {
	return &v1.LeadActivity{
		ID:          r.ID,
		LeadID:      r.LeadID,
		Type:        r.Type,
		Content:     r.Content,
		CreatedBy:   r.CreatedBy,
		CreatorName: r.CreatorName,
		CreatedAt:   r.CreatedAt,
	}
}

func toTrialPassDTO(r *repository.TrialPassRow) *v1.TrialPass {
	return &v1.TrialPass{
		ID:         r.ID,
		LeadID:     r.LeadID,
		BranchID:   r.BranchID,
		BranchName: r.BranchName,
		ValidDate:  r.ValidDate.Format(consts.DateLayout),
		QRToken:    r.QRToken,
		UsedAt:     r.UsedAt,
		CreatedBy:  r.CreatedBy,
		CreatedAt:  r.CreatedAt,
	}
}

func toLeadDTO(r *repository.LeadRow, acts []repository.LeadActivityRow, passes []repository.TrialPassRow) *v1.Lead {
	var trialDate *string
	if r.TrialDate != nil {
		s := r.TrialDate.Format(consts.DateLayout)
		trialDate = &s
	}
	out := &v1.Lead{
		ID:               r.ID,
		BranchID:         r.BranchID,
		BranchName:       r.BranchName,
		FullName:         r.FullName,
		Phone:            r.Phone,
		Email:            r.Email,
		Source:           r.Source,
		InterestPlanID:   r.InterestPlanID,
		InterestPlanName: r.InterestPlanName,
		Status:           r.Status,
		AssignedTo:       r.AssignedTo,
		AssignedName:     r.AssignedName,
		AssignedToName:   r.AssignedName,
		TrialDate:        trialDate,
		MemberID:         r.MemberID,
		MemberCode:       r.MemberCode,
		Note:             r.Note,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
	if len(acts) > 0 {
		out.Activities = make([]*v1.LeadActivity, 0, len(acts))
		for i := range acts {
			out.Activities = append(out.Activities, toLeadActivityDTO(&acts[i]))
		}
	}
	if len(passes) > 0 {
		out.TrialPasses = make([]*v1.TrialPass, 0, len(passes))
		for i := range passes {
			tp := toTrialPassDTO(&passes[i])
			out.TrialPasses = append(out.TrialPasses, tp)
		}
		out.TrialPass = out.TrialPasses[0]
	}
	return out
}
