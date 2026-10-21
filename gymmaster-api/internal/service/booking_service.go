package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/gorm"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Giá trị mặc định khi bảng settings chưa có khóa tương ứng.
const (
	bookingDefaultAdvanceDays   = 7
	bookingDefaultCancelMinutes = 120
	bookingDefaultNoShowLimit   = 3
	bookingDefaultBlockDays     = 7

	// Điểm danh mở từ 30 phút trước giờ học đến 24 giờ sau khi lớp kết thúc.
	bookingAttendanceOpenBefore = 30 * time.Minute
	bookingAttendanceCloseAfter = 24 * time.Hour
	// Lượt BOOKED chuyển NO_SHOW khi lớp đã kết thúc quá 30 phút.
	bookingNoShowGrace = 30 * time.Minute
	// Cửa sổ đếm số lần NO_SHOW để khóa đặt lớp.
	bookingNoShowWindowDays = 30

	bookingDuplicateMsg = "Hội viên đã đặt buổi lớp này"
	bookingNotifyLink   = "/member/bookings"
)

// BookingAttendance là một dòng điểm danh.
type BookingAttendance struct {
	BookingID uint
	Status    string // ATTENDED | NO_SHOW
}

type (
	// BookingService là nghiệp vụ đặt chỗ lớp nhóm dùng chung cho quầy lễ tân (admin)
	// và cổng hội viên. Mọi thao tác thay đổi chỗ đều khóa dòng class_schedules (SELECT ... FOR UPDATE)
	// trong transaction để không vượt sức chứa khi có nhiều yêu cầu đồng thời.
	BookingService interface {
		// Book đặt chỗ cho hội viên. source: COUNTER (nhân viên đặt hộ, kiểm tra phạm vi chi nhánh)
		// hoặc PORTAL (hội viên tự đặt). Còn chỗ → BOOKED, hết chỗ → WAITLIST.
		Book(ctx context.Context, scheduleID, memberID uint, source string) (*entity.ClassBooking, error)
		// Cancel hủy lượt đặt. byMember = true: áp dụng hạn hủy booking.cancel_before_minutes
		// (người gọi phải tự xác minh lượt đặt thuộc hội viên đang đăng nhập).
		// byMember = false: nhân viên hủy hộ, không giới hạn thời gian, kiểm tra phạm vi chi nhánh.
		// Hủy một lượt BOOKED sẽ tự đẩy người đầu danh sách chờ lên BOOKED và gửi thông báo.
		Cancel(ctx context.Context, bookingID uint, byMember bool) error
		// ListMemberBookings: lượt đặt của hội viên có buổi lớp bắt đầu trong [from, to).
		ListMemberBookings(ctx context.Context, memberID uint, from, to time.Time) ([]entity.ClassBookingDetail, error)
		// GetBooking trả chi tiết một lượt đặt (không kiểm tra phạm vi).
		GetBooking(ctx context.Context, id uint) (*entity.ClassBookingDetail, error)
		// MarkAttendance điểm danh (ATTENDED / NO_SHOW) cho các lượt đặt của một buổi lớp.
		MarkAttendance(ctx context.Context, scheduleID uint, items []BookingAttendance) (int, error)
		// MarkNoShows (cron): lượt BOOKED của lớp đã kết thúc > 30 phút → NO_SHOW, khóa đặt lớp hội viên
		// vắng quá booking.no_show_limit lần/30 ngày, lớp đã qua → COMPLETED. Trả số lượt chuyển NO_SHOW.
		MarkNoShows(ctx context.Context, now time.Time) (int, error)
	}

	bookingService struct {
		scheduleRepo repository.ClassScheduleRepository
		bookingRepo  repository.ClassBookingRepository
		tx           postgres.TxManager
		settings     SettingService
		notifier     Notifier
		now          func() time.Time
		loc          *time.Location
	}
)

