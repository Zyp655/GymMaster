package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/equipment/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	JobRemindMaintenance = "remind_maintenance"

	auditEntityEquipment      = "equipment"
	auditEntityMaintenanceLog = "maintenance_log"
)

var (
	validEquipmentCategories = []string{
		entity.EquipmentCategoryCardio, entity.EquipmentCategoryStrength, entity.EquipmentCategoryFreeWeight,
		entity.EquipmentCategoryFunctional, entity.EquipmentCategoryAccessory, entity.EquipmentCategoryOther,
	}
	validEquipmentStatuses = []string{
		entity.EquipmentStatusActive, entity.EquipmentStatusBroken, entity.EquipmentStatusMaintenance, entity.EquipmentStatusRetired,
	}
	validMaintenanceTypes = []string{
		entity.MaintenanceTypePeriodic, entity.MaintenanceTypeRepair,
	}
	validMaintenanceStatuses = []string{
		entity.MaintenanceStatusOpen, entity.MaintenanceStatusInProgress, entity.MaintenanceStatusDone,
	}
	equipmentOrderable      = []string{"id", "code", "name", "category", "status", "next_maintenance_date", "created_at"}
	maintenanceLogOrderable = []string{"id", "type", "status", "cost", "created_at"}
)

type (
	EquipmentService interface {
		ListEquipments(ctx context.Context, req *v1.ListEquipmentsReq) (*v1.ListEquipmentsRes, error)
		GetEquipment(ctx context.Context, id uint) (*v1.Equipment, error)
		CreateEquipment(ctx context.Context, in v1.EquipmentInput) (*v1.Equipment, error)
		UpdateEquipment(ctx context.Context, id uint, in v1.EquipmentInput) (*v1.Equipment, error)
		DeleteEquipment(ctx context.Context, id uint) error
		ReportIssue(ctx context.Context, id uint, description string) (*v1.MaintenanceLog, error)

		ListMaintenanceLogs(ctx context.Context, req *v1.ListMaintenanceLogsReq) (*v1.ListMaintenanceLogsRes, error)
		CreateMaintenanceLog(ctx context.Context, in v1.MaintenanceLogInput) (*v1.MaintenanceLog, error)
		UpdateMaintenanceLog(ctx context.Context, id uint, in v1.MaintenanceLogInput) (*v1.MaintenanceLog, error)
		DeleteMaintenanceLog(ctx context.Context, id uint) error
		MaintenanceSummary(ctx context.Context, req *v1.MaintenanceSummaryReq) (*v1.MaintenanceSummaryRes, error)

		RemindDueMaintenance(ctx context.Context, now time.Time) (int, error)
	}

	equipmentService struct {
		repo     repository.EquipmentRepository
		tx       postgres.TxManager
		audit    AuditService
		notifier Notifier
		now      func() time.Time
	}
)

func NewEquipmentService(repo repository.EquipmentRepository, tx postgres.TxManager, audit AuditService, notifier Notifier) EquipmentService {
	return &equipmentService{repo: repo, tx: tx, audit: audit, notifier: notifier, now: time.Now}
}

func equipmentStartOfDay(t time.Time) time.Time {
	t = t.In(memberVNLoc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func equipmentParseDateOpt(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(consts.DateLayout, s)
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Ngày phải có định dạng YYYY-MM-DD")
	}
	return &t, nil
}

func equipmentParseTimeOpt(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04", s, memberVNLoc); err == nil {
		return &t, nil
	}
	if t, err := time.ParseInLocation(consts.DateLayout, s, memberVNLoc); err == nil {
		return &t, nil
	}
	return nil, rescode.New(rescode.BadRequest, "Thời gian không hợp lệ")
}

// ---- Thiết bị (Equipment) -------------------------------------------------------

func (s *equipmentService) ListEquipments(ctx context.Context, req *v1.ListEquipmentsReq) (*v1.ListEquipmentsRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem thiết bị của chi nhánh này")
	}
	q := req.PageReq.ToQuery(equipmentOrderable, "id")
	q.OrderBy = "eq." + q.OrderBy
	rows, total, err := s.repo.ListEquipments(ctx, q, repository.EquipmentFilter{
		BranchIDs: branchIDs,
		RoomID:    req.RoomID,
		Category:  strings.TrimSpace(req.Category),
		Status:    strings.TrimSpace(req.Status),
		Search:    req.Search,
	})
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, func(r repository.EquipmentRow) *v1.Equipment {
		return toEquipmentRes(r, nil)
	}), total, q), nil
}

