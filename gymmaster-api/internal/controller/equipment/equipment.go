package equipment

import (
	"context"

	v1 "gymmaster-api/api/equipment/v1"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	svc service.EquipmentService
}

func NewV1(svc service.EquipmentService) *ControllerV1 {
	return &ControllerV1{svc: svc}
}

func (c *ControllerV1) ListEquipments(ctx context.Context, req *v1.ListEquipmentsReq) (res *v1.ListEquipmentsRes, err error) {
	return c.svc.ListEquipments(ctx, req)
}

func (c *ControllerV1) GetEquipment(ctx context.Context, req *v1.GetEquipmentReq) (res *v1.GetEquipmentRes, err error) {
	return c.svc.GetEquipment(ctx, req.ID)
}

func (c *ControllerV1) CreateEquipment(ctx context.Context, req *v1.CreateEquipmentReq) (res *v1.CreateEquipmentRes, err error) {
	return c.svc.CreateEquipment(ctx, req.EquipmentInput)
}

func (c *ControllerV1) UpdateEquipment(ctx context.Context, req *v1.UpdateEquipmentReq) (res *v1.UpdateEquipmentRes, err error) {
	return c.svc.UpdateEquipment(ctx, req.ID, req.EquipmentInput)
}

func (c *ControllerV1) DeleteEquipment(ctx context.Context, req *v1.DeleteEquipmentReq) (res *v1.DeleteEquipmentRes, err error) {
	return &v1.DeleteEquipmentRes{}, c.svc.DeleteEquipment(ctx, req.ID)
}

func (c *ControllerV1) ReportIssue(ctx context.Context, req *v1.ReportIssueReq) (res *v1.ReportIssueRes, err error) {
	return c.svc.ReportIssue(ctx, req.ID, req.Description)
}

func (c *ControllerV1) ListMaintenanceLogs(ctx context.Context, req *v1.ListMaintenanceLogsReq) (res *v1.ListMaintenanceLogsRes, err error) {
	return c.svc.ListMaintenanceLogs(ctx, req)
}

func (c *ControllerV1) CreateMaintenanceLog(ctx context.Context, req *v1.CreateMaintenanceLogReq) (res *v1.CreateMaintenanceLogRes, err error) {
	return c.svc.CreateMaintenanceLog(ctx, req.MaintenanceLogInput)
}

func (c *ControllerV1) UpdateMaintenanceLog(ctx context.Context, req *v1.UpdateMaintenanceLogReq) (res *v1.UpdateMaintenanceLogRes, err error) {
	return c.svc.UpdateMaintenanceLog(ctx, req.ID, req.MaintenanceLogInput)
}

func (c *ControllerV1) DeleteMaintenanceLog(ctx context.Context, req *v1.DeleteMaintenanceLogReq) (res *v1.DeleteMaintenanceLogRes, err error) {
	return &v1.DeleteMaintenanceLogRes{}, c.svc.DeleteMaintenanceLog(ctx, req.ID)
}

func (c *ControllerV1) MaintenanceSummary(ctx context.Context, req *v1.MaintenanceSummaryReq) (res *v1.MaintenanceSummaryRes, err error) {
	return c.svc.MaintenanceSummary(ctx, req)
}