func NewBookingService(
	scheduleRepo repository.ClassScheduleRepository,
	bookingRepo repository.ClassBookingRepository,
	tx postgres.TxManager,
	settings SettingService,
	notifier Notifier,
) BookingService {
	return &bookingService{
		scheduleRepo: scheduleRepo,
		bookingRepo:  bookingRepo,
		tx:           tx,
		settings:     settings,
		notifier:     notifier,
		now:          time.Now,
		loc:          classLocation(),
	}
}

// bookingCheckScheduleGuard: không nhận đặt chỗ khi buổi lớp rơi vào ngày lễ đóng cửa / ngoài giờ đặc biệt, hoặc HLV
// thực dạy (dạy thay nếu có) đang nghỉ phép đã duyệt (chờ quản lý phân công dạy thay hoặc hủy buổi).
func bookingCheckScheduleGuard(ctx context.Context, sch *entity.ClassSchedule) error {
	guard := currentScheduleGuard()
	h, err := guard.HolidayOn(ctx, sch.BranchID, sch.StartAt)
	if err != nil {
		return err
	}
	if msg := holidaySlotViolation(h, sch.StartAt, sch.EndAt); msg != "" {
		return rescode.New(rescode.BranchHolidayClosed, msg+", không thể đặt chỗ")
	}
	teacher := sch.TrainerID
	if sch.SubstituteTrainerID != nil {
		teacher = *sch.SubstituteTrainerID
	}
	if err := guard.CheckTrainerLeave(ctx, teacher, sch.StartAt, sch.EndAt); err != nil {
		if gerror.Code(err) == rescode.TrainerOnLeave {
			return rescode.New(rescode.TrainerOnLeave, "HLV của buổi lớp đang nghỉ phép, chưa thể đặt chỗ (chờ phân công HLV dạy thay)")
		}
		return err
	}
	return nil
}