func (s *equipmentService) GetEquipment(ctx context.Context, id uint) (*v1.Equipment, error) {
	row, err := s.repo.FindEquipmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thiết bị")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền truy cập thiết bị của chi nhánh này")
	}
	logs, err := s.repo.ListEquipmentLogs(ctx, id)
	if err != nil {
		return nil, err
	}
	return toEquipmentRes(*row, dto.Map(logs, toMaintenanceLogRes)), nil
}

func (s *equipmentService) prepareEquipment(ctx context.Context, eq *entity.Equipment, in v1.EquipmentInput, excludeID uint) error {
	if in.BranchID == 0 {
		return rescode.New(rescode.BadRequest, "Vui lòng chọn chi nhánh")
	}
	if !reqctx.CanAccessBranch(ctx, in.BranchID) {
		return rescode.New(rescode.PermissionDenied, "Bạn không được thao tác thiết bị tại chi nhánh này")
	}
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	if code == "" {
		return rescode.New(rescode.BadRequest, "Vui lòng nhập mã thiết bị")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return rescode.New(rescode.BadRequest, "Vui lòng nhập tên thiết bị")
	}
	dup, err := s.repo.ExistsEquipmentCode(ctx, code, excludeID)
	if err != nil {
		return err
	}
	if dup {
		return rescode.New(rescode.Conflict, "Mã thiết bị đã tồn tại")
	}
	cat := strings.ToUpper(strings.TrimSpace(in.Category))
	if cat == "" {
		cat = entity.EquipmentCategoryCardio
	}
	if !slices.Contains(validEquipmentCategories, cat) {
		return rescode.New(rescode.BadRequest, "Phân loại thiết bị không hợp lệ")
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = entity.EquipmentStatusActive
	}
	if !slices.Contains(validEquipmentStatuses, status) {
		return rescode.New(rescode.BadRequest, "Trạng thái thiết bị không hợp lệ")
	}
	if in.Price < 0 || in.MaintenanceIntervalDays < 0 {
		return rescode.New(rescode.BadRequest, "Giá mua và chu kỳ bảo trì không được âm")
	}
	purchaseDate, err := equipmentParseDateOpt(in.PurchaseDate)
	if err != nil {
		return err
	}
	warrantyUntil, err := equipmentParseDateOpt(in.WarrantyUntil)
	if err != nil {
		return err
	}
	nextMaintDate, err := equipmentParseDateOpt(in.NextMaintenanceDate)
	if err != nil {
		return err
	}
	today := equipmentStartOfDay(s.now())
	if purchaseDate != nil && purchaseDate.After(today) {
		return rescode.New(rescode.BadRequest, "Ngày mua không được ở tương lai")
	}
	if purchaseDate != nil && warrantyUntil != nil && warrantyUntil.Before(*purchaseDate) {
		return rescode.New(rescode.BadRequest, "Ngày hết bảo hành phải sau hoặc bằng ngày mua")
	}
	if in.RoomID != nil && *in.RoomID > 0 {
		if lk, ok := s.repo.(repository.EquipmentRoomLookup); ok {
			roomBranch, err := lk.RoomBranchID(ctx, *in.RoomID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return rescode.New(rescode.BadRequest, "Phòng tập không tồn tại")
				}
				return err
			}
			if roomBranch != in.BranchID {
				return rescode.New(rescode.BadRequest, "Phòng tập không thuộc chi nhánh của thiết bị")
			}
		}
	}
	if nextMaintDate == nil && excludeID == 0 && in.MaintenanceIntervalDays > 0 {
		base := equipmentStartOfDay(s.now())
		if purchaseDate != nil {
			base = *purchaseDate
		}
		d := base.AddDate(0, 0, in.MaintenanceIntervalDays)
		nextMaintDate = &d
	}

	eq.BranchID = in.BranchID
	eq.RoomID = in.RoomID
	eq.Code = code
	eq.Name = name
	eq.Category = cat
	eq.PurchaseDate = purchaseDate
	eq.Price = in.Price
	eq.WarrantyUntil = warrantyUntil
	eq.Status = status
	eq.MaintenanceIntervalDays = in.MaintenanceIntervalDays
	eq.NextMaintenanceDate = nextMaintDate
	eq.ImageURL = strings.TrimSpace(in.ImageURL)
	eq.Note = strings.TrimSpace(in.Note)
	return nil
}

