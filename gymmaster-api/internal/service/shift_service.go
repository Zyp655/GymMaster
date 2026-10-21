package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/shift/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	shiftOverlapMsg   = "Nhân viên đã có ca trùng giờ"
	shiftMaxDuration  = 16 * time.Hour
	shiftMaxListRange = 62 // ngày
)

type (
	ShiftService interface {
		List(ctx context.Context, req *v1.ListReq) ([]*v1.Shift, error)
		Create(ctx context.Context, in v1.ShiftInput) (*v1.Shift, error)
		Update(ctx context.Context, id uint, in v1.ShiftInput) (*v1.Shift, error)
		Delete(ctx context.Context, id uint) error
		CopyWeek(ctx context.Context, req *v1.CopyWeekReq) (*v1.CopyWeekRes, error)
	}

	shiftService struct {
		shiftRepo repository.ShiftRepository
		tx        postgres.TxManager
		loc       *time.Location
		audit     AuditService
	}
)

const (
	auditEntityShift = "shift"
	auditShiftCopy   = "COPY_WEEK"
)

// NewShiftService: audit tùy chọn (test đơn vị có thể bỏ qua).
func NewShiftService(shiftRepo repository.ShiftRepository, tx postgres.TxManager, audit ...AuditService) ShiftService {
	loc, err := time.LoadLocation(consts.TimeZone)
	if err != nil {
		loc = time.FixedZone("ICT", 7*3600)
	}
	return &shiftService{shiftRepo: shiftRepo, tx: tx, loc: loc, audit: optionalAudit(audit)}
}

func (s *shiftService) List(ctx context.Context, req *v1.ListReq) ([]*v1.Shift, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem lịch ca của chi nhánh này")
	}
	from, err := s.parseDate(req.From)
	if err != nil {
		return nil, err
	}
	to, err := s.parseDate(req.To)
	if err != nil {
		return nil, err
	}
	if to.Before(from) {
		return nil, rescode.New(rescode.BadRequest, "Ngày kết thúc phải sau hoặc bằng ngày bắt đầu")
	}
	if to.Sub(from) > shiftMaxListRange*24*time.Hour {
		return nil, rescode.New(rescode.BadRequest, "Chỉ xem được tối đa 62 ngày mỗi lần")
	}
	items, err := s.shiftRepo.List(ctx, repository.ShiftFilter{
		BranchIDs: branchIDs, StaffID: req.StaffID, From: from, To: to.AddDate(0, 0, 1),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Shift, 0, len(items))
	for i := range items {
		out = append(out, toShiftRes(&items[i]))
	}
	return out, nil
}

func (s *shiftService) Create(ctx context.Context, in v1.ShiftInput) (*v1.Shift, error) {
	sh, err := s.build(ctx, in)
	if err != nil {
		return nil, err
	}
	sh.CreatedBy = reqctx.UserID(ctx)
	sh.UpdatedBy = sh.CreatedBy
	if err := s.shiftRepo.Create(ctx, sh); err != nil {
		return nil, mapShiftErr(err)
	}
	res, err := s.get(ctx, sh.ID)
	if err != nil {
		return nil, err
	}
	auditLogSafe(s.audit, ctx, AuditCreate, auditEntityShift, sh.ID, nil, res)
	return res, nil
}

func (s *shiftService) Update(ctx context.Context, id uint, in v1.ShiftInput) (*v1.Shift, error) {
	cur, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	sh, err := s.build(ctx, in)
	if err != nil {
		return nil, err
	}
	sh.ID = cur.ID
	sh.CreatedAt, sh.CreatedBy = cur.CreatedAt, cur.CreatedBy
	sh.UpdatedBy = reqctx.UserID(ctx)
	if err := s.shiftRepo.Update(ctx, sh); err != nil {
		return nil, mapShiftErr(err)
	}
	res, err := s.get(ctx, id)
	if err != nil {
		return nil, err
	}
	auditLogSafe(s.audit, ctx, AuditUpdate, auditEntityShift, id, toShiftRes(cur), res)
	return res, nil
}

func (s *shiftService) Delete(ctx context.Context, id uint) error {
	cur, err := s.find(ctx, id)
	if err != nil {
		return err
	}
	if err := s.shiftRepo.Delete(ctx, id); err != nil {
		return err
	}
	auditLogSafe(s.audit, ctx, AuditDelete, auditEntityShift, id, toShiftRes(cur), nil)
	return nil
}