func (s *bookingService) Book(ctx context.Context, scheduleID, memberID uint, source string) (*entity.ClassBooking, error) {
	if source == "" {
		source = entity.BookingSourceCounter
	}
	if source != entity.BookingSourceCounter && source != entity.BookingSourcePortal {
		return nil, rescode.New(rescode.BadRequest, "Nguồn đặt chỗ không hợp lệ")
	}
	now := s.now()
	advanceDays := s.settings.Int(ctx, SettingBookingAdvanceDays, bookingDefaultAdvanceDays)

	var out *entity.ClassBooking
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		sch, err := s.lockSchedule(ctx, scheduleID)
		if err != nil {
			return err
		}
		if source == entity.BookingSourceCounter && !reqctx.CanAccessBranch(ctx, sch.BranchID) {
			return rescode.New(rescode.NotFound, "Không tìm thấy buổi lớp")
		}
		if sch.Status != entity.ClassScheduleScheduled {
			return rescode.New(rescode.BadRequest, "Buổi lớp đã bị hủy hoặc đã kết thúc")
		}
		if !sch.StartAt.After(now) {
			return rescode.New(rescode.BadRequest, "Buổi lớp đã bắt đầu, không thể đặt chỗ")
		}
		classDay := sch.StartAt.In(s.loc).Format(consts.DateLayout)
		lastDay := now.In(s.loc).AddDate(0, 0, advanceDays).Format(consts.DateLayout)
		if classDay > lastDay {
			return rescode.New(rescode.BadRequest, fmt.Sprintf("Chỉ được đặt lớp trước tối đa %d ngày", advanceDays))
		}
		if err := bookingCheckScheduleGuard(ctx, sch); err != nil {
			return err
		}

		m, err := s.bookingRepo.FindMember(ctx, memberID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return rescode.New(rescode.NotFound, "Không tìm thấy hội viên")
			}
			return err
		}
		if m.Status != entity.MemberStatusActive {
			return rescode.New(rescode.BadRequest, "Hội viên không ở trạng thái hoạt động")
		}
		if m.BookingBlockedUntil != nil && m.BookingBlockedUntil.After(now) {
			return rescode.New(rescode.BadRequest, fmt.Sprintf(
				"Hội viên bị khóa đặt lớp đến %s do vắng mặt nhiều lần", m.BookingBlockedUntil.In(s.loc).Format("15:04 02/01/2006")))
		}

		subs, err := s.bookingRepo.ActiveSubscriptions(ctx, memberID)
		if err != nil {
			return err
		}
		sub, err := bookingPickSubscription(subs, sch.BranchID, classDay)
		if err != nil {
			return err
		}

		exists, err := s.bookingRepo.ExistsActive(ctx, sch.ID, memberID)
		if err != nil {
			return err
		}
		if exists {
			return rescode.New(rescode.Conflict, bookingDuplicateMsg)
		}
		overlap, err := s.bookingRepo.MemberOverlap(ctx, memberID, sch.StartAt, sch.EndAt, sch.ID)
		if err != nil {
			return err
		}
		if overlap {
			return rescode.New(rescode.Conflict, "Hội viên đã đặt một lớp khác trùng giờ")
		}

		subID := sub.ID
		b := &entity.ClassBooking{
			ScheduleID:     sch.ID,
			MemberID:       memberID,
			SubscriptionID: &subID,
			BookedAt:       now,
			Source:         source,
			CreatedBy:      reqctx.UserID(ctx),
		}
		if sch.BookedCount < sch.Capacity {
			b.Status = entity.BookingStatusBooked
		} else {
			pos, err := s.bookingRepo.MaxWaitlistPosition(ctx, sch.ID)
			if err != nil {
				return err
			}
			pos++
			b.Status = entity.BookingStatusWaitlist
			b.WaitlistPosition = &pos
		}
		if err := s.bookingRepo.Create(ctx, b); err != nil {
			return mapDuplicate(err, bookingDuplicateMsg)
		}
		if b.Status == entity.BookingStatusBooked {
			if err := s.scheduleRepo.AddBooked(ctx, sch.ID, 1); err != nil {
				return err
			}
		}
		out = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *bookingService) Cancel(ctx context.Context, bookingID uint, byMember bool) error {
	now := s.now()
	cancelBefore := s.settings.Int(ctx, SettingBookingCancelBeforeMinutes, bookingDefaultCancelMinutes)

	var promoted []uint
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		b, err := s.findBooking(ctx, bookingID)
		if err != nil {
			return err
		}
		// Khóa dòng lịch trước (cùng thứ tự khóa với Book) rồi đọc lại lượt đặt.
		sch, err := s.lockSchedule(ctx, b.ScheduleID)
		if err != nil {
			return err
		}
		if !byMember && !reqctx.CanAccessBranch(ctx, sch.BranchID) {
			return rescode.New(rescode.NotFound, "Không tìm thấy lượt đặt chỗ")
		}
		if b, err = s.findBooking(ctx, bookingID); err != nil {
			return err
		}
		if b.Status != entity.BookingStatusBooked && b.Status != entity.BookingStatusWaitlist {
			return rescode.New(rescode.BadRequest, "Lượt đặt chỗ không còn hiệu lực để hủy")
		}
		if byMember {
			if !sch.StartAt.After(now) {
				return rescode.New(rescode.BadRequest, "Buổi lớp đã bắt đầu, không thể hủy")
			}
			if now.After(sch.StartAt.Add(-time.Duration(cancelBefore) * time.Minute)) {
				return rescode.New(rescode.BadRequest, fmt.Sprintf("Chỉ được hủy đặt lớp trước giờ học %d phút", cancelBefore))
			}
		}

		wasBooked := b.Status == entity.BookingStatusBooked
		b.Status = entity.BookingStatusCancelled
		b.CancelledAt = &now
		b.WaitlistPosition = nil
		if err := s.bookingRepo.UpdateStatus(ctx, b); err != nil {
			return err
		}
		if !wasBooked {
			return nil
		}
		if err := s.scheduleRepo.AddBooked(ctx, sch.ID, -1); err != nil {
			return err
		}
		if sch.Status == entity.ClassScheduleScheduled && sch.StartAt.After(now) {
			promoted, err = bookingPromoteWaitlist(ctx, s.scheduleRepo, s.bookingRepo, sch.ID, 1)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	bookingNotifyPromoted(ctx, s.notifier, s.bookingRepo, promoted, s.loc)
	return nil
}

func (s *bookingService) ListMemberBookings(ctx context.Context, memberID uint, from, to time.Time) ([]entity.ClassBookingDetail, error) {
	if !to.After(from) {
		return nil, rescode.New(rescode.BadRequest, "Khoảng thời gian không hợp lệ")
	}
	return s.bookingRepo.ListByMember(ctx, memberID, from, to)
}

func (s *bookingService) GetBooking(ctx context.Context, id uint) (*entity.ClassBookingDetail, error) {
	d, err := s.bookingRepo.FindDetail(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy lượt đặt chỗ")
		}
		return nil, err
	}
	return d, nil
}

