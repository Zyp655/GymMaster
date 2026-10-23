package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/class/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	classMaxListDays      = 62
	classMaxRecurringDays = 84 // 12 tuần
	classAuditCancel      = "CANCEL"

	classCancelScopeOne       = "one"
	classCancelScopeFollowing = "following"

	classRoomOverlapMsg    = "Phòng đã có lớp khác trùng giờ"
	classTrainerOverlapMsg = "HLV đã có lớp khác trùng giờ"
	classTrainerPTMsg      = "HLV đã có buổi PT trùng giờ"
	classOverlapMsg        = "Trùng lịch phòng hoặc HLV"
	classPastMsg           = "Thời điểm đã qua"
)

type (
	ClassScheduleService interface {
		List(ctx context.Context, req *v1.ListScheduleReq) ([]*v1.ClassSchedule, error)
		Get(ctx context.Context, id uint) (*v1.ClassSchedule, error)
		Create(ctx context.Context, req *v1.CreateScheduleReq) (*v1.ClassSchedule, error)
		CreateRecurring(ctx context.Context, req *v1.CreateRecurringReq) (*v1.CreateRecurringRes, error)
		Update(ctx context.Context, req *v1.UpdateScheduleReq) (*v1.ClassSchedule, error)
		Cancel(ctx context.Context, id uint, reason, scope string) (*v1.CancelScheduleRes, error)
		Bookings(ctx context.Context, id uint, status string) ([]*v1.ClassBooking, error)
	}

	classScheduleService struct {
		scheduleRepo repository.ClassScheduleRepository
		bookingRepo  repository.ClassBookingRepository
		tx           postgres.TxManager
		audit        AuditService
		notifier     Notifier
		now          func() time.Time
		loc          *time.Location
	}

	// classSlot là các thông tin đã kiểm tra hợp lệ, dùng chung cho tạo/sửa/tạo lặp.
	classSlot struct {
		classType *entity.ClassType
		branch    *entity.Branch
		room      *entity.Room
		trainerID uint
		duration  time.Duration
		capacity  int
	}
)

func NewClassScheduleService(
	scheduleRepo repository.ClassScheduleRepository,
	bookingRepo repository.ClassBookingRepository,
	tx postgres.TxManager,
	audit AuditService,
	notifier Notifier,
) ClassScheduleService {
	return &classScheduleService{
		scheduleRepo: scheduleRepo,
		bookingRepo:  bookingRepo,
		tx:           tx,
		audit:        audit,
		notifier:     notifier,
		now:          time.Now,
		loc:          classLocation(),
	}
}