func (s *equipmentService) CreateEquipment(ctx context.Context, in v1.EquipmentInput) (*v1.Equipment, error) {
	if in.MaintenanceIntervalDays == 0 && in.NextMaintenanceDate == "" {
		in.MaintenanceIntervalDays = 90
	}
	uid := reqctx.UserID(ctx)
	eq := &entity.Equipment{BaseEntity: entity.BaseEntity{CreatedBy: uid, UpdatedBy: uid}}
	if err := s.prepareEquipment(ctx, eq, in, 0); err != nil {
		return nil, err
	}
	if err := s.repo.CreateEquipment(ctx, eq); err != nil {
		return nil, mapDuplicate(err, "Mã thiết bị đã tồn tại")
	}
	res := toEquipmentRes(repository.EquipmentRow{Equipment: *eq}, nil)
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityEquipment, eq.ID, nil, res)
	}
	return res, nil
}

func (s *equipmentService) UpdateEquipment(ctx context.Context, id uint, in v1.EquipmentInput) (*v1.Equipment, error) {
	row, err := s.repo.FindEquipmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thiết bị")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được thao tác thiết bị tại chi nhánh này")
	}
	eq := row.Equipment
	before := toEquipmentRes(*row, nil)
	if err := s.prepareEquipment(ctx, &eq, in, id); err != nil {
		return nil, err
	}
	eq.UpdatedBy = reqctx.UserID(ctx)
	if err := s.repo.UpdateEquipment(ctx, &eq); err != nil {
		return nil, mapDuplicate(err, "Mã thiết bị đã tồn tại")
	}
	res := toEquipmentRes(repository.EquipmentRow{Equipment: eq, BranchName: row.BranchName, RoomName: row.RoomName}, nil)
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityEquipment, eq.ID, before, res)
	}
	return res, nil
}

func (s *equipmentService) DeleteEquipment(ctx context.Context, id uint) error {
	row, err := s.repo.FindEquipmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy thiết bị")
		}
		return err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return rescode.New(rescode.PermissionDenied, "Bạn không được xóa thiết bị của chi nhánh này")
	}
	if err := s.repo.DeleteEquipment(ctx, id, reqctx.UserID(ctx)); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityEquipment, id, toEquipmentRes(*row, nil), nil)
	}
	return nil
}

func (s *equipmentService) ReportIssue(ctx context.Context, id uint, description string) (*v1.MaintenanceLog, error) {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập mô tả sự cố")
	}
	row, err := s.repo.FindEquipmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thiết bị")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được thao tác thiết bị tại chi nhánh này")
	}
	uid := reqctx.UserID(ctx)
	log := &entity.MaintenanceLog{
		EquipmentID: id,
		Type:        entity.MaintenanceTypeRepair,
		ReportedBy:  uid,
		Description: desc,
		Status:      entity.MaintenanceStatusOpen,
	}
	eq := row.Equipment
	eq.Status = entity.EquipmentStatusBroken
	eq.UpdatedBy = uid

	run := func(txCtx context.Context) error {
		if err := s.repo.CreateMaintenanceLog(txCtx, log); err != nil {
			return err
		}
		return s.repo.UpdateEquipment(txCtx, &eq)
	}
	if s.tx != nil {
		err = s.tx.WithTx(ctx, run)
	} else {
		err = run(ctx)
	}
	if err != nil {
		return nil, err
	}
	return toMaintenanceLogRes(repository.MaintenanceLogRow{
		MaintenanceLog: *log,
		EquipmentCode:  row.Code,
		EquipmentName:  row.Name,
		BranchID:       row.BranchID,
		BranchName:     row.BranchName,
	}), nil
}

// ---- Bảo trì / sửa chữa (MaintenanceLog) ----------------------------------------

func (s *equipmentService) ListMaintenanceLogs(ctx context.Context, req *v1.ListMaintenanceLogsReq) (*v1.ListMaintenanceLogsRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem bảo trì của chi nhánh này")
	}
	q := req.PageReq.ToQuery(maintenanceLogOrderable, "created_at")
	q.OrderBy = "ml." + q.OrderBy
	rows, total, err := s.repo.ListMaintenanceLogs(ctx, q, repository.MaintenanceLogFilter{
		BranchIDs:   branchIDs,
		EquipmentID: req.EquipmentID,
		Type:        strings.TrimSpace(req.Type),
		Status:      strings.TrimSpace(req.Status),
	})
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, toMaintenanceLogRes), total, q), nil
}