func (s *bookingService) MarkAttendance(ctx context.Context, scheduleID uint, items []BookingAttendance) (int, error) {
	if len(items) == 0 {
		return 0, rescode.New(rescode.BadRequest, "Vui lòng chọn hội viên cần điểm danh")
	}
	now := s.now()
	updated := 0
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var noShowMembers []uint
		sch, err := s.lockSchedule(ctx, scheduleID)
		if err != nil {
			return err
		}
		if !reqctx.CanAccessBranch(ctx, sch.BranchID) {
			return rescode.New(rescode.NotFound, "Không tìm thấy buổi lớp")
		}
		if sch.Status == entity.ClassScheduleCancelled {
			return rescode.New(rescode.BadRequest, "Buổi lớp đã bị hủy")
		}
		if now.Before(sch.StartAt.Add(-bookingAttendanceOpenBefore)) {
			return rescode.New(rescode.BadRequest, "Chưa đến giờ điểm danh (mở từ 30 phút trước giờ học)")
		}
		if now.After(sch.EndAt.Add(bookingAttendanceCloseAfter)) {
			return rescode.New(rescode.BadRequest, "Đã quá thời hạn điểm danh (24 giờ sau khi lớp kết thúc)")
		}
		for _, it := range items {
			if it.Status != entity.BookingStatusAttended && it.Status != entity.BookingStatusNoShow {
				return rescode.New(rescode.BadRequest, "Trạng thái điểm danh phải là ATTENDED hoặc NO_SHOW")
			}
			b, err := s.bookingRepo.FindByID(ctx, it.BookingID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return rescode.New(rescode.BadRequest, fmt.Sprintf("Lượt đặt chỗ #%d không thuộc buổi lớp này", it.BookingID))
				}
				return err
			}
			if b.ScheduleID != sch.ID {
				return rescode.New(rescode.BadRequest, fmt.Sprintf("Lượt đặt chỗ #%d không thuộc buổi lớp này", it.BookingID))
			}
			switch b.Status {
			case entity.BookingStatusBooked, entity.BookingStatusAttended, entity.BookingStatusNoShow:
			default:
				return rescode.New(rescode.BadRequest,
					fmt.Sprintf("Lượt đặt chỗ #%d đang ở trạng thái %s, không thể điểm danh", it.BookingID, b.Status))
			}
			prevStatus := b.Status
			b.Status = it.Status
			if it.Status == entity.BookingStatusAttended {
				t := now
				b.AttendedAt = &t
			} else {
				b.AttendedAt = nil
			}
			if err := s.bookingRepo.UpdateStatus(ctx, b); err != nil {
				return err
			}
			if it.Status == entity.BookingStatusNoShow && prevStatus != entity.BookingStatusNoShow {
				noShowMembers = append(noShowMembers, b.MemberID)
				// Sửa ATTENDED → NO_SHOW: đánh giá lớp hội viên đã gửi (chỉ được gửi khi đã tham gia) bị gỡ.
				if prevStatus == entity.BookingStatusAttended {
					if rr, ok := s.bookingRepo.(bookingReviewRemover); ok {
						if err := rr.DeleteClassReview(ctx, b.MemberID, sch.ID); err != nil {
							return err
						}
					}
				}
			}
			updated++
		}
		// Vắng mặt điểm danh tay được tính vào hạn mức khóa đặt lớp giống cron MarkNoShows.
		return s.applyNoShowBlocks(ctx, noShowMembers, now)
	})
	if err != nil {
		return 0, err
	}
	return updated, nil
}

