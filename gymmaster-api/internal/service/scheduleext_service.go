package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/scheduleext/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	auditEntityTrainerAvailability = "trainer_availability"
	auditEntityLeaveRequest        = "leave_request"
	auditEntityBranchHoliday       = "branch_holiday"
	auditEntityClassSchedule       = "class_schedule"
	auditEntityPTSession           = "pt_session"
	auditActionSubstitute          = "SUBSTITUTE_TRAINER"
	auditActionReassign            = "REASSIGN_TRAINER"
)

// scheduleHHMMPattern: giờ dạng H:mm hoặc HH:mm (00:00–23:59).
var scheduleHHMMPattern = regexp.MustCompile(`^([01]?\d|2[0-3]):([0-5]\d)$`)

// scheduleNormalizeHHMM chuẩn hóa "7:00"/"07:00" → "07:00" để so sánh chuỗi đúng thứ tự thời gian; ok=false nếu sai định dạng.
func scheduleNormalizeHHMM(raw string) (string, bool) {
	m := scheduleHHMMPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if m == nil {
		return "", false
	}
	h := m[1]
	if len(h) == 1 {
		h = "0" + h
	}
	return h + ":" + m[2], true
}

var (
	leaveRequestOrderable  = []string{"id", "from_at", "to_at", "status", "created_at"}
	branchHolidayOrderable = []string{"id", "date", "name", "created_at"}
)

type (
	ScheduleExtService interface {
		GetTrainerAvailability(ctx context.Context, req *v1.GetTrainerAvailabilityReq) (v1.GetTrainerAvailabilityRes, error)
		SaveTrainerAvailability(ctx context.Context, req *v1.SaveTrainerAvailabilityReq) (v1.SaveTrainerAvailabilityRes, error)

		ListLeaveRequests(ctx context.Context, req *v1.ListLeaveRequestsReq) (*v1.ListLeaveRequestsRes, error)
		CreateLeaveRequest(ctx context.Context, req *v1.CreateLeaveRequestReq) (*v1.LeaveRequestItem, error)
		ApproveLeaveRequest(ctx context.Context, req *v1.ApproveLeaveRequestReq) (*v1.LeaveRequestItem, error)
		RejectLeaveRequest(ctx context.Context, req *v1.RejectLeaveRequestReq) (*v1.LeaveRequestItem, error)
		GetLeaveImpacts(ctx context.Context, req *v1.GetLeaveImpactsReq) (*v1.GetLeaveImpactsRes, error)

		SubstituteClassTrainer(ctx context.Context, req *v1.SubstituteClassTrainerReq) (*v1.ImpactedClassSchedule, error)
		ReassignPTSession(ctx context.Context, req *v1.ReassignPTSessionReq) (*v1.ImpactedPTSession, error)

		ListBranchHolidays(ctx context.Context, req *v1.ListBranchHolidaysReq) (*v1.ListBranchHolidaysRes, error)
		CreateBranchHoliday(ctx context.Context, req *v1.CreateBranchHolidayReq) (*v1.BranchHolidayItem, error)
		UpdateBranchHoliday(ctx context.Context, req *v1.UpdateBranchHolidayReq) (*v1.BranchHolidayItem, error)
		DeleteBranchHoliday(ctx context.Context, id uint) error
		PublicHolidays(ctx context.Context, req *v1.PublicHolidaysReq) (v1.PublicHolidaysRes, error)
	}

	scheduleExtService struct {
		repo  repository.ScheduleExtRepository
		tx    postgres.TxManager
		audit AuditService
		now   func() time.Time
	}
)

func NewScheduleExtService(repo repository.ScheduleExtRepository, tx postgres.TxManager, audit AuditService) ScheduleExtService {
	return &scheduleExtService{repo: repo, tx: tx, audit: audit, now: time.Now}
}

func (s *scheduleExtService) runTx(ctx context.Context, fn func(context.Context) error) error {
	if s.tx != nil {
		return s.tx.WithTx(ctx, fn)
	}
	return fn(ctx)
}

func resolveScheduleBranchScope(ctx context.Context, requested uint) (uint, error) {
	if requested > 0 {
		if !reqctx.CanAccessBranch(ctx, requested) {
			return 0, rescode.New(rescode.PermissionDenied, "Bạn không có quyền thao tác trên chi nhánh này")
		}
		return requested, nil
	}
	if allowed := reqctx.AllowedBranches(ctx); len(allowed) > 0 {
		return allowed[0], nil
	}
	return 0, nil
}

// ---- Trainer Availability -------------------------------------------------------

// GetTrainerAvailability: HLV xem lịch của chính mình; người khác chỉ xem lịch rảnh tại chi nhánh trong phạm vi được
// phân công (HLV ngoài phạm vi → 403, lịch ở chi nhánh khác của HLV bị lọc bỏ).
func (s *scheduleExtService) GetTrainerAvailability(ctx context.Context, req *v1.GetTrainerAvailabilityReq) (v1.GetTrainerAvailabilityRes, error) {
	self := false
	if uid := reqctx.UserID(ctx); uid != nil && *uid == req.ID {
		self = true
	}
	allowed := reqctx.AllowedBranches(ctx)
	if !self && req.BranchID > 0 && !reqctx.CanAccessBranch(ctx, req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem lịch rảnh HLV của chi nhánh này")
	}
	if !self && req.BranchID == 0 && len(allowed) > 0 {
		inScope := false
		for _, b := range allowed {
			ok, err := s.repo.TrainerEligible(ctx, req.ID, b)
			if err != nil {
				return nil, err
			}
			if ok {
				inScope = true
				break
			}
		}
		if !inScope {
			return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem lịch rảnh của HLV chi nhánh khác")
		}
	}
	rows, err := s.repo.ListTrainerAvailability(ctx, req.ID, req.BranchID)
	if err != nil {
		return nil, err
	}
	if !self && len(allowed) > 0 {
		kept := rows[:0]
		for _, r := range rows {
			if reqctx.CanAccessBranch(ctx, r.BranchID) {
				kept = append(kept, r)
			}
		}
		rows = kept
	}
	return dto.Map(rows, toTrainerAvailabilityItem), nil
}

