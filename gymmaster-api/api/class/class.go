// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package class

import (
	"context"

	v1 "gymmaster-api/api/class/v1"
)

type IClassV1 interface {
	ListType(ctx context.Context, req *v1.ListTypeReq) (res *v1.ListTypeRes, err error)
	TypeOptions(ctx context.Context, req *v1.TypeOptionsReq) (res *v1.TypeOptionsRes, err error)
	GetType(ctx context.Context, req *v1.GetTypeReq) (res *v1.GetTypeRes, err error)
	CreateType(ctx context.Context, req *v1.CreateTypeReq) (res *v1.CreateTypeRes, err error)
	UpdateType(ctx context.Context, req *v1.UpdateTypeReq) (res *v1.UpdateTypeRes, err error)
	DeleteType(ctx context.Context, req *v1.DeleteTypeReq) (res *v1.DeleteTypeRes, err error)

	ListSchedule(ctx context.Context, req *v1.ListScheduleReq) (res *v1.ListScheduleRes, err error)
	GetSchedule(ctx context.Context, req *v1.GetScheduleReq) (res *v1.GetScheduleRes, err error)
	CreateSchedule(ctx context.Context, req *v1.CreateScheduleReq) (res *v1.CreateScheduleRes, err error)
	CreateRecurring(ctx context.Context, req *v1.CreateRecurringReq) (res *v1.CreateRecurringRes, err error)
	UpdateSchedule(ctx context.Context, req *v1.UpdateScheduleReq) (res *v1.UpdateScheduleRes, err error)
	CancelSchedule(ctx context.Context, req *v1.CancelScheduleReq) (res *v1.CancelScheduleRes, err error)

	ListScheduleBookings(ctx context.Context, req *v1.ListScheduleBookingsReq) (res *v1.ListScheduleBookingsRes, err error)
	CreateBooking(ctx context.Context, req *v1.CreateBookingReq) (res *v1.CreateBookingRes, err error)
	CancelBooking(ctx context.Context, req *v1.CancelBookingReq) (res *v1.CancelBookingRes, err error)
	ListMemberBookings(ctx context.Context, req *v1.ListMemberBookingsReq) (res *v1.ListMemberBookingsRes, err error)
	Attendance(ctx context.Context, req *v1.AttendanceReq) (res *v1.AttendanceRes, err error)
}
