package class

import (
	"context"
	"time"

	v1 "gymmaster-api/api/class/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/service"
)

func (c *ControllerV1) ListScheduleBookings(ctx context.Context, req *v1.ListScheduleBookingsReq) (res *v1.ListScheduleBookingsRes, err error) {
	items, err := c.scheduleService.Bookings(ctx, req.ID, req.Status)
	if err != nil {
		return nil, err
	}
	out := v1.ListScheduleBookingsRes(items)
	return &out, nil
}

func (c *ControllerV1) CreateBooking(ctx context.Context, req *v1.CreateBookingReq) (res *v1.CreateBookingRes, err error) {
	b, err := c.bookingService.Book(ctx, req.ID, req.MemberID, entity.BookingSourceCounter)
	if err != nil {
		return nil, err
	}
	return c.bookingRes(ctx, b.ID)
}

func (c *ControllerV1) CancelBooking(ctx context.Context, req *v1.CancelBookingReq) (res *v1.CancelBookingRes, err error) {
	if err := c.bookingService.Cancel(ctx, req.ID, false); err != nil {
		return nil, err
	}
	return c.bookingRes(ctx, req.ID)
}

func (c *ControllerV1) ListMemberBookings(ctx context.Context, req *v1.ListMemberBookingsReq) (res *v1.ListMemberBookingsRes, err error) {
	loc, lerr := time.LoadLocation(consts.TimeZone)
	if lerr != nil {
		loc = time.FixedZone("ICT", 7*3600)
	}
	today := time.Now().In(loc)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)
	from, to := today.AddDate(0, 0, -30), today.AddDate(0, 0, 31)
	if req.From != "" {
		if from, err = time.ParseInLocation(consts.DateLayout, req.From, loc); err != nil {
			return nil, rescode.New(rescode.BadRequest, "Ngày phải có dạng YYYY-MM-DD")
		}
	}
	if req.To != "" {
		if to, err = time.ParseInLocation(consts.DateLayout, req.To, loc); err != nil {
			return nil, rescode.New(rescode.BadRequest, "Ngày phải có dạng YYYY-MM-DD")
		}
		to = to.AddDate(0, 0, 1)
	}
	if to.Sub(from) > 366*24*time.Hour {
		return nil, rescode.New(rescode.BadRequest, "Chỉ xem được tối đa 1 năm mỗi lần")
	}
	items, err := c.bookingService.ListMemberBookings(ctx, req.MemberID, from, to)
	if err != nil {
		return nil, err
	}
	out := make(v1.ListMemberBookingsRes, 0, len(items))
	for i := range items {
		// Nhân viên chỉ thấy lượt đặt tại chi nhánh thuộc phạm vi của mình.
		if reqctx.CanAccessBranch(ctx, items[i].BranchID) {
			out = append(out, service.ClassBookingToRes(&items[i]))
		}
	}
	return &out, nil
}

func (c *ControllerV1) Attendance(ctx context.Context, req *v1.AttendanceReq) (res *v1.AttendanceRes, err error) {
	items := make([]service.BookingAttendance, 0, len(req.Items))
	for _, it := range req.Items {
		if it == nil {
			continue
		}
		items = append(items, service.BookingAttendance{BookingID: it.BookingID, Status: it.Status})
	}
	n, err := c.bookingService.MarkAttendance(ctx, req.ID, items)
	if err != nil {
		return nil, err
	}
	return &v1.AttendanceRes{Updated: n}, nil
}

func (c *ControllerV1) bookingRes(ctx context.Context, id uint) (*v1.ClassBooking, error) {
	d, err := c.bookingService.GetBooking(ctx, id)
	if err != nil {
		return nil, err
	}
	return service.ClassBookingToRes(d), nil
}