func (s *classScheduleService) List(ctx context.Context, req *v1.ListScheduleReq) ([]*v1.ClassSchedule, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem lịch lớp của chi nhánh này")
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
	if to.Sub(from) > classMaxListDays*24*time.Hour {
		return nil, rescode.New(rescode.BadRequest, "Chỉ xem được tối đa 62 ngày mỗi lần")
	}
	trainerID := req.TrainerID
	if trainerID == 0 && classTrainerOnly(ctx) {
		trainerID = reqctx.Auth(ctx).UserID
	}
	items, err := s.scheduleRepo.List(ctx, repository.ClassScheduleFilter{
		BranchIDs: branchIDs, From: from, To: to.AddDate(0, 0, 1),
		TrainerID: trainerID, ClassTypeID: req.ClassTypeID, RoomID: req.RoomID, Status: req.Status,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*v1.ClassSchedule, 0, len(items))
	for i := range items {
		out = append(out, classScheduleToRes(&items[i]))
	}
	return out, nil
}

func (s *classScheduleService) Get(ctx context.Context, id uint) (*v1.ClassSchedule, error) {
	d, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	return classScheduleToRes(d), nil
}

func (s *classScheduleService) Create(ctx context.Context, req *v1.CreateScheduleReq) (*v1.ClassSchedule, error) {
	start, err := classParseTime(req.StartAt, s.loc)
	if err != nil {
		return nil, err
	}
	if !start.After(s.now()) {
		return nil, rescode.New(rescode.BadRequest, "Không thể tạo lịch lớp trong quá khứ")
	}
	slot, err := s.prepare(ctx, req.ClassTypeID, req.BranchID, req.RoomID, req.TrainerID, req.DurationMin, req.Capacity, true)
	if err != nil {
		return nil, err
	}
	end := start.Add(slot.duration)
	if err := s.checkSlot(ctx, slot, start, end, 0, slot.trainerID); err != nil {
		return nil, err
	}
	actor := reqctx.UserID(ctx)
	sch := &entity.ClassSchedule{
		ClassTypeID: slot.classType.ID,
		BranchID:    slot.branch.ID,
		RoomID:      slot.room.ID,
		TrainerID:   slot.trainerID,
		StartAt:     start,
		EndAt:       end,
		Capacity:    slot.capacity,
		Status:      entity.ClassScheduleScheduled,
		Note:        strings.TrimSpace(req.Note),
		CreatedBy:   actor,
		UpdatedBy:   actor,
	}
	if err := s.scheduleRepo.Create(ctx, sch); err != nil {
		return nil, classMapOverlapErr(err)
	}
	s.audit.Log(ctx, AuditCreate, "class_schedule", sch.ID, nil, sch)
	return s.Get(ctx, sch.ID)
}

func (s *classScheduleService) CreateRecurring(ctx context.Context, req *v1.CreateRecurringReq) (*v1.CreateRecurringRes, error) {
	startDate, err := s.parseDate(req.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := s.parseDate(req.EndDate)
	if err != nil {
		return nil, err
	}
	if endDate.Before(startDate) {
		return nil, rescode.New(rescode.BadRequest, "Ngày kết thúc phải sau hoặc bằng ngày bắt đầu")
	}
	if endDate.After(startDate.AddDate(0, 0, classMaxRecurringDays-1)) {
		return nil, rescode.New(rescode.BadRequest, "Lịch lặp tối đa 12 tuần")
	}
	weekdays := map[int]bool{}
	for _, d := range req.Weekdays {
		if d < 1 || d > 7 {
			return nil, rescode.New(rescode.BadRequest, "Thứ trong tuần phải từ 1 (Thứ Hai) đến 7 (Chủ nhật)")
		}
		weekdays[d] = true
	}
	if len(weekdays) == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn thứ trong tuần")
	}
	tod, err := time.Parse("15:04", strings.TrimSpace(req.StartTime))
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Giờ bắt đầu phải có dạng HH:mm")
	}
	slot, err := s.prepare(ctx, req.ClassTypeID, req.BranchID, req.RoomID, req.TrainerID, req.DurationMin, req.Capacity, true)
	if err != nil {
		return nil, err
	}
	// Giờ mở cửa giống nhau cho mọi buổi → kiểm tra một lần.
	sample := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), tod.Hour(), tod.Minute(), 0, 0, s.loc)
	if err := classCheckOpeningHours(slot.branch, sample, sample.Add(slot.duration), s.loc); err != nil {
		return nil, err
	}

	now := s.now()
	seriesID := uuid.NewString()
	actor := reqctx.UserID(ctx)
	note := strings.TrimSpace(req.Note)
	res := &v1.CreateRecurringRes{ScheduleIDs: []uint{}, Skipped: []*v1.RecurringSkipped{}}
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			wd := int(d.Weekday())
			if wd == 0 {
				wd = 7
			}
			if !weekdays[wd] {
				continue
			}
			start := time.Date(d.Year(), d.Month(), d.Day(), tod.Hour(), tod.Minute(), 0, 0, s.loc)
			end := start.Add(slot.duration)
			day := d.Format(consts.DateLayout)
			skip := func(reason string) {
				res.Skipped = append(res.Skipped, &v1.RecurringSkipped{Date: day, Reason: reason})
			}
			if !start.After(now) {
				skip(classPastMsg)
				continue
			}
			// Ngày lễ đóng cửa / ngoài giờ đặc biệt, hoặc HLV nghỉ phép đã duyệt → bỏ qua buổi này.
			if h, err := currentScheduleGuard().HolidayOn(ctx, slot.branch.ID, start); err != nil {
				return err
			} else if msg := holidaySlotViolation(h, start, end); msg != "" {
				skip(msg)
				continue
			}
			if err := currentScheduleGuard().CheckTrainerLeave(ctx, slot.trainerID, start, end); err != nil {
				if gerror.Code(err) == rescode.TrainerOnLeave {
					skip("HLV nghỉ phép")
					continue
				}
				return err
			}
			reason, err := s.slotConflict(ctx, slot, start, end, 0, slot.trainerID)
			if err != nil {
				return err
			}
			if reason != "" {
				skip(reason)
				continue
			}
			sid := seriesID
			sch := &entity.ClassSchedule{
				ClassTypeID: slot.classType.ID,
				BranchID:    slot.branch.ID,
				RoomID:      slot.room.ID,
				TrainerID:   slot.trainerID,
				StartAt:     start,
				EndAt:       end,
				Capacity:    slot.capacity,
				Status:      entity.ClassScheduleScheduled,
				SeriesID:    &sid,
				Note:        note,
				CreatedBy:   actor,
				UpdatedBy:   actor,
			}
			created, err := s.scheduleRepo.CreateIfFree(ctx, sch)
			if err != nil {
				return err
			}
			if !created {
				skip(classOverlapMsg)
				continue
			}
			res.ScheduleIDs = append(res.ScheduleIDs, sch.ID)
		}
		res.Created = len(res.ScheduleIDs)
		if res.Created > 0 {
			res.SeriesID = seriesID
			s.audit.Log(ctx, AuditCreate, "class_schedule_series", 0, nil, g.Map{
				"series_id": seriesID, "schedule_ids": res.ScheduleIDs, "request": req, "skipped": res.Skipped,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *classScheduleService) Update(ctx context.Context, req *v1.UpdateScheduleReq) (*v1.ClassSchedule, error) {
	start, err := classParseTime(req.StartAt, s.loc)
	if err != nil {
		return nil, err
	}
	now := s.now()
	var promoted []uint
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		sch, err := s.lock(ctx, req.ID)
		if err != nil {
			return err
		}
		if sch.Status != entity.ClassScheduleScheduled {
			return rescode.New(rescode.BadRequest, "Chỉ sửa được buổi lớp chưa bị hủy và chưa hoàn thành")
		}
		if !sch.StartAt.After(now) {
			return rescode.New(rescode.BadRequest, "Buổi lớp đã bắt đầu, không thể sửa")
		}
		if !start.After(now) {
			return rescode.New(rescode.BadRequest, "Không thể dời lịch lớp về quá khứ")
		}
		slot, err := s.prepare(ctx, sch.ClassTypeID, sch.BranchID, req.RoomID, req.TrainerID, req.DurationMin, req.Capacity, false)
		if err != nil {
			return err
		}
		if slot.capacity < sch.BookedCount {
			return rescode.New(rescode.BadRequest,
				fmt.Sprintf("Sức chứa không được nhỏ hơn số chỗ đã đặt (%d)", sch.BookedCount))
		}
		end := start.Add(slot.duration)
		// HLV thực dạy: HLV dạy thay (nếu đã phân công và không đổi HLV chính), ngược lại HLV chính.
		teacherID := slot.trainerID
		if sch.SubstituteTrainerID != nil && slot.trainerID == sch.TrainerID {
			teacherID = *sch.SubstituteTrainerID
		}
		if err := s.checkSlot(ctx, slot, start, end, sch.ID, teacherID); err != nil {
			return err
		}
		before := *sch
		sch.RoomID = slot.room.ID
		if slot.trainerID != sch.TrainerID {
			// Đổi HLV chính → bỏ phân công dạy thay cũ (HLV thực dạy là HLV chính mới đã kiểm tra ở trên).
			sch.SubstituteTrainerID = nil
		}
		sch.TrainerID = slot.trainerID
		sch.StartAt = start
		sch.EndAt = end
		sch.Capacity = slot.capacity
		sch.Note = strings.TrimSpace(req.Note)
		sch.UpdatedBy = reqctx.UserID(ctx)
		if err := s.scheduleRepo.Update(ctx, sch); err != nil {
			return classMapOverlapErr(err)
		}
		// Tăng sức chứa → tự đẩy người trong danh sách chờ lên.
		if free := sch.Capacity - sch.BookedCount; free > 0 {
			if promoted, err = bookingPromoteWaitlist(ctx, s.scheduleRepo, s.bookingRepo, sch.ID, free); err != nil {
				return err
			}
		}
		s.audit.Log(ctx, AuditUpdate, "class_schedule", sch.ID, before, sch)
		return nil
	})
	if err != nil {
		return nil, err
	}
	bookingNotifyPromoted(ctx, s.notifier, s.bookingRepo, promoted, s.loc)
	return s.Get(ctx, req.ID)
}