func (s *bookingService) MarkNoShows(ctx context.Context, now time.Time) (int, error) {
	marked := 0
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		memberIDs, err := s.bookingRepo.MarkNoShows(ctx, now.Add(-bookingNoShowGrace))
		if err != nil {
			return err
		}
		marked = len(memberIDs)
		if _, err := s.scheduleRepo.CompleteEnded(ctx, now); err != nil {
			return err
		}
		return s.applyNoShowBlocks(ctx, memberIDs, now)
	})
	if err != nil {
		return 0, err
	}
	return marked, nil
}

// bookingReviewRemover: repo gỡ đánh giá lớp của hội viên (repo thật cài; repo giả trong test có thể không).
type bookingReviewRemover interface {
	DeleteClassReview(ctx context.Context, memberID, scheduleID uint) error
}

// applyNoShowBlocks khóa đặt lớp (booking.block_days) các hội viên có số lần NO_SHOW trong 30 ngày ≥ booking.no_show_limit.
// Dùng chung cho cron MarkNoShows và điểm danh tay NO_SHOW. Chạy trong transaction của người gọi.
func (s *bookingService) applyNoShowBlocks(ctx context.Context, memberIDs []uint, now time.Time) error {
	if len(memberIDs) == 0 {
		return nil
	}
	limit := s.settings.Int(ctx, SettingBookingNoShowLimit, bookingDefaultNoShowLimit)
	blockDays := s.settings.Int(ctx, SettingBookingBlockDays, bookingDefaultBlockDays)
	if limit <= 0 {
		return nil
	}
	since := now.AddDate(0, 0, -bookingNoShowWindowDays)
	until := now.AddDate(0, 0, blockDays)
	seen := map[uint]bool{}
	for _, id := range memberIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		n, err := s.bookingRepo.CountNoShows(ctx, id, since)
		if err != nil {
			return err
		}
		if n < int64(limit) {
			continue
		}
		m, err := s.bookingRepo.FindMember(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		if m.BookingBlockedUntil != nil && !m.BookingBlockedUntil.Before(until) {
			continue
		}
		if err := s.bookingRepo.SetMemberBlockedUntil(ctx, id, until); err != nil {
			return err
		}
	}
	return nil
}

// ---- helpers ----------------------------------------------------------------

func (s *bookingService) lockSchedule(ctx context.Context, id uint) (*entity.ClassSchedule, error) {
	sch, err := s.scheduleRepo.FindForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy buổi lớp")
		}
		return nil, err
	}
	return sch, nil
}

func (s *bookingService) findBooking(ctx context.Context, id uint) (*entity.ClassBooking, error) {
	b, err := s.bookingRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy lượt đặt chỗ")
		}
		return nil, err
	}
	return b, nil
}