func (s *shiftService) CopyWeek(ctx context.Context, req *v1.CopyWeekReq) (*v1.CopyWeekRes, error) {
	if !reqctx.CanAccessBranch(ctx, req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xếp ca cho chi nhánh này")
	}
	from, err := s.parseDate(req.FromWeekStart)
	if err != nil {
		return nil, err
	}
	to, err := s.parseDate(req.ToWeekStart)
	if err != nil {
		return nil, err
	}
	if from.Weekday() != time.Monday || to.Weekday() != time.Monday {
		return nil, rescode.New(rescode.BadRequest, "Ngày bắt đầu tuần phải là thứ Hai")
	}
	if from.Equal(to) {
		return nil, rescode.New(rescode.BadRequest, "Tuần đích phải khác tuần nguồn")
	}
	days := int(to.Sub(from).Hours()) / 24 // giờ Việt Nam không có DST nên luôn là bội số 24h

	src, err := s.shiftRepo.List(ctx, repository.ShiftFilter{
		BranchIDs: []uint{req.BranchID}, From: from, To: from.AddDate(0, 0, 7),
	})
	if err != nil {
		return nil, err
	}

	res := &v1.CopyWeekRes{}
	actor := reqctx.UserID(ctx)
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		belongs := map[uint]bool{}
		for _, it := range src {
			// Chỉ sao chép ca BẮT ĐẦU trong tuần nguồn (ca qua đêm từ tuần trước không tính).
			if it.StartAt.Before(from) {
				continue
			}
			ok, seen := belongs[it.StaffID]
			if !seen {
				var err error
				if ok, err = s.shiftRepo.StaffInBranch(ctx, it.StaffID, req.BranchID); err != nil {
					return err
				}
				belongs[it.StaffID] = ok
			}
			if !ok {
				res.Skipped++
				continue
			}
			sh := &entity.Shift{
				BranchID:  req.BranchID,
				StaffID:   it.StaffID,
				StartAt:   it.StartAt.In(s.loc).AddDate(0, 0, days),
				EndAt:     it.EndAt.In(s.loc).AddDate(0, 0, days),
				Note:      it.Note,
				CreatedBy: actor,
				UpdatedBy: actor,
			}
			created, err := s.shiftRepo.CreateIfFree(ctx, sh)
			if err != nil {
				return err
			}
			if created {
				res.Created++
			} else {
				res.Skipped++
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	auditLogSafe(s.audit, ctx, auditShiftCopy, auditEntityShift, 0,
		map[string]any{"branch_id": req.BranchID, "from_week_start": req.FromWeekStart},
		map[string]any{"to_week_start": req.ToWeekStart, "created": res.Created, "skipped": res.Skipped})
	return res, nil
}

// ---- helpers ----------------------------------------------------------------

func (s *shiftService) parseDate(v string) (time.Time, error) {
	t, err := time.ParseInLocation(consts.DateLayout, v, s.loc)
	if err != nil {
		return time.Time{}, rescode.New(rescode.BadRequest, "Ngày phải có dạng YYYY-MM-DD")
	}
	return t, nil
}

func (s *shiftService) parseTime(v, label string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	// Chấp nhận thêm dạng không có múi giờ (hiểu theo giờ Việt Nam).
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04"} {
		if t, err := time.ParseInLocation(layout, v, s.loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, rescode.New(rescode.BadRequest, label+" không hợp lệ")
}

func (s *shiftService) build(ctx context.Context, in v1.ShiftInput) (*entity.Shift, error) {
	start, err := s.parseTime(in.StartAt, "Giờ bắt đầu")
	if err != nil {
		return nil, err
	}
	end, err := s.parseTime(in.EndAt, "Giờ kết thúc")
	if err != nil {
		return nil, err
	}
	if !end.After(start) {
		return nil, rescode.New(rescode.BadRequest, "Giờ kết thúc phải sau giờ bắt đầu")
	}
	if end.Sub(start) > shiftMaxDuration {
		return nil, rescode.New(rescode.BadRequest, "Một ca làm không được dài quá 16 giờ")
	}
	if !reqctx.CanAccessBranch(ctx, in.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xếp ca cho chi nhánh này")
	}
	ok, err := s.shiftRepo.StaffInBranch(ctx, in.StaffID, in.BranchID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, rescode.New(rescode.BadRequest, "Nhân viên không thuộc chi nhánh của ca làm hoặc đã bị khóa")
	}
	return &entity.Shift{
		BranchID: in.BranchID,
		StaffID:  in.StaffID,
		StartAt:  start,
		EndAt:    end,
		Note:     strings.TrimSpace(in.Note),
	}, nil
}

func (s *shiftService) find(ctx context.Context, id uint) (*entity.ShiftDetail, error) {
	sh, err := s.shiftRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy ca làm")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, sh.BranchID) {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy ca làm")
	}
	return sh, nil
}

func (s *shiftService) get(ctx context.Context, id uint) (*v1.Shift, error) {
	sh, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	return toShiftRes(sh), nil
}

func mapShiftErr(err error) error {
	if isExclusionViolation(err) {
		return rescode.New(rescode.Conflict, shiftOverlapMsg)
	}
	return err
}

func toShiftRes(sh *entity.ShiftDetail) *v1.Shift {
	return &v1.Shift{
		ID:         sh.ID,
		BranchID:   sh.BranchID,
		BranchName: sh.BranchName,
		StaffID:    sh.StaffID,
		StaffName:  sh.StaffName,
		StaffCode:  sh.StaffCode,
		Position:   sh.Position,
		StartAt:    sh.StartAt,
		EndAt:      sh.EndAt,
		Note:       sh.Note,
	}
}