func (s *equipmentService) applyDoneSideEffects(txCtx context.Context, eq *entity.Equipment, logID uint) error {
	openCount, err := s.repo.CountOpenLogsExcluding(txCtx, eq.ID, logID)
	if err != nil {
		return err
	}
	changed := false
	if (eq.Status == entity.EquipmentStatusBroken || eq.Status == entity.EquipmentStatusMaintenance) && openCount == 0 {
		eq.Status = entity.EquipmentStatusActive
		changed = true
	}
	if eq.MaintenanceIntervalDays > 0 {
		today := equipmentStartOfDay(s.now())
		next := today.AddDate(0, 0, eq.MaintenanceIntervalDays)
		eq.NextMaintenanceDate = &next
		changed = true
	}
	if changed {
		eq.UpdatedBy = reqctx.UserID(txCtx)
		return s.repo.UpdateEquipment(txCtx, eq)
	}
	return nil
}

func (s *equipmentService) CreateMaintenanceLog(ctx context.Context, in v1.MaintenanceLogInput) (*v1.MaintenanceLog, error) {
	if in.EquipmentID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn thiết bị")
	}
	eqRow, err := s.repo.FindEquipmentByID(ctx, in.EquipmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thiết bị")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, eqRow.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được thao tác thiết bị tại chi nhánh này")
	}
	typ := strings.ToUpper(strings.TrimSpace(in.Type))
	if typ == "" {
		typ = entity.MaintenanceTypePeriodic
	}
	if !slices.Contains(validMaintenanceTypes, typ) {
		return nil, rescode.New(rescode.BadRequest, "Loại phiếu bảo trì không hợp lệ")
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = entity.MaintenanceStatusOpen
	}
	if !slices.Contains(validMaintenanceStatuses, status) {
		return nil, rescode.New(rescode.BadRequest, "Trạng thái phiếu bảo trì không hợp lệ")
	}
	if in.Cost < 0 {
		return nil, rescode.New(rescode.BadRequest, "Chi phí bảo trì không được âm")
	}
	startedAt, err := equipmentParseTimeOpt(in.StartedAt)
	if err != nil {
		return nil, err
	}
	finishedAt, err := equipmentParseTimeOpt(in.FinishedAt)
	if err != nil {
		return nil, err
	}
	now := s.now()
	if status == entity.MaintenanceStatusDone && finishedAt == nil {
		finishedAt = &now
	}
	// Kiểm tra SAU khi gán mặc định (DONE + bắt đầu ở tương lai + bỏ trống giờ xong → giờ xong = now < giờ bắt đầu).
	if err := equipmentCheckLogTimes(startedAt, finishedAt, status, now); err != nil {
		return nil, err
	}

	log := &entity.MaintenanceLog{
		EquipmentID: in.EquipmentID,
		Type:        typ,
		ReportedBy:  reqctx.UserID(ctx),
		Description: strings.TrimSpace(in.Description),
		Cost:        in.Cost,
		Status:      status,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
	}
	eq := eqRow.Equipment

	run := func(txCtx context.Context) error {
		if err := s.repo.CreateMaintenanceLog(txCtx, log); err != nil {
			return err
		}
		if status == entity.MaintenanceStatusDone {
			return s.applyDoneSideEffects(txCtx, &eq, log.ID)
		}
		return nil
	}
	if s.tx != nil {
		err = s.tx.WithTx(ctx, run)
	} else {
		err = run(ctx)
	}
	if err != nil {
		return nil, err
	}
	return toMaintenanceLogRes(repository.MaintenanceLogRow{
		MaintenanceLog: *log,
		EquipmentCode:  eqRow.Code,
		EquipmentName:  eqRow.Name,
		BranchID:       eqRow.BranchID,
		BranchName:     eqRow.BranchName,
	}), nil
}