func (s *classScheduleService) Cancel(ctx context.Context, id uint, reason, scope string) (*v1.CancelScheduleRes, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập lý do hủy")
	}
	if scope == "" {
		scope = classCancelScopeOne
	}
	if scope != classCancelScopeOne && scope != classCancelScopeFollowing {
		return nil, rescode.New(rescode.BadRequest, "Phạm vi hủy phải là one hoặc following")
	}
	now := s.now()
	actor := reqctx.UserID(ctx)
	type cancelled struct {
		detail  *entity.ClassScheduleDetail
		userIDs []uint
	}
	var done []cancelled
	res := &v1.CancelScheduleRes{ScheduleIDs: []uint{}}

	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		sch, err := s.lock(ctx, id)
		if err != nil {
			return err
		}
		if sch.Status != entity.ClassScheduleScheduled {
			return rescode.New(rescode.BadRequest, "Buổi lớp đã bị hủy hoặc đã hoàn thành")
		}
		if !sch.StartAt.After(now) {
			return rescode.New(rescode.BadRequest, "Không thể hủy buổi lớp đã diễn ra")
		}
		targets := []uint{sch.ID}
		if scope == classCancelScopeFollowing && sch.SeriesID != nil {
			series, err := s.scheduleRepo.SeriesFrom(ctx, *sch.SeriesID, sch.StartAt)
			if err != nil {
				return err
			}
			for _, x := range series {
				if x.ID != sch.ID && x.StartAt.After(now) && x.BranchID == sch.BranchID {
					targets = append(targets, x.ID)
				}
			}
		}
		for _, tid := range targets {
			cur, err := s.scheduleRepo.FindForUpdate(ctx, tid)
			if err != nil {
				return err
			}
			if cur.Status != entity.ClassScheduleScheduled {
				continue
			}
			detail, err := s.scheduleRepo.FindByID(ctx, tid)
			if err != nil {
				return err
			}
			bookings, err := s.bookingRepo.ListActiveBySchedule(ctx, tid)
			if err != nil {
				return err
			}
			var userIDs []uint
			for i := range bookings {
				b := bookings[i].ClassBooking
				b.Status = entity.BookingStatusCancelled
				b.CancelledAt = &now
				b.WaitlistPosition = nil
				if err := s.bookingRepo.UpdateStatus(ctx, &b); err != nil {
					return err
				}
				if uid := bookings[i].MemberUserID; uid != nil && !slices.Contains(userIDs, *uid) {
					userIDs = append(userIDs, *uid)
				}
			}
			if err := s.scheduleRepo.MarkCancelled(ctx, tid, reason, actor); err != nil {
				return err
			}
			s.audit.Log(ctx, classAuditCancel, "class_schedule", tid,
				g.Map{"status": cur.Status, "booked_count": cur.BookedCount},
				g.Map{"status": entity.ClassScheduleCancelled, "cancel_reason": reason, "scope": scope, "cancelled_bookings": len(bookings)})
			res.CancelledSchedules++
			res.CancelledBookings += len(bookings)
			res.ScheduleIDs = append(res.ScheduleIDs, tid)
			done = append(done, cancelled{detail: detail, userIDs: userIDs})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, c := range done {
		if len(c.userIDs) == 0 {
			continue
		}
		d := c.detail
		title := "Lớp " + d.ClassTypeName + " đã bị hủy"
		body := fmt.Sprintf("Buổi lớp %s lúc %s tại %s (%s) đã bị hủy. Lý do: %s",
			d.ClassTypeName, d.StartAt.In(s.loc).Format("15:04 02/01/2006"), d.RoomName, d.BranchName, reason)
		if err := s.notifier.Notify(ctx, c.userIDs, NotifyClassCancelled, title, body, bookingNotifyLink); err != nil {
			g.Log().Warningf(ctx, "class cancelled notify: %v", err)
		}
	}
	return res, nil
}