// scheduleIsSupervisor: OWNER/MANAGER được thao tác lịch của nhân viên khác (trong phạm vi chi nhánh).
func scheduleIsSupervisor(ctx context.Context) bool {
	return reqctx.HasRole(ctx, consts.RoleOwner) || reqctx.HasRole(ctx, consts.RoleManager)
}

// scheduleCheckSelf: nhân viên không phải quản lý chỉ được thao tác lịch của chính mình.
func scheduleCheckSelf(ctx context.Context, targetUserID uint, msg string) error {
	if scheduleIsSupervisor(ctx) {
		return nil
	}
	if uid := reqctx.UserID(ctx); uid == nil || *uid != targetUserID {
		return rescode.New(rescode.PermissionDenied, msg)
	}
	return nil
}

// SaveTrainerAvailability thay toàn bộ lịch rảnh của HLV tại MỘT chi nhánh (branch_id bắt buộc). Lịch rảnh ở chi nhánh
// khác giữ nguyên. Danh sách rỗng = xóa cấu hình tại chi nhánh → HLV nhận PT trong mọi giờ mở cửa (không giới hạn).
func (s *scheduleExtService) SaveTrainerAvailability(ctx context.Context, req *v1.SaveTrainerAvailabilityReq) (v1.SaveTrainerAvailabilityRes, error) {
	if err := scheduleCheckSelf(ctx, req.ID, "HLV chỉ được cập nhật lịch rảnh của chính mình"); err != nil {
		return nil, err
	}
	if req.BranchID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn chi nhánh cần cấu hình lịch rảnh")
	}
	if !reqctx.CanAccessBranch(ctx, req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền cấu hình lịch rảnh cho chi nhánh này")
	}
	ok, err := s.repo.TrainerEligible(ctx, req.ID, req.BranchID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, rescode.New(rescode.BadRequest, "HLV không tồn tại, đã bị khóa hoặc không thuộc chi nhánh này")
	}
	inSlots := req.Slots
	if len(inSlots) == 0 && len(req.Items) > 0 {
		inSlots = req.Items
	}
	slots := make([]entity.TrainerAvailability, 0, len(inSlots))
	for _, sl := range inSlots {
		if sl.DOW < 0 || sl.DOW > 6 {
			return nil, rescode.New(rescode.BadRequest, "Thứ trong tuần phải từ 0 đến 6")
		}
		if sl.BranchID != 0 && sl.BranchID != req.BranchID {
			return nil, rescode.New(rescode.BadRequest, "Mỗi lần chỉ lưu lịch rảnh của một chi nhánh")
		}
		st, ok1 := scheduleNormalizeHHMM(sl.StartTime)
		et, ok2 := scheduleNormalizeHHMM(sl.EndTime)
		if !ok1 || !ok2 {
			return nil, rescode.New(rescode.BadRequest, "Giờ rảnh phải có dạng HH:mm (00:00 - 23:59)")
		}
		if st >= et {
			return nil, rescode.New(rescode.BadRequest, "Giờ bắt đầu phải trước giờ kết thúc")
		}
		effFrom, err := workoutParseDateOpt(sl.EffectiveFrom)
		if err != nil {
			return nil, err
		}
		effTo, err := workoutParseDateOpt(sl.EffectiveTo)
		if err != nil {
			return nil, err
		}
		if effFrom != nil && effTo != nil && effTo.Before(*effFrom) {
			return nil, rescode.New(rescode.BadRequest, "Ngày hết hiệu lực phải sau ngày bắt đầu hiệu lực")
		}
		for _, o := range slots {
			if o.DOW == sl.DOW && o.StartTime < et && st < o.EndTime && scheduleEffOverlap(o.EffectiveFrom, o.EffectiveTo, effFrom, effTo) {
				return nil, rescode.New(rescode.BadRequest, fmt.Sprintf("Khung giờ rảnh %s %s-%s bị chồng với khung %s-%s",
					scheduleDOWLabel(sl.DOW), st, et, o.StartTime, o.EndTime))
			}
		}
		slots = append(slots, entity.TrainerAvailability{
			TrainerID:     req.ID,
			BranchID:      req.BranchID,
			DOW:           sl.DOW,
			StartTime:     st,
			EndTime:       et,
			EffectiveFrom: effFrom,
			EffectiveTo:   effTo,
		})
	}
	before, _ := s.repo.ListTrainerAvailability(ctx, req.ID, req.BranchID)
	err = s.runTx(ctx, func(txCtx context.Context) error {
		return s.repo.ReplaceTrainerAvailability(txCtx, req.ID, req.BranchID, slots)
	})
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityTrainerAvailability, req.ID,
			map[string]any{"branch_id": req.BranchID, "slots": dto.Map(before, toTrainerAvailabilityItem)},
			map[string]any{"branch_id": req.BranchID, "slots": slots})
	}
	return s.GetTrainerAvailability(ctx, &v1.GetTrainerAvailabilityReq{ID: req.ID, BranchID: req.BranchID})
}

// scheduleEffOverlap: hai khoảng hiệu lực (nil = không giới hạn) có giao nhau.
func scheduleEffOverlap(aFrom, aTo, bFrom, bTo *time.Time) bool {
	if aTo != nil && bFrom != nil && aTo.Before(*bFrom) {
		return false
	}
	if bTo != nil && aFrom != nil && bTo.Before(*aFrom) {
		return false
	}
	return true
}

func scheduleDOWLabel(dow int) string {
	if dow == 0 {
		return "Chủ nhật"
	}
	return fmt.Sprintf("Thứ %d", dow+1)
}

// ---- Leave Requests -------------------------------------------------------------

func (s *scheduleExtService) ListLeaveRequests(ctx context.Context, req *v1.ListLeaveRequestsReq) (*v1.ListLeaveRequestsRes, error) {
	branchID, err := resolveScheduleBranchScope(ctx, req.BranchID)
	if err != nil {
		return nil, err
	}
	q := req.PageReq.ToQuery(leaveRequestOrderable, "id")
	q.OrderBy = "lr." + q.OrderBy
	staffID := req.StaffID
	// Nhân viên/HLV không phải quản lý chỉ xem đơn nghỉ phép của chính mình.
	if auth := reqctx.Auth(ctx); !scheduleIsSupervisor(ctx) && (auth == nil || !auth.HasPermission(consts.PermLeaveManage)) {
		uid := reqctx.UserID(ctx)
		if uid == nil {
			return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
		}
		staffID = *uid
		branchID = 0
	}
	rows, total, err := s.repo.ListLeaveRequests(ctx, q, staffID, branchID, strings.TrimSpace(req.Status))
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, toLeaveRequestItem), total, q), nil
}