func (s *equipmentService) UpdateMaintenanceLog(ctx context.Context, id uint, in v1.MaintenanceLogInput) (*v1.MaintenanceLog, error) {
	row, err := s.repo.FindMaintenanceLogByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy phiếu bảo trì")
		}
		return nil, err
	}
	eqRow, err := s.repo.FindEquipmentByID(ctx, row.EquipmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thiết bị")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, eqRow.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được thao tác thiết bị tại chi nhánh này")
	}

	log := row.MaintenanceLog
	if typ := strings.ToUpper(strings.TrimSpace(in.Type)); typ != "" {
		if !slices.Contains(validMaintenanceTypes, typ) {
			return nil, rescode.New(rescode.BadRequest, "Loại phiếu bảo trì không hợp lệ")
		}
		log.Type = typ
	}
	if status := strings.ToUpper(strings.TrimSpace(in.Status)); status != "" {
		if !slices.Contains(validMaintenanceStatuses, status) {
			return nil, rescode.New(rescode.BadRequest, "Trạng thái phiếu bảo trì không hợp lệ")
		}
		log.Status = status
	}
	if in.Cost < 0 {
		return nil, rescode.New(rescode.BadRequest, "Chi phí bảo trì không được âm")
	}
	log.Cost = in.Cost
	if in.Description != "" {
		log.Description = strings.TrimSpace(in.Description)
	}
	if in.StartedAt != "" {
		startedAt, err := equipmentParseTimeOpt(in.StartedAt)
		if err != nil {
			return nil, err
		}
		log.StartedAt = startedAt
	}
	if in.FinishedAt != "" {
		finishedAt, err := equipmentParseTimeOpt(in.FinishedAt)
		if err != nil {
			return nil, err
		}
		log.FinishedAt = finishedAt
	}
	now := s.now()
	if log.Status == entity.MaintenanceStatusDone && log.FinishedAt == nil {
		log.FinishedAt = &now
	}
	if err := equipmentCheckLogTimes(log.StartedAt, log.FinishedAt, log.Status, now); err != nil {
		return nil, err
	}

	eq := eqRow.Equipment
	run := func(txCtx context.Context) error {
		if err := s.repo.UpdateMaintenanceLog(txCtx, &log); err != nil {
			return err
		}
		if log.Status == entity.MaintenanceStatusDone {
			return s.applyDoneSideEffects(txCtx, &eq, log.ID)
		}
		return nil
	}
	if s.tx != nil {
		err = s.tx.WithTx(ctx, run)
	} else {
		err = run(ctx)
	}
	if err != nil {
		return nil, err
	}
	return toMaintenanceLogRes(repository.MaintenanceLogRow{
		MaintenanceLog: log,
		EquipmentCode:  eqRow.Code,
		EquipmentName:  eqRow.Name,
		BranchID:       eqRow.BranchID,
		BranchName:     eqRow.BranchName,
		ReporterName:   row.ReporterName,
	}), nil
}

func (s *equipmentService) DeleteMaintenanceLog(ctx context.Context, id uint) error {
	row, err := s.repo.FindMaintenanceLogByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy phiếu bảo trì")
		}
		return err
	}
	if row.BranchID > 0 && !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return rescode.New(rescode.PermissionDenied, "Bạn không được xóa phiếu bảo trì của chi nhánh này")
	}
	if err := s.repo.DeleteMaintenanceLog(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityMaintenanceLog, id, toMaintenanceLogRes(*row), nil)
	}
	return nil
}

func (s *equipmentService) MaintenanceSummary(ctx context.Context, req *v1.MaintenanceSummaryReq) (*v1.MaintenanceSummaryRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem thống kê của chi nhánh này")
	}
	from, err := equipmentParseDateOpt(req.From)
	if err != nil {
		return nil, err
	}
	to, err := equipmentParseDateOpt(req.To)
	if err != nil {
		return nil, err
	}
	if to != nil {
		endExclusive := to.AddDate(0, 0, 1)
		to = &endExclusive
	}
	data, err := s.repo.MaintenanceSummary(ctx, branchIDs, from, to)
	if err != nil {
		return nil, err
	}
	return &v1.MaintenanceSummaryRes{
		TotalCost: data.TotalCost,
		TotalLogs: data.TotalLogs,
		ByBranch: dto.Map(data.ByBranch, func(r repository.MaintenanceBranchCostRow) *v1.MaintenanceBranchCost {
			return &v1.MaintenanceBranchCost{
				BranchID:   r.BranchID,
				BranchName: r.BranchName,
				TotalCost:  r.TotalCost,
				TotalLogs:  r.TotalLogs,
			}
		}),
		ByMonth: dto.Map(data.ByMonth, func(r repository.MaintenanceMonthCostRow) *v1.MaintenanceMonthCost {
			return &v1.MaintenanceMonthCost{
				Month:     r.Month,
				TotalCost: r.TotalCost,
				TotalLogs: r.TotalLogs,
			}
		}),
	}, nil
}