func (s *classScheduleService) Bookings(ctx context.Context, id uint, status string) ([]*v1.ClassBooking, error) {
	if _, err := s.find(ctx, id); err != nil {
		return nil, err
	}
	items, err := s.bookingRepo.ListBySchedule(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.ClassBooking, 0, len(items))
	for i := range items {
		if status != "" && items[i].Status != status {
			continue
		}
		out = append(out, ClassBookingToRes(&items[i]))
	}
	return out, nil
}

// ---- helpers ----------------------------------------------------------------

// prepare kiểm tra loại lớp, chi nhánh, phòng, HLV, thời lượng, sức chứa.
// newSchedule = true: loại lớp phải đang hoạt động.
func (s *classScheduleService) prepare(ctx context.Context, classTypeID, branchID, roomID, trainerID uint,
	durationMin, capacity int, newSchedule bool) (*classSlot, error) {
	if !reqctx.CanAccessBranch(ctx, branchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xếp lịch lớp cho chi nhánh này")
	}
	ct, err := s.scheduleRepo.FindClassType(ctx, classTypeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.BadRequest, "Loại lớp không tồn tại")
		}
		return nil, err
	}
	if newSchedule && ct.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Loại lớp đã ngừng hoạt động")
	}
	branch, err := s.scheduleRepo.FindBranch(ctx, branchID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.BadRequest, "Chi nhánh không tồn tại")
		}
		return nil, err
	}
	if branch.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh đang ngừng hoạt động")
	}
	room, err := s.scheduleRepo.FindRoom(ctx, roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.BadRequest, "Phòng không tồn tại")
		}
		return nil, err
	}
	if room.BranchID != branchID {
		return nil, rescode.New(rescode.BadRequest, "Phòng không thuộc chi nhánh của lớp")
	}
	if room.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Phòng đang ngừng hoạt động")
	}
	if durationMin == 0 {
		durationMin = ct.DefaultDurationMin
	}
	if durationMin < 15 || durationMin > 240 {
		return nil, rescode.New(rescode.BadRequest, "Thời lượng từ 15 đến 240 phút")
	}
	if capacity == 0 {
		capacity = min(ct.DefaultCapacity, room.Capacity)
	}
	if capacity < 1 {
		return nil, rescode.New(rescode.BadRequest, "Sức chứa phải lớn hơn 0")
	}
	if capacity > room.Capacity {
		return nil, rescode.New(rescode.BadRequest, fmt.Sprintf("Sức chứa vượt quá sức chứa phòng (%d)", room.Capacity))
	}
	ok, err := s.scheduleRepo.TrainerInBranch(ctx, trainerID, branchID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, rescode.New(rescode.BadRequest, "HLV không thuộc chi nhánh, không phải HLV hoặc đã bị khóa")
	}
	return &classSlot{
		classType: ct, branch: branch, room: room, trainerID: trainerID,
		duration: time.Duration(durationMin) * time.Minute, capacity: capacity,
	}, nil
}