// bookingPickSubscription chọn đăng ký đủ điều kiện đặt lớp (subs đã lọc ACTIVE, sắp theo end_date):
// gói có lớp nhóm, còn hiệu lực vào ngày lớp, đúng phạm vi chi nhánh, gói SESSION còn buổi.
// Nếu không có, trả lỗi ứng với điều kiện "gần đạt nhất".
func bookingPickSubscription(subs []repository.ClassBookingSubscription, branchID uint, classDay string) (*repository.ClassBookingSubscription, error) {
	msgs := []string{
		"Hội viên không có gói tập đang hoạt động",
		"Gói tập của hội viên không bao gồm lớp nhóm",
		"Gói tập của hội viên không còn hiệu lực vào ngày diễn ra lớp",
		"Gói tập của hội viên không áp dụng tại chi nhánh này",
		"Gói tập theo buổi của hội viên đã hết buổi",
	}
	best := 0
	for i := range subs {
		sub := &subs[i]
		stage := 1
		if sub.Status != "ACTIVE" {
			continue
		}
		if sub.IncludesClasses {
			stage = 2
			if sub.StartDate.Format(consts.DateLayout) <= classDay && classDay <= sub.EndDate.Format(consts.DateLayout) {
				stage = 3
				if sub.AccessScope == entity.AccessAllBranches || sub.BranchID == branchID {
					stage = 4
					if sub.PlanType != entity.PlanTypeSession || (sub.RemainingSessions != nil && *sub.RemainingSessions > 0) {
						return sub, nil
					}
				}
			}
		}
		if stage > best {
			best = stage
		}
	}
	return nil, rescode.New(rescode.BadRequest, msgs[best])
}

// bookingPromoteWaitlist đẩy tối đa n người đầu danh sách chờ lên BOOKED (dòng lịch phải đang bị khóa).
// Trả id các lượt đặt được đẩy lên.
func bookingPromoteWaitlist(ctx context.Context, scheduleRepo repository.ClassScheduleRepository,
	bookingRepo repository.ClassBookingRepository, scheduleID uint, n int) ([]uint, error) {
	var ids []uint
	for i := 0; i < n; i++ {
		w, err := bookingRepo.FirstWaitlist(ctx, scheduleID)
		if err != nil {
			return nil, err
		}
		if w == nil {
			break
		}
		w.Status = entity.BookingStatusBooked
		w.WaitlistPosition = nil
		if err := bookingRepo.UpdateStatus(ctx, w); err != nil {
			return nil, err
		}
		if err := scheduleRepo.AddBooked(ctx, scheduleID, 1); err != nil {
			return nil, err
		}
		ids = append(ids, w.ID)
	}
	return ids, nil
}

// bookingNotifyPromoted gửi thông báo BOOKING_PROMOTED cho hội viên (có tài khoản cổng) vừa được lên BOOKED.
// Lỗi gửi thông báo chỉ được log.
func bookingNotifyPromoted(ctx context.Context, notifier Notifier, bookingRepo repository.ClassBookingRepository,
	bookingIDs []uint, loc *time.Location) {
	for _, id := range bookingIDs {
		d, err := bookingRepo.FindDetail(ctx, id)
		if err != nil {
			g.Log().Warningf(ctx, "booking promoted notify: %v", err)
			continue
		}
		if d.MemberUserID == nil {
			continue
		}
		title := "Bạn đã có chỗ trong lớp " + d.ClassTypeName
		body := fmt.Sprintf("Lớp %s lúc %s tại %s (%s) đã có chỗ cho bạn từ danh sách chờ.",
			d.ClassTypeName, d.StartAt.In(loc).Format("15:04 02/01/2006"), d.RoomName, d.BranchName)
		if err := notifier.Notify(ctx, []uint{*d.MemberUserID}, NotifyBookingPromoted, title, body, bookingNotifyLink); err != nil {
			g.Log().Warningf(ctx, "booking promoted notify: %v", err)
		}
	}
}

// classLocation trả múi giờ Việt Nam dùng cho lịch lớp.
func classLocation() *time.Location {
	loc, err := time.LoadLocation(consts.TimeZone)
	if err != nil {
		return time.FixedZone("ICT", 7*3600)
	}
	return loc
}