// scheduleParseLeaveTime đọc thời điểm nghỉ phép (RFC3339, YYYY-MM-DDTHH:mm[:ss], YYYY-MM-DD giờ VN); rỗng = def.
// Không dùng workoutParsePerformedAt vì hàm đó chặn thời điểm tương lai (đơn nghỉ phép luôn ở tương lai).
func scheduleParseLeaveTime(raw string, def time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04", consts.DateLayout} {
		if t, err := time.ParseInLocation(layout, raw, subVNLoc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, rescode.New(rescode.BadRequest, "Thời gian không hợp lệ")
}

func (s *scheduleExtService) CreateLeaveRequest(ctx context.Context, req *v1.CreateLeaveRequestReq) (*v1.LeaveRequestItem, error) {
	if req.BranchID != nil && *req.BranchID > 0 && !reqctx.CanAccessBranch(ctx, *req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền tạo đơn nghỉ phép tại chi nhánh này")
	}
	staffID := req.StaffID
	if staffID == 0 {
		if uid := reqctx.UserID(ctx); uid != nil {
			staffID = *uid
		}
	}
	if staffID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn nhân viên/HLV")
	}
	if err := scheduleCheckSelf(ctx, staffID, "Bạn chỉ được tạo đơn nghỉ phép cho chính mình"); err != nil {
		return nil, err
	}
	fromAt, err := scheduleParseLeaveTime(req.FromAt, s.now())
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Thời gian bắt đầu nghỉ không hợp lệ")
	}
	toAt, err := scheduleParseLeaveTime(req.ToAt, fromAt)
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Thời gian kết thúc nghỉ không hợp lệ")
	}
	// Nếu truyền dạng YYYY-MM-DD cho to_at thì cộng đến cuối ngày
	if len(strings.TrimSpace(req.ToAt)) == 10 {
		toAt = toAt.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}
	if !toAt.After(fromAt) {
		return nil, rescode.New(rescode.BadRequest, "Thời gian kết thúc phải sau thời gian bắt đầu")
	}
	lr := &entity.StaffLeaveRequest{
		StaffID:  staffID,
		BranchID: req.BranchID,
		FromAt:   fromAt,
		ToAt:     toAt,
		Reason:   strings.TrimSpace(req.Reason),
		Status:   entity.LeaveStatusPending,
	}
	if err := s.repo.CreateLeaveRequest(ctx, lr); err != nil {
		return nil, err
	}
	row, err := s.repo.FindLeaveRequestByID(ctx, lr.ID)
	if err != nil {
		row = &repository.LeaveRequestRow{StaffLeaveRequest: *lr}
	}
	res := toLeaveRequestItem(*row)
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityLeaveRequest, lr.ID, nil, res)
	}
	return res, nil
}

func (s *scheduleExtService) ApproveLeaveRequest(ctx context.Context, req *v1.ApproveLeaveRequestReq) (*v1.LeaveRequestItem, error) {
	return s.reviewLeaveRequest(ctx, req.ID, entity.LeaveStatusApproved, req.ReviewNote)
}

func (s *scheduleExtService) RejectLeaveRequest(ctx context.Context, req *v1.RejectLeaveRequestReq) (*v1.LeaveRequestItem, error) {
	return s.reviewLeaveRequest(ctx, req.ID, entity.LeaveStatusRejected, req.ReviewNote)
}

func (s *scheduleExtService) reviewLeaveRequest(ctx context.Context, id uint, status, note string) (*v1.LeaveRequestItem, error) {
	row, err := s.repo.FindLeaveRequestByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy đơn nghỉ phép")
		}
		return nil, err
	}
	if row.BranchID != nil && *row.BranchID > 0 && !reqctx.CanAccessBranch(ctx, *row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền duyệt đơn nghỉ phép của chi nhánh này")
	}
	now := s.now()
	lr := row.StaffLeaveRequest
	lr.Status = status
	lr.ApprovedBy = reqctx.UserID(ctx)
	lr.ApprovedAt = &now
	lr.ReviewNote = strings.TrimSpace(note)
	if err := s.repo.UpdateLeaveRequest(ctx, &lr); err != nil {
		return nil, err
	}
	updated, err := s.repo.FindLeaveRequestByID(ctx, id)
	if err != nil {
		updated = &repository.LeaveRequestRow{StaffLeaveRequest: lr}
	}
	res := toLeaveRequestItem(*updated)
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityLeaveRequest, id, row.Status, res)
	}
	return res, nil
}

func (s *scheduleExtService) GetLeaveImpacts(ctx context.Context, req *v1.GetLeaveImpactsReq) (*v1.GetLeaveImpactsRes, error) {
	lrRow, err := s.repo.FindLeaveRequestByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy đơn nghỉ phép")
		}
		return nil, err
	}
	classes, err := s.repo.ListImpactedClassSchedules(ctx, lrRow.StaffID, lrRow.FromAt, lrRow.ToAt)
	if err != nil {
		return nil, err
	}
	ptSessions, err := s.repo.ListImpactedPTSessions(ctx, lrRow.StaffID, lrRow.FromAt, lrRow.ToAt)
	if err != nil {
		return nil, err
	}
	return &v1.GetLeaveImpactsRes{
		LeaveRequest:   toLeaveRequestItem(*lrRow),
		ClassSchedules: dto.Map(classes, toImpactedClassSchedule),
		PTSessions:     dto.Map(ptSessions, toImpactedPTSession),
	}, nil
}

// ---- Substitution & Reassignment ------------------------------------------------

// scheduleTrainerBusyMsg: thông điệp khi HLV mới bận (repo.TrainerBusy trả "PT" | "CLASS").
func scheduleTrainerBusyMsg(kind string) string {
	if kind == "PT" {
		return "HLV mới đã có buổi PT khác trùng giờ"
	}
	return "HLV mới đã dạy lớp nhóm khác trùng giờ"
}

