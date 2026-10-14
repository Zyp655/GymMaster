package payroll

import (
	"context"

	v1 "gymmaster-api/api/payroll/v1"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	svc service.PayrollService
}

func NewV1(svc service.PayrollService) *ControllerV1 {
	return &ControllerV1{svc: svc}
}

func NewAdminV1(svc service.PayrollService) *ControllerV1 {
	return &ControllerV1{svc: svc}
}

func (c *ControllerV1) ScanAttendance(ctx context.Context, req *v1.ScanAttendanceReq) (res *v1.ScanAttendanceRes, err error) {
	return c.svc.ScanAttendance(ctx, req)
}

func (c *ControllerV1) ListAttendances(ctx context.Context, req *v1.ListAttendancesReq) (res v1.ListAttendancesRes, err error) {
	return c.svc.ListAttendances(ctx, req)
}

func (c *ControllerV1) GeneratePayroll(ctx context.Context, req *v1.GeneratePayrollReq) (res *v1.GeneratePayrollRes, err error) {
	return c.svc.GeneratePayroll(ctx, req)
}

func (c *ControllerV1) ListPayrolls(ctx context.Context, req *v1.ListPayrollsReq) (res v1.ListPayrollsRes, err error) {
	return c.svc.ListPayrolls(ctx, req)
}

func (c *ControllerV1) GetPayroll(ctx context.Context, req *v1.GetPayrollReq) (res *v1.GetPayrollRes, err error) {
	return c.svc.GetPayroll(ctx, req.ID)
}

func (c *ControllerV1) UpdatePayroll(ctx context.Context, req *v1.UpdatePayrollReq) (res *v1.UpdatePayrollRes, err error) {
	return c.svc.UpdatePayroll(ctx, req)
}

func (c *ControllerV1) LockPayroll(ctx context.Context, req *v1.LockPayrollReq) (res *v1.LockPayrollRes, err error) {
	return c.svc.LockPayroll(ctx, req.ID)
}

func (c *ControllerV1) PayPayroll(ctx context.Context, req *v1.PayPayrollReq) (res *v1.PayPayrollRes, err error) {
	return c.svc.PayPayroll(ctx, req.ID)
}

func (c *ControllerV1) ExportPayroll(ctx context.Context, req *v1.ExportPayrollReq) (res *v1.ExportPayrollRes, err error) {
	return c.svc.ExportPayroll(ctx, req.ID)
}

func (c *ControllerV1) MyPayslips(ctx context.Context, _ *v1.MyPayslipsReq) (res v1.MyPayslipsRes, err error) {
	return c.svc.MyPayslips(ctx)
}