// checkSlot dùng cho tạo/sửa một buổi: giờ mở cửa, ngày lễ (đóng cửa / giờ đặc biệt), HLV thực dạy (teacherID)
// không nghỉ phép đã duyệt, trùng lịch (phòng, HLV, PT).
func (s *classScheduleService) checkSlot(ctx context.Context, slot *classSlot, start, end time.Time, excludeID, teacherID uint) error {
	if err := classCheckOpeningHours(slot.branch, start, end, s.loc); err != nil {
		return err
	}
	guard := currentScheduleGuard()
	if err := guard.CheckBranchOpen(ctx, slot.branch.ID, start, end); err != nil {
		return err
	}
	if err := guard.CheckTrainerLeave(ctx, teacherID, start, end); err != nil {
		return err
	}
	reason, err := s.slotConflict(ctx, slot, start, end, excludeID, teacherID)
	if err != nil {
		return err
	}
	if reason != "" {
		return rescode.New(rescode.Conflict, reason)
	}
	return nil
}

// slotConflict trả lý do trùng lịch ("" nếu trống). teacherID là HLV thực dạy của buổi; so với các lớp khác theo HLV thực
// dạy của chúng (kể cả lớp HLV đang dạy thay) và buổi PT của HLV.
func (s *classScheduleService) slotConflict(ctx context.Context, slot *classSlot, start, end time.Time, excludeID, teacherID uint) (string, error) {
	busy, err := s.scheduleRepo.RoomOverlap(ctx, slot.room.ID, start, end, excludeID)
	if err != nil || busy {
		return classRoomOverlapMsg, err
	}
	busy, err = s.scheduleRepo.TrainerOverlap(ctx, teacherID, start, end, excludeID)
	if err != nil || busy {
		return classTrainerOverlapMsg, err
	}
	busy, err = s.scheduleRepo.TrainerPTOverlap(ctx, teacherID, start, end)
	if err != nil || busy {
		return classTrainerPTMsg, err
	}
	return "", nil
}

