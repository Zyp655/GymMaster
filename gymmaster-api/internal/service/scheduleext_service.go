package service

import (
	"context"
	"errors"
	"strings"
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
)

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

// ---- Trainer Availability -------------------------------------------------------

func (s *scheduleExtService) GetTrainerAvailability(ctx context.Context, req *v1.GetTrainerAvailabilityReq) (v1.GetTrainerAvailabilityRes, error) {
	rows, err := s.repo.ListTrainerAvailability(ctx, req.ID, req.BranchID)
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toTrainerAvailabilityItem), nil
}

func (s *scheduleExtService) SaveTrainerAvailability(ctx context.Context, req *v1.SaveTrainerAvailabilityReq) (v1.SaveTrainerAvailabilityRes, error) {
	inSlots := req.Slots
	if len(inSlots) == 0 && len(req.Items) > 0 {
		inSlots = req.Items
	}
	slots := make([]entity.TrainerAvailability, 0, len(inSlots))
	for _, sl := range inSlots {
		if sl.DOW < 0 || sl.DOW > 6 {
			return nil, rescode.New(rescode.BadRequest, "Thứ trong tuần phải từ 0 đến 6")
		}
		st := strings.TrimSpace(sl.StartTime)
		et := strings.TrimSpace(sl.EndTime)
		if st == "" || et == "" || st >= et {
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
		bID := sl.BranchID
		if bID == 0 {
			bID = req.BranchID
		}
		if bID == 0 {
			bID = 1
		}
		slots = append(slots, entity.TrainerAvailability{
			TrainerID:     req.ID,
			BranchID:      bID,
			DOW:           sl.DOW,
			StartTime:     st,
			EndTime:       et,
			EffectiveFrom: effFrom,
			EffectiveTo:   effTo,
		})
	}
	err := s.runTx(ctx, func(txCtx context.Context) error {
		return s.repo.ReplaceTrainerAvailability(txCtx, req.ID, req.BranchID, slots)
	})
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityTrainerAvailability, req.ID, nil, slots)
	}
	return s.GetTrainerAvailability(ctx, &v1.GetTrainerAvailabilityReq{ID: req.ID, BranchID: req.BranchID})
}

// ---- Leave Requests -------------------------------------------------------------

func (s *scheduleExtService) ListLeaveRequests(ctx context.Context, req *v1.ListLeaveRequestsReq) (*v1.ListLeaveRequestsRes, error) {
	q := req.PageReq.ToQuery(leaveRequestOrderable, "id")
	q.OrderBy = "lr." + q.OrderBy
	rows, total, err := s.repo.ListLeaveRequests(ctx, q, req.StaffID, req.BranchID, strings.TrimSpace(req.Status))
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, toLeaveRequestItem), total, q), nil
}

func (s *scheduleExtService) CreateLeaveRequest(ctx context.Context, req *v1.CreateLeaveRequestReq) (*v1.LeaveRequestItem, error) {
	staffID := req.StaffID
	if staffID == 0 {
		if uid := reqctx.UserID(ctx); uid != nil {
			staffID = *uid
		}
	}
	if staffID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn nhân viên/HLV")
	}
	fromAt, err := workoutParsePerformedAt(req.FromAt, s.now())
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Thời gian bắt đầu nghỉ không hợp lệ")
	}
	toAt, err := workoutParsePerformedAt(req.ToAt, fromAt)
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

func (s *scheduleExtService) SubstituteClassTrainer(ctx context.Context, req *v1.SubstituteClassTrainerReq) (*v1.ImpactedClassSchedule, error) {
	subID := req.SubstituteTrainerID
	if subID == 0 {
		subID = req.TrainerID
	}
	if subID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn huấn luyện viên dạy thay")
	}
	if _, err := s.repo.FindClassScheduleByID(ctx, req.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy buổi lớp học")
		}
		return nil, err
	}
	if err := s.repo.SetClassSubstituteTrainer(ctx, req.ID, &subID, strings.TrimSpace(req.Note), reqctx.UserID(ctx)); err != nil {
		return nil, err
	}
	updated, err := s.repo.FindClassScheduleByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return toImpactedClassSchedule(*updated), nil
}

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
	origTrainerID := sess.TrainerID
	if sess.OriginalTrainerID != nil && *sess.OriginalTrainerID > 0 {
		origTrainerID = *sess.OriginalTrainerID
	}
	if err := s.repo.ReassignPTSessionTrainer(ctx, req.ID, newTrainerID, origTrainerID); err != nil {
		return nil, err
	}
	updated, err := s.repo.FindPTSessionByID(ctx, req.ID)
	if err != nil {
		return nil, err
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

func (s *scheduleExtService) CreateBranchHoliday(ctx context.Context, req *v1.CreateBranchHolidayReq) (*v1.BranchHolidayItem, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên ngày nghỉ lễ")
	}
	date, err := time.Parse(consts.DateLayout, strings.TrimSpace(req.Date))
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Ngày nghỉ lễ phải có định dạng YYYY-MM-DD")
	}
	closed := true
	if req.Closed != nil {
		closed = *req.Closed
	}
	h := &entity.BranchHoliday{
		BranchID:   req.BranchID,
		Date:       date,
		Name:       name,
		Closed:     closed,
		OpenTime:   req.OpenTime,
		CloseTime:  req.CloseTime,
		ExtendSubs: req.ExtendSubs,
	}
	var extendedCount int64
	err = s.runTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.CreateBranchHoliday(txCtx, h); err != nil {
			return err
		}
		if h.Closed && h.ExtendSubs {
			cnt, err := s.repo.ExtendActiveTimeSubscriptions(txCtx, h.BranchID, h.Date)
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
	row, err := s.repo.FindBranchHolidayByID(ctx, h.ID)
	if err != nil {
		row = &repository.BranchHolidayRow{BranchHoliday: *h, BranchName: "Toàn hệ thống"}
	}
	res := toBranchHolidayItem(*row)
	res.ExtendedSubscriptions = extendedCount
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityBranchHoliday, h.ID, nil, res)
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
	if err := s.repo.DeleteBranchHoliday(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityBranchHoliday, id, toBranchHolidayItem(*existing), nil)
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