// RemindDueMaintenance (07:00): nhắc các thiết bị ACTIVE đã đến hạn bảo trì định kỳ.
func (s *equipmentService) RemindDueMaintenance(ctx context.Context, now time.Time) (int, error) {
	today := equipmentStartOfDay(now)
	due, err := s.repo.ListDueMaintenance(ctx, today)
	if err != nil {
		return 0, err
	}
	if len(due) == 0 || s.notifier == nil {
		return len(due), nil
	}
	byBranch := map[uint][]repository.EquipmentRow{}
	for _, eq := range due {
		byBranch[eq.BranchID] = append(byBranch[eq.BranchID], eq)
	}
	branchIDs := make([]uint, 0, len(byBranch))
	for bid := range byBranch {
		branchIDs = append(branchIDs, bid)
	}
	sort.Slice(branchIDs, func(i, j int) bool { return branchIDs[i] < branchIDs[j] })

	var firstErr error
	for _, bid := range branchIDs {
		list := byBranch[bid]
		names := make([]string, 0, len(list))
		for _, eq := range list {
			dueStr := ""
			if eq.NextMaintenanceDate != nil {
				dueStr = eq.NextMaintenanceDate.Format("02/01/2006")
			}
			names = append(names, fmt.Sprintf("%s - %s (hạn %s)", eq.Code, eq.Name, dueStr))
		}
		branchID := bid
		title := fmt.Sprintf("%d thiết bị đến hạn bảo trì định kỳ", len(list))
		if list[0].BranchName != "" {
			title = fmt.Sprintf("%d thiết bị đến hạn bảo trì – %s", len(list), list[0].BranchName)
		}
		link := fmt.Sprintf("/admin/equipments?branch_id=%d", bid)
		if err := s.notifier.NotifyPermission(ctx, consts.PermEquipmentManage, &branchID, NotifyMaintenanceDue,
			title, strings.Join(names, "\n"), link); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return len(due), firstErr
}

func toEquipmentRes(r repository.EquipmentRow, logs []*v1.MaintenanceLog) *v1.Equipment {
	return &v1.Equipment{
		ID:                      r.ID,
		BranchID:                r.BranchID,
		BranchName:              r.BranchName,
		RoomID:                  r.RoomID,
		RoomName:                r.RoomName,
		Code:                    r.Code,
		Name:                    r.Name,
		Category:                r.Category,
		PurchaseDate:            workoutFmtDateOpt(r.PurchaseDate),
		Price:                   r.Price,
		WarrantyUntil:           workoutFmtDateOpt(r.WarrantyUntil),
		Status:                  r.Status,
		MaintenanceIntervalDays: r.MaintenanceIntervalDays,
		NextMaintenanceDate:     workoutFmtDateOpt(r.NextMaintenanceDate),
		ImageURL:                r.ImageURL,
		Note:                    r.Note,
		MaintenanceLogs:         logs,
		CreatedAt:               r.CreatedAt,
		UpdatedAt:               r.UpdatedAt,
	}
}

func toMaintenanceLogRes(r repository.MaintenanceLogRow) *v1.MaintenanceLog {
	return &v1.MaintenanceLog{
		ID:            r.ID,
		EquipmentID:   r.EquipmentID,
		EquipmentCode: r.EquipmentCode,
		EquipmentName: r.EquipmentName,
		BranchID:      r.BranchID,
		BranchName:    r.BranchName,
		Type:          r.Type,
		ReportedBy:    r.ReportedBy,
		ReporterName:  r.ReporterName,
		Description:   r.Description,
		Cost:          r.Cost,
		Status:        r.Status,
		StartedAt:     r.StartedAt,
		FinishedAt:    r.FinishedAt,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

// equipmentCheckLogTimes: thời gian hoàn thành phiếu bảo trì không được trước thời gian bắt đầu.
// equipmentCheckLogTimes: giờ hoàn thành ≥ giờ bắt đầu; phiếu DONE không được bắt đầu/hoàn thành ở tương lai
// (cho phép lệch đồng hồ 5 phút).
func equipmentCheckLogTimes(startedAt, finishedAt *time.Time, status string, now time.Time) error {
	if startedAt != nil && finishedAt != nil && finishedAt.Before(*startedAt) {
		return rescode.New(rescode.BadRequest, "Thời gian hoàn thành phải sau thời gian bắt đầu bảo trì")
	}
	if status == entity.MaintenanceStatusDone {
		limit := now.Add(5 * time.Minute)
		if (startedAt != nil && startedAt.After(limit)) || (finishedAt != nil && finishedAt.After(limit)) {
			return rescode.New(rescode.BadRequest, "Phiếu đã hoàn thành không được có thời gian bắt đầu/hoàn thành ở tương lai")
		}
	}
	return nil
}