// checkNewTrainer: HLV mới tồn tại, ACTIVE, là HLV, thuộc chi nhánh; chi nhánh mở cửa trong khung giờ; HLV không nghỉ
// phép (và nằm trong lịch rảnh nếu checkAvailability) và không trùng buổi PT/lớp khác.
func (s *scheduleExtService) checkNewTrainer(ctx context.Context, trainerID, branchID uint, start, end time.Time,
	excludeClassID, excludePTID uint, checkAvailability bool) error {
	ok, err := s.repo.TrainerEligible(ctx, trainerID, branchID)
	if err != nil {
		return err
	}
	if !ok {
		return rescode.New(rescode.BadRequest, "HLV dạy thay không tồn tại, đã bị khóa, không phải HLV hoặc không thuộc chi nhánh")
	}
	g := currentScheduleGuard()
	if err := g.CheckBranchOpen(ctx, branchID, start, end); err != nil {
		return err
	}
	if checkAvailability {
		if err := g.CheckTrainerSlot(ctx, trainerID, branchID, start, end); err != nil {
			return err
		}
	} else if err := g.CheckTrainerLeave(ctx, trainerID, start, end); err != nil {
		return err
	}
	busy, err := s.repo.TrainerBusy(ctx, trainerID, start, end, excludeClassID, excludePTID)
	if err != nil {
		return err
	}
	if busy != "" {
		return rescode.New(rescode.Conflict, scheduleTrainerBusyMsg(busy))
	}
	return nil
}

// SubstituteClassTrainer phân công HLV dạy thay cho buổi lớp SCHEDULED chưa kết thúc. Chọn lại HLV chính = bỏ dạy thay.
func (s *scheduleExtService) SubstituteClassTrainer(ctx context.Context, req *v1.SubstituteClassTrainerReq) (*v1.ImpactedClassSchedule, error) {
	subID := req.SubstituteTrainerID
	if subID == 0 {
		subID = req.TrainerID
	}
	if subID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn huấn luyện viên dạy thay")
	}
	cls, err := s.repo.FindClassScheduleByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy buổi lớp học")
		}
		return nil, err
	}
	if cls.BranchID > 0 && !reqctx.CanAccessBranch(ctx, cls.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền phân công dạy thay tại chi nhánh này")
	}
	if cls.Status != entity.ClassScheduleScheduled {
		return nil, rescode.New(rescode.BadRequest, "Chỉ phân công dạy thay cho buổi lớp đang xếp lịch (chưa hủy/chưa kết thúc)")
	}
	if !cls.EndAt.After(s.now()) {
		return nil, rescode.New(rescode.BadRequest, "Buổi lớp đã kết thúc, không thể phân công dạy thay")
	}
	var newSub *uint
	if subID != cls.TrainerID {
		if cls.SubstituteTrainerID != nil && *cls.SubstituteTrainerID == subID {
			return nil, rescode.New(rescode.BadRequest, "HLV này đã được phân công dạy thay buổi lớp")
		}
		if err := s.checkNewTrainer(ctx, subID, cls.BranchID, cls.StartAt, cls.EndAt, cls.ID, 0, false); err != nil {
			return nil, err
		}
		newSub = &subID
	} else if cls.SubstituteTrainerID == nil {
		return nil, rescode.New(rescode.BadRequest, "HLV dạy thay phải khác HLV phụ trách lớp")
	} else if err := s.checkNewTrainer(ctx, subID, cls.BranchID, cls.StartAt, cls.EndAt, cls.ID, 0, false); err != nil {
		// Trả lớp về HLV chính cũng phải kiểm tra như HLV mới: HLV chính có thể đã nghỉ phép / nhận PT / dạy lớp khác.
		return nil, err
	}
	if err := s.repo.SetClassSubstituteTrainer(ctx, req.ID, newSub, strings.TrimSpace(req.Note), reqctx.UserID(ctx)); err != nil {
		if isExclusionViolation(err) {
			return nil, rescode.New(rescode.Conflict, "HLV vừa được xếp lớp khác trùng giờ, vui lòng thử lại")
		}
		return nil, err
	}
	updated, err := s.repo.FindClassScheduleByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, auditActionSubstitute, auditEntityClassSchedule, req.ID,
			map[string]any{"substitute_trainer_id": cls.SubstituteTrainerID},
			map[string]any{"substitute_trainer_id": newSub, "note": strings.TrimSpace(req.Note)})
	}
	return toImpactedClassSchedule(*updated), nil
}

// ReassignPTSession chuyển buổi PT SCHEDULED chưa kết thúc sang HLV khác (kiểm tra lịch rảnh, nghỉ phép, nghỉ lễ, trùng lịch).
func (s *scheduleExtService) ReassignPTSession(ctx context.Context, req *v1.ReassignPTSessionReq) (*v1.ImpactedPTSession, error) {
	newTrainerID := req.TrainerID
	if newTrainerID == 0 {
		newTrainerID = req.NewTrainerID
	}
	if newTrainerID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn huấn luyện viên dạy thay")
	}
	sess, err := s.repo.FindPTSessionByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy buổi tập PT")
		}
		return nil, err
	}
	if sess.BranchID > 0 && !reqctx.CanAccessBranch(ctx, sess.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền chuyển giao buổi PT tại chi nhánh này")
	}
	if sess.Status != entity.PTSessionScheduled {
		return nil, rescode.New(rescode.BadRequest, "Chỉ chuyển HLV cho buổi PT đang xếp lịch (chưa hoàn thành/hủy)")
	}
	if !sess.EndAt.After(s.now()) {
		return nil, rescode.New(rescode.BadRequest, "Buổi PT đã kết thúc, không thể chuyển HLV")
	}
	if newTrainerID == sess.TrainerID {
		return nil, rescode.New(rescode.BadRequest, "HLV mới phải khác HLV hiện tại của buổi PT")
	}
	if err := s.checkNewTrainer(ctx, newTrainerID, sess.BranchID, sess.StartAt, sess.EndAt, 0, sess.ID, true); err != nil {
		return nil, err
	}
	origTrainerID := sess.TrainerID
	if sess.OriginalTrainerID != nil && *sess.OriginalTrainerID > 0 {
		origTrainerID = *sess.OriginalTrainerID
	}
	if err := s.repo.ReassignPTSessionTrainer(ctx, req.ID, newTrainerID, origTrainerID); err != nil {
		if isExclusionViolation(err) {
			return nil, rescode.New(rescode.Conflict, "HLV mới vừa có buổi PT khác trùng giờ, vui lòng thử lại")
		}
		return nil, err
	}
	updated, err := s.repo.FindPTSessionByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, auditActionReassign, auditEntityPTSession, req.ID,
			map[string]any{"trainer_id": sess.TrainerID},
			map[string]any{"trainer_id": newTrainerID, "original_trainer_id": origTrainerID, "note": strings.TrimSpace(req.Note)})
	}
	return toImpactedPTSession(*updated), nil
}