func (s *classScheduleService) find(ctx context.Context, id uint) (*entity.ClassScheduleDetail, error) {
	d, err := s.scheduleRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy buổi lớp")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, d.BranchID) {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy buổi lớp")
	}
	return d, nil
}

func (s *classScheduleService) lock(ctx context.Context, id uint) (*entity.ClassSchedule, error) {
	sch, err := s.scheduleRepo.FindForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy buổi lớp")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, sch.BranchID) {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy buổi lớp")
	}
	return sch, nil
}

func (s *classScheduleService) parseDate(v string) (time.Time, error) {
	t, err := time.ParseInLocation(consts.DateLayout, strings.TrimSpace(v), s.loc)
	if err != nil {
		return time.Time{}, rescode.New(rescode.BadRequest, "Ngày phải có dạng YYYY-MM-DD")
	}
	return t, nil
}

// classTrainerOnly: người gọi chỉ có vai trò HLV và không có quyền class.manage.
func classTrainerOnly(ctx context.Context) bool {
	p := reqctx.Auth(ctx)
	if p == nil || len(p.Roles) == 0 || reqctx.HasPermission(ctx, consts.PermClassManage) {
		return false
	}
	for _, r := range p.Roles {
		if r != consts.RoleTrainer {
			return false
		}
	}
	return true
}