// ---- Branch Holidays ------------------------------------------------------------

func (s *scheduleExtService) ListBranchHolidays(ctx context.Context, req *v1.ListBranchHolidaysReq) (*v1.ListBranchHolidaysRes, error) {
	q := req.PageReq.ToQuery(branchHolidayOrderable, "date")
	q.OrderBy = "bh." + q.OrderBy
	rows, total, err := s.repo.ListBranchHolidays(ctx, q, req.BranchID, req.Year)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, toBranchHolidayItem), total, q), nil
}

// checkHolidayScope: lịch riêng chi nhánh → phải có quyền chi nhánh đó; lịch toàn hệ thống → chỉ user không giới hạn chi nhánh.
func checkHolidayScope(ctx context.Context, branchID *uint) error {
	if branchID != nil && *branchID > 0 {
		if !reqctx.CanAccessBranch(ctx, *branchID) {
			return rescode.New(rescode.PermissionDenied, "Bạn không có quyền cấu hình lịch nghỉ lễ cho chi nhánh này")
		}
		return nil
	}
	if allowed := reqctx.AllowedBranches(ctx); len(allowed) > 0 {
		return rescode.New(rescode.PermissionDenied, "Bạn chỉ được cấu hình lịch nghỉ lễ cho chi nhánh được phân công")
	}
	return nil
}

// buildBranchHoliday kiểm tra & chuẩn hóa dữ liệu ngày lễ (dùng chung thêm/sửa):
//   - closed=true (mặc định): đóng cửa cả ngày, bỏ giờ đặc biệt.
//   - closed=false: bắt buộc open_time/close_time – mở cửa theo giờ đặc biệt (HH:mm, mở < đóng).
func buildBranchHoliday(branchID *uint, dateStr, nameStr string, closedIn *bool, openIn, closeIn *string) (*entity.BranchHoliday, error) {
	name := strings.TrimSpace(nameStr)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên ngày nghỉ lễ")
	}
	date, err := time.Parse(consts.DateLayout, strings.TrimSpace(dateStr))
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Ngày nghỉ lễ phải có định dạng YYYY-MM-DD")
	}
	if branchID != nil && *branchID == 0 {
		branchID = nil
	}
	h := &entity.BranchHoliday{BranchID: branchID, Date: date, Name: name, Closed: true}
	if closedIn != nil {
		h.Closed = *closedIn
	}
	if h.Closed {
		return h, nil
	}
	openRaw, closeRaw := "", ""
	if openIn != nil {
		openRaw = strings.TrimSpace(*openIn)
	}
	if closeIn != nil {
		closeRaw = strings.TrimSpace(*closeIn)
	}
	if openRaw == "" || closeRaw == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập đủ giờ mở cửa và giờ đóng cửa đặc biệt")
	}
	openT, ok1 := scheduleNormalizeHHMM(openRaw)
	closeT, ok2 := scheduleNormalizeHHMM(closeRaw)
	if !ok1 || !ok2 {
		return nil, rescode.New(rescode.BadRequest, "Giờ mở/đóng cửa đặc biệt phải có dạng HH:mm (vd 07:00)")
	}
	if openT >= closeT {
		return nil, rescode.New(rescode.BadRequest, "Giờ mở cửa phải trước giờ đóng cửa")
	}
	h.OpenTime, h.CloseTime = &openT, &closeT
	return h, nil
}

// holidayImpacts liệt kê buổi lớp / PT đã xếp (SCHEDULED) bị ảnh hưởng: cả ngày nếu đóng cửa, ngoài khung giờ nếu giờ đặc biệt.
func (s *scheduleExtService) holidayImpacts(ctx context.Context, h *entity.BranchHoliday) ([]*v1.ImpactedClassSchedule, []*v1.ImpactedPTSession, error) {
	if !h.Closed && h.OpenTime == nil {
		return nil, nil, nil
	}
	dayStart := time.Date(h.Date.Year(), h.Date.Month(), h.Date.Day(), 0, 0, 0, 0, subVNLoc)
	dayEnd := dayStart.AddDate(0, 0, 1)
	classes, err := s.repo.ListScheduledClassesBetween(ctx, h.BranchID, dayStart, dayEnd)
	if err != nil {
		return nil, nil, err
	}
	sessions, err := s.repo.ListScheduledPTSessionsBetween(ctx, h.BranchID, dayStart, dayEnd)
	if err != nil {
		return nil, nil, err
	}
	outC := make([]*v1.ImpactedClassSchedule, 0, len(classes))
	for _, c := range classes {
		if holidaySlotViolation(h, c.StartAt, c.EndAt) != "" {
			outC = append(outC, toImpactedClassSchedule(c))
		}
	}
	outP := make([]*v1.ImpactedPTSession, 0, len(sessions))
	for _, p := range sessions {
		if holidaySlotViolation(h, p.StartAt, p.EndAt) != "" {
			outP = append(outP, toImpactedPTSession(p))
		}
	}
	return outC, outP, nil
}

func (s *scheduleExtService) holidayResult(ctx context.Context, h *entity.BranchHoliday) (*v1.BranchHolidayItem, error) {
	row, err := s.repo.FindBranchHolidayByID(ctx, h.ID)
	if err != nil {
		row = &repository.BranchHolidayRow{BranchHoliday: *h, BranchName: "Toàn hệ thống"}
	}
	res := toBranchHolidayItem(*row)
	if res.AffectedClassSchedules, res.AffectedPTSessions, err = s.holidayImpacts(ctx, h); err != nil {
		return nil, err
	}
	return res, nil
}

const holidayDuplicateMsg = "Chi nhánh đã có lịch nghỉ lễ trong ngày này, vui lòng sửa lịch hiện có"

// checkHolidayDate: ngày lễ không được ở quá khứ (giờ VN) và không trùng lịch khác cùng phạm vi/cùng ngày.
func (s *scheduleExtService) checkHolidayDate(ctx context.Context, h *entity.BranchHoliday, excludeID uint) error {
	if h.Date.Format(consts.DateLayout) < s.now().In(subVNLoc).Format(consts.DateLayout) {
		return rescode.New(rescode.BadRequest, "Không thể tạo/chuyển lịch nghỉ lễ về ngày đã qua")
	}
	dup, err := s.repo.BranchHolidayExists(ctx, h.BranchID, h.Date, excludeID)
	if err != nil {
		return err
	}
	if dup {
		return rescode.New(rescode.Conflict, holidayDuplicateMsg)
	}
	return nil
}

func (s *scheduleExtService) CreateBranchHoliday(ctx context.Context, req *v1.CreateBranchHolidayReq) (*v1.BranchHolidayItem, error) {
	if err := checkHolidayScope(ctx, req.BranchID); err != nil {
		return nil, err
	}
	h, err := buildBranchHoliday(req.BranchID, req.Date, req.Name, req.Closed, req.OpenTime, req.CloseTime)
	if err != nil {
		return nil, err
	}
	if err := s.checkHolidayDate(ctx, h, 0); err != nil {
		return nil, err
	}
	h.ExtendSubs = req.ExtendSubs && h.Closed
	var extendedCount int64
	err = s.runTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.CreateBranchHoliday(txCtx, h); err != nil {
			return mapDuplicate(err, holidayDuplicateMsg)
		}
		if h.ExtendSubs {
			cnt, err := s.repo.ExtendActiveTimeSubscriptions(txCtx, h.ID, h.BranchID, h.Date)
			if err != nil {
				return err
			}
			extendedCount = cnt
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	res, err := s.holidayResult(ctx, h)
	if err != nil {
		return nil, err
	}
	res.ExtendedSubscriptions = extendedCount
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityBranchHoliday, h.ID, nil, res)
	}
	return res, nil
}

func holidayBranchKey(b *uint) uint {
	if b == nil {
		return 0
	}
	return *b
}

// UpdateBranchHoliday sửa ngày/tên/trạng thái đóng cửa/giờ đặc biệt. Việc cộng bù gói tập (extend_subs) chỉ chạy một lần
// lúc tạo nên không thay đổi khi sửa.
func (s *scheduleExtService) UpdateBranchHoliday(ctx context.Context, req *v1.UpdateBranchHolidayReq) (*v1.BranchHolidayItem, error) {
	existing, err := s.repo.FindBranchHolidayByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy ngày nghỉ lễ")
		}
		return nil, err
	}
	if err := checkHolidayScope(ctx, existing.BranchID); err != nil {
		return nil, err
	}
	branchID := existing.BranchID
	if req.BranchID != nil {
		branchID = req.BranchID
		if err := checkHolidayScope(ctx, branchID); err != nil {
			return nil, err
		}
	}
	h, err := buildBranchHoliday(branchID, req.Date, req.Name, req.Closed, req.OpenTime, req.CloseTime)
	if err != nil {
		return nil, err
	}
	h.ID = existing.ID
	h.ExtendSubs = existing.ExtendSubs
	h.CreatedAt = existing.CreatedAt
	h.UpdatedAt = s.now()
	sameDay := h.Date.Format(consts.DateLayout) == existing.Date.Format(consts.DateLayout)
	sameScope := holidayBranchKey(h.BranchID) == holidayBranchKey(existing.BranchID)
	// Đã cộng bù gói theo ngày/chi nhánh cũ → không đổi ngày, chi nhánh hay mở cửa lại (xóa rồi tạo lại để hoàn tác).
	if existing.ExtendSubs && (!sameDay || !sameScope || !h.Closed) {
		return nil, rescode.New(rescode.BadRequest,
			"Ngày lễ đã cộng bù gói tập: không thể đổi ngày, chi nhánh hoặc chuyển sang mở cửa. Hãy xóa (hoàn tác cộng bù) rồi tạo lại")
	}
	if !sameDay || !sameScope {
		if err := s.checkHolidayDate(ctx, h, h.ID); err != nil {
			return nil, err
		}
	}
	if err := s.repo.UpdateBranchHoliday(ctx, h); err != nil {
		return nil, mapDuplicate(err, holidayDuplicateMsg)
	}
	res, err := s.holidayResult(ctx, h)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityBranchHoliday, h.ID, toBranchHolidayItem(*existing), res)
	}
	return res, nil
}

func (s *scheduleExtService) DeleteBranchHoliday(ctx context.Context, id uint) error {
	existing, err := s.repo.FindBranchHolidayByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy ngày nghỉ lễ")
		}
		return err
	}
	if existing.BranchID != nil && *existing.BranchID > 0 {
		if !reqctx.CanAccessBranch(ctx, *existing.BranchID) {
			return rescode.New(rescode.PermissionDenied, "Bạn không có quyền xóa ngày nghỉ lễ của chi nhánh này")
		}
	} else if allowed := reqctx.AllowedBranches(ctx); len(allowed) > 0 {
		return rescode.New(rescode.PermissionDenied, "Bạn không có quyền xóa ngày nghỉ lễ cấp toàn hệ thống")
	}
	// Hoàn tác phần cộng bù gói (nếu có) cùng transaction với xóa → tạo/xóa lặp lại không cộng dồn.
	var reverted int64
	err = s.runTx(ctx, func(txCtx context.Context) error {
		if existing.ExtendSubs {
			n, err := s.repo.RevertHolidayExtensions(txCtx, id)
			if err != nil {
				return err
			}
			reverted = n
		}
		return s.repo.DeleteBranchHoliday(txCtx, id)
	})
	if err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityBranchHoliday, id, toBranchHolidayItem(*existing),
			map[string]any{"reverted_subscriptions": reverted})
	}
	return nil
}