func classParseTime(v string, loc *time.Location) (time.Time, error) {
	v = strings.TrimSpace(v)
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04"} {
		if t, err := time.ParseInLocation(layout, v, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, rescode.New(rescode.BadRequest, "Giờ bắt đầu không hợp lệ")
}

// classCheckOpeningHours: buổi lớp phải nằm trọn trong giờ mở cửa của chi nhánh (trong cùng một ngày).
func classCheckOpeningHours(b *entity.Branch, start, end time.Time, loc *time.Location) error {
	ls, le := start.In(loc), end.In(loc)
	if ls.Format(consts.DateLayout) != le.Format(consts.DateLayout) ||
		ls.Format("15:04") < b.OpenTime || le.Format("15:04") > b.CloseTime {
		return rescode.New(rescode.BadRequest,
			fmt.Sprintf("Lịch lớp phải nằm trong giờ mở cửa của chi nhánh (%s - %s)", b.OpenTime, b.CloseTime))
	}
	return nil
}

func classMapOverlapErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23P01" {
		switch pgErr.ConstraintName {
		case "ex_class_room_overlap":
			return rescode.New(rescode.Conflict, classRoomOverlapMsg)
		case "ex_class_trainer_overlap":
			return rescode.New(rescode.Conflict, classTrainerOverlapMsg)
		}
		return rescode.New(rescode.Conflict, classOverlapMsg)
	}
	return err
}

func classScheduleToRes(d *entity.ClassScheduleDetail) *v1.ClassSchedule {
	series := ""
	if d.SeriesID != nil {
		series = *d.SeriesID
	}
	return &v1.ClassSchedule{
		ID:                    d.ID,
		ClassTypeID:           d.ClassTypeID,
		ClassTypeCode:         d.ClassTypeCode,
		ClassTypeName:         d.ClassTypeName,
		ClassTypeColor:        d.ClassTypeColor,
		BranchID:              d.BranchID,
		BranchName:            d.BranchName,
		RoomID:                d.RoomID,
		RoomName:              d.RoomName,
		TrainerID:             d.TrainerID,
		TrainerName:           d.TrainerName,
		StartAt:               d.StartAt,
		SubstituteTrainerID:   d.SubstituteTrainerID,
		SubstituteTrainerName: d.SubstituteTrainerName,
		EndAt:                 d.EndAt,
		DurationMin:           int(d.EndAt.Sub(d.StartAt).Minutes()),
		Capacity:              d.Capacity,
		BookedCount:           d.BookedCount,
		WaitlistCount:         d.WaitlistCount,
		AttendedCount:         d.AttendedCount,
		AvailableSlots:        max(d.Capacity-d.BookedCount, 0),
		Status:                d.Status,
		SeriesID:              series,
		CancelReason:          d.CancelReason,
		Note:                  d.Note,
		CreatedAt:             d.CreatedAt,
		UpdatedAt:             d.UpdatedAt,
	}
}

// ClassBookingToRes chuyển lượt đặt chỗ sang DTO (dùng lại được ở module cổng hội viên).
func ClassBookingToRes(d *entity.ClassBookingDetail) *v1.ClassBooking {
	return &v1.ClassBooking{
		ID:               d.ID,
		ScheduleID:       d.ScheduleID,
		MemberID:         d.MemberID,
		MemberCode:       d.MemberCode,
		MemberName:       d.MemberName,
		MemberPhone:      d.MemberPhone,
		SubscriptionID:   d.SubscriptionID,
		Status:           d.Status,
		WaitlistPosition: d.WaitlistPosition,
		WaitlistRank:     d.WaitlistRank,
		Source:           d.Source,
		BookedAt:         d.BookedAt,
		CancelledAt:      d.CancelledAt,
		AttendedAt:       d.AttendedAt,
		ClassTypeID:      d.ClassTypeID,
		ClassTypeName:    d.ClassTypeName,
		ClassTypeColor:   d.ClassTypeColor,
		BranchID:         d.BranchID,
		BranchName:       d.BranchName,
		RoomName:         d.RoomName,
		TrainerID:        d.TrainerID,
		TrainerName:      d.TrainerName,
		StartAt:          d.StartAt,
		EndAt:            d.EndAt,
		ScheduleStatus:   d.ScheduleStatus,
	}
}