func (s *scheduleExtService) PublicHolidays(ctx context.Context, req *v1.PublicHolidaysReq) (v1.PublicHolidaysRes, error) {
	rows, err := s.repo.ListUpcomingHolidays(ctx, req.BranchID, s.now().In(memberVNLoc))
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toBranchHolidayItem), nil
}

// ---- Mappers --------------------------------------------------------------------

func toTrainerAvailabilityItem(r repository.TrainerAvailabilityRow) *v1.TrainerAvailabilityItem {
	return &v1.TrainerAvailabilityItem{
		ID:            r.ID,
		TrainerID:     r.TrainerID,
		BranchID:      r.BranchID,
		BranchName:    r.BranchName,
		DOW:           r.DOW,
		StartTime:     r.StartTime,
		EndTime:       r.EndTime,
		EffectiveFrom: workoutFmtDateOpt(r.EffectiveFrom),
		EffectiveTo:   workoutFmtDateOpt(r.EffectiveTo),
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func toLeaveRequestItem(r repository.LeaveRequestRow) *v1.LeaveRequestItem {
	return &v1.LeaveRequestItem{
		ID:           r.ID,
		StaffID:      r.StaffID,
		StaffName:    r.StaffName,
		BranchID:     r.BranchID,
		BranchName:   r.BranchName,
		FromAt:       r.FromAt,
		ToAt:         r.ToAt,
		Reason:       r.Reason,
		Status:       r.Status,
		ApprovedBy:   r.ApprovedBy,
		ApproverName: r.ApproverName,
		ApprovedAt:   r.ApprovedAt,
		ReviewNote:   r.ReviewNote,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func toImpactedClassSchedule(r repository.ImpactedClassRow) *v1.ImpactedClassSchedule {
	return &v1.ImpactedClassSchedule{
		ID:                    r.ID,
		ClassTypeName:         r.ClassTypeName,
		BranchID:              r.BranchID,
		BranchName:            r.BranchName,
		RoomName:              r.RoomName,
		TrainerID:             r.TrainerID,
		TrainerName:           r.TrainerName,
		SubstituteTrainerID:   r.SubstituteTrainerID,
		SubstituteTrainerName: r.SubstituteTrainerName,
		StartAt:               r.StartAt,
		EndAt:                 r.EndAt,
		BookedCount:           r.BookedCount,
		Status:                r.Status,
	}
}

func toImpactedPTSession(r repository.ImpactedPTSessionRow) *v1.ImpactedPTSession {
	return &v1.ImpactedPTSession{
		ID:                  r.ID,
		ContractID:          r.ContractID,
		MemberID:            r.MemberID,
		MemberCode:          r.MemberCode,
		MemberName:          r.MemberName,
		TrainerID:           r.TrainerID,
		TrainerName:         r.TrainerName,
		OriginalTrainerID:   r.OriginalTrainerID,
		OriginalTrainerName: r.OriginalTrainerName,
		BranchID:            r.BranchID,
		BranchName:          r.BranchName,
		StartAt:             r.StartAt,
		EndAt:               r.EndAt,
		Status:              r.Status,
	}
}

func toBranchHolidayItem(r repository.BranchHolidayRow) *v1.BranchHolidayItem {
	return &v1.BranchHolidayItem{
		ID:         r.ID,
		BranchID:   r.BranchID,
		BranchName: r.BranchName,
		Date:       r.Date.Format(consts.DateLayout),
		Name:       r.Name,
		Closed:     r.Closed,
		OpenTime:   r.OpenTime,
		CloseTime:  r.CloseTime,
		ExtendSubs: r.ExtendSubs,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

// ---- Ràng buộc lịch dùng chung (PT / lớp nhóm) ------------------------------------
//
// ScheduleGuard kiểm tra lịch nghỉ lễ chi nhánh, khung giờ rảnh (trainer_availabilities) và đơn nghỉ phép đã duyệt
// của HLV (PLAN.md – lịch rảnh HLV). Được đăng ký một lần khi khởi tạo module (SetScheduleGuard); khi chưa đăng ký
// (unit test với repo giả) các kiểm tra được bỏ qua.

type ScheduleGuard struct {
	db *gorm.DB
}

var activeScheduleGuard atomic.Pointer[ScheduleGuard]

func NewScheduleGuard(db *gorm.DB) *ScheduleGuard { return &ScheduleGuard{db: db} }

// SetScheduleGuard đăng ký ScheduleGuard dùng cho đặt lịch PT, xếp lịch lớp và khung giờ rảnh trên cổng hội viên.
func SetScheduleGuard(g *ScheduleGuard) { activeScheduleGuard.Store(g) }

func currentScheduleGuard() *ScheduleGuard { return activeScheduleGuard.Load() }

// HolidayOn trả lịch ngày lễ áp dụng cho chi nhánh trong ngày day (giờ VN): lịch riêng chi nhánh được ưu tiên hơn lịch
// toàn hệ thống (cùng thứ tự với check-in). nil nếu ngày thường.
func (g *ScheduleGuard) HolidayOn(ctx context.Context, branchID uint, day time.Time) (*entity.BranchHoliday, error) {
	if g == nil {
		return nil, nil
	}
	var rows []entity.BranchHoliday
	err := postgres.Conn(ctx, g.db).
		Where("date = ? AND (branch_id = ? OR branch_id IS NULL)", day.In(subVNLoc).Format(consts.DateLayout), branchID).
		Order("branch_id NULLS LAST").Order("id").Limit(1).Find(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

// ClosedHoliday trả lịch nghỉ lễ ĐÓNG CỬA của chi nhánh trong ngày day (giờ VN), nil nếu chi nhánh mở cửa.
func (g *ScheduleGuard) ClosedHoliday(ctx context.Context, branchID uint, day time.Time) (*entity.BranchHoliday, error) {
	h, err := g.HolidayOn(ctx, branchID, day)
	if err != nil || h == nil || !h.Closed {
		return nil, err
	}
	return h, nil
}

// holidaySlotViolation trả lý do khung [start, end) không xếp được vì ngày lễ h ("" nếu được):
// đóng cửa cả ngày, hoặc nằm ngoài giờ mở cửa đặc biệt.
func holidaySlotViolation(h *entity.BranchHoliday, start, end time.Time) string {
	if h == nil {
		return ""
	}
	day := start.In(subVNLoc).Format("02/01/2006")
	if h.Closed {
		return fmt.Sprintf("Chi nhánh nghỉ lễ ngày %s (%s)", day, h.Name)
	}
	if h.OpenTime == nil || h.CloseTime == nil || *h.OpenTime == "" || *h.CloseTime == "" {
		return ""
	}
	ls, le := start.In(subVNLoc), end.In(subVNLoc)
	if ls.Format(consts.DateLayout) != le.Format(consts.DateLayout) || ls.Format("15:04") < *h.OpenTime || le.Format("15:04") > *h.CloseTime {
		return fmt.Sprintf("Ngày %s (%s) chi nhánh chỉ mở cửa %s - %s", day, h.Name, *h.OpenTime, *h.CloseTime)
	}
	return ""
}

// CheckBranchOpen trả lỗi BRANCH_HOLIDAY_CLOSED nếu ngày của start là ngày lễ đóng cửa, hoặc [start, end) nằm ngoài
// giờ mở cửa đặc biệt của ngày lễ.
func (g *ScheduleGuard) CheckBranchOpen(ctx context.Context, branchID uint, start, end time.Time) error {
	h, err := g.HolidayOn(ctx, branchID, start)
	if err != nil || h == nil {
		return err
	}
	if msg := holidaySlotViolation(h, start, end); msg != "" {
		return rescode.New(rescode.BranchHolidayClosed, msg+", không thể xếp lịch")
	}
	return nil
}

// CheckTrainerLeave trả lỗi TRAINER_ON_LEAVE nếu HLV có đơn nghỉ phép đã duyệt giao với [start, end)
// (lớp nhóm không áp lịch rảnh PT, chỉ áp nghỉ phép).
func (g *ScheduleGuard) CheckTrainerLeave(ctx context.Context, trainerID uint, start, end time.Time) error {
	if g == nil {
		return nil
	}
	var n int64
	if err := postgres.Conn(ctx, g.db).Model(&entity.StaffLeaveRequest{}).
		Where("staff_id = ? AND status = ? AND from_at < ? AND to_at > ?", trainerID, entity.LeaveStatusApproved, end, start).
		Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return rescode.New(rescode.TrainerOnLeave, "HLV có lịch nghỉ phép đã duyệt trong khung giờ này")
	}
	return nil
}

// trainerDayWindows trả khung giờ rảnh của HLV tại chi nhánh trong ngày day.
// configured=false khi HLV chưa cấu hình lịch rảnh nào tại chi nhánh (không giới hạn – tương thích dữ liệu cũ).
func (g *ScheduleGuard) trainerDayWindows(ctx context.Context, trainerID, branchID uint, day time.Time) (windows []entity.TrainerAvailability, configured bool, err error) {
	var rows []entity.TrainerAvailability
	if err := postgres.Conn(ctx, g.db).Where("trainer_id = ? AND branch_id = ?", trainerID, branchID).Find(&rows).Error; err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	local := day.In(subVNLoc)
	ds := local.Format(consts.DateLayout)
	for _, r := range rows {
		if r.DOW != int(local.Weekday()) {
			continue
		}
		if r.EffectiveFrom != nil && ds < r.EffectiveFrom.Format(consts.DateLayout) {
			continue
		}
		if r.EffectiveTo != nil && ds > r.EffectiveTo.Format(consts.DateLayout) {
			continue
		}
		windows = append(windows, r)
	}
	return windows, true, nil
}

// TrainerSlotChecker trả hàm kiểm tra nhanh một khung [start, end) trong ngày day (dùng khi sinh nhiều khung giờ):
// "" nếu HLV nhận được buổi, ngược lại mã lý do: PT_OUTSIDE_AVAILABILITY | TRAINER_ON_LEAVE.
func (g *ScheduleGuard) TrainerSlotChecker(ctx context.Context, trainerID, branchID uint, day time.Time) (func(start, end time.Time) string, error) {
	if g == nil {
		return func(time.Time, time.Time) string { return "" }, nil
	}
	windows, configured, err := g.trainerDayWindows(ctx, trainerID, branchID, day)
	if err != nil {
		return nil, err
	}
	local := day.In(subVNLoc)
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, subVNLoc)
	var leaves []entity.StaffLeaveRequest
	if err := postgres.Conn(ctx, g.db).
		Where("staff_id = ? AND status = ? AND from_at < ? AND to_at > ?", trainerID, entity.LeaveStatusApproved,
			dayStart.AddDate(0, 0, 1), dayStart).Find(&leaves).Error; err != nil {
		return nil, err
	}
	return func(start, end time.Time) string {
		for _, lv := range leaves {
			if lv.FromAt.Before(end) && lv.ToAt.After(start) {
				return rescode.TrainerOnLeave.Message()
			}
		}
		if !configured {
			return ""
		}
		st, en := start.In(subVNLoc).Format("15:04"), end.In(subVNLoc).Format("15:04")
		if end.In(subVNLoc).Format(consts.DateLayout) != start.In(subVNLoc).Format(consts.DateLayout) {
			return rescode.PTOutsideAvailability.Message()
		}
		for _, w := range windows {
			if st >= w.StartTime && en <= w.EndTime {
				return ""
			}
		}
		return rescode.PTOutsideAvailability.Message()
	}, nil
}

// CheckTrainerSlot trả lỗi nếu HLV nghỉ phép (đã duyệt) hoặc khung giờ nằm ngoài lịch rảnh đã cấu hình.
func (g *ScheduleGuard) CheckTrainerSlot(ctx context.Context, trainerID, branchID uint, start, end time.Time) error {
	if g == nil {
		return nil
	}
	check, err := g.TrainerSlotChecker(ctx, trainerID, branchID, start)
	if err != nil {
		return err
	}
	switch check(start, end) {
	case "":
		return nil
	case rescode.TrainerOnLeave.Message():
		return rescode.New(rescode.TrainerOnLeave, "HLV có lịch nghỉ phép đã duyệt trong khung giờ này")
	default:
		return rescode.New(rescode.PTOutsideAvailability, "Khung giờ nằm ngoài lịch rảnh đã đăng ký của HLV")
	}
}
