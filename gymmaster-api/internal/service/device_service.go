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

	v1 "gymmaster-api/api/device/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	JobDeviceHeartbeatCheck = "device_heartbeat_check"
	NotifyDeviceOffline     = "DEVICE_OFFLINE"
	auditEntityAccessDevice = "access_device"
	deviceOfflineThreshold  = 10 * time.Minute
)

var (
	validDeviceTypes    = []string{entity.DeviceTypeTurnstile, entity.DeviceTypeDoor, entity.DeviceTypeKiosk}
	validDeviceStatuses = []string{entity.DeviceStatusActive, entity.DeviceStatusOffline, entity.DeviceStatusDisabled}
)

type (
	DeviceService interface {
		List(ctx context.Context, req *v1.ListAccessDevicesReq) (v1.ListAccessDevicesRes, error)
		Create(ctx context.Context, req *v1.CreateAccessDeviceReq) (*v1.CreateAccessDeviceRes, error)
		Update(ctx context.Context, req *v1.UpdateAccessDeviceReq) (*v1.AccessDeviceItem, error)
		Delete(ctx context.Context, id uint) error
		RotateKey(ctx context.Context, id uint) (*v1.RotateDeviceKeyRes, error)
		DeviceCheckin(ctx context.Context, req *v1.DeviceCheckinReq) (*v1.DeviceCheckinRes, error)
		DeviceHeartbeat(ctx context.Context, req *v1.DeviceHeartbeatReq) (*v1.DeviceHeartbeatRes, error)
		CheckOfflineDevices(ctx context.Context, now time.Time) (int, error)
	}

	deviceService struct {
		repo       repository.DeviceRepository
		checkinSvc CheckinService
		tx         postgres.TxManager
		audit      AuditService
		notifier   Notifier
		now        func() time.Time
	}
)

func NewDeviceService(
	repo repository.DeviceRepository,
	checkinSvc CheckinService,
	tx postgres.TxManager,
	audit AuditService,
	notifier Notifier,
) DeviceService {
	return &deviceService{
		repo:       repo,
		checkinSvc: checkinSvc,
		tx:         tx,
		audit:      audit,
		notifier:   notifier,
		now:        time.Now,
	}
}

func generateDeviceKey() (rawKey, keyHash string, err error) {
	hexPart, err := hash.RandomToken(16)
	if err != nil {
		return "", "", err
	}
	rawKey = "GMD-" + hexPart
	return rawKey, hash.SHA256(rawKey), nil
}

func (s *deviceService) List(ctx context.Context, req *v1.ListAccessDevicesReq) (v1.ListAccessDevicesRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem thiết bị tại chi nhánh này")
	}
	rows, err := s.repo.List(ctx, repository.AccessDeviceFilter{
		BranchIDs: branchIDs,
		Status:    strings.ToUpper(strings.TrimSpace(req.Status)),
	})
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toAccessDeviceItem), nil
}

func (s *deviceService) Create(ctx context.Context, req *v1.CreateAccessDeviceReq) (*v1.CreateAccessDeviceRes, error) {
	if req.BranchID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn chi nhánh")
	}
	if !reqctx.CanAccessBranch(ctx, req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được tạo thiết bị tại chi nhánh này")
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập mã thiết bị")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên thiết bị")
	}
	devType := strings.ToUpper(strings.TrimSpace(req.Type))
	if devType == "" {
		devType = entity.DeviceTypeKiosk
	}
	if !slices.Contains(validDeviceTypes, devType) {
		return nil, rescode.New(rescode.BadRequest, "Loại thiết bị không hợp lệ")
	}
	dup, err := s.repo.ExistsCode(ctx, code, 0)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, rescode.New(rescode.Conflict, "Mã thiết bị đã tồn tại")
	}
	rawKey, keyHash, err := generateDeviceKey()
	if err != nil {
		return nil, err
	}
	now := s.now()
	dev := &entity.AccessDevice{
		BaseEntity: entity.BaseEntity{CreatedAt: now, UpdatedAt: now},
		BranchID:   req.BranchID,
		Code:       code,
		Name:       name,
		Type:       devType,
		APIKeyHash: keyHash,
		Status:     entity.DeviceStatusActive,
	}
	if err := s.repo.Create(ctx, dev); err != nil {
		return nil, mapDuplicate(err, "Mã thiết bị đã tồn tại")
	}
	item := toAccessDeviceItem(repository.AccessDeviceRow{AccessDevice: *dev})
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityAccessDevice, dev.ID, nil, item)
	}
	return &v1.CreateAccessDeviceRes{
		Device:    item,
		RawAPIKey: rawKey,
	}, nil
}

func (s *deviceService) Update(ctx context.Context, req *v1.UpdateAccessDeviceReq) (*v1.AccessDeviceItem, error) {
	row, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thiết bị")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được thao tác thiết bị tại chi nhánh này")
	}
	dev := row.AccessDevice
	before := toAccessDeviceItem(*row)
	if name := strings.TrimSpace(req.Name); name != "" {
		dev.Name = name
	}
	if devType := strings.ToUpper(strings.TrimSpace(req.Type)); devType != "" {
		if !slices.Contains(validDeviceTypes, devType) {
			return nil, rescode.New(rescode.BadRequest, "Loại thiết bị không hợp lệ")
		}
		dev.Type = devType
	}
	if status := strings.ToUpper(strings.TrimSpace(req.Status)); status != "" {
		if !slices.Contains(validDeviceStatuses, status) {
			return nil, rescode.New(rescode.BadRequest, "Trạng thái thiết bị không hợp lệ")
		}
		dev.Status = status
	}
	dev.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, &dev); err != nil {
		return nil, err
	}
	res := toAccessDeviceItem(repository.AccessDeviceRow{AccessDevice: dev, BranchName: row.BranchName})
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityAccessDevice, dev.ID, before, res)
	}
	return res, nil
}

func (s *deviceService) Delete(ctx context.Context, id uint) error {
	row, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy thiết bị")
		}
		return err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return rescode.New(rescode.PermissionDenied, "Bạn không được xóa thiết bị tại chi nhánh này")
	}
	if err := s.repo.Delete(ctx, id, reqctx.UserID(ctx)); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityAccessDevice, id, toAccessDeviceItem(*row), nil)
	}
	return nil
}

func (s *deviceService) RotateKey(ctx context.Context, id uint) (*v1.RotateDeviceKeyRes, error) {
	row, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thiết bị")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được thao tác thiết bị tại chi nhánh này")
	}
	rawKey, keyHash, err := generateDeviceKey()
	if err != nil {
		return nil, err
	}
	dev := row.AccessDevice
	dev.APIKeyHash = keyHash
	dev.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, &dev); err != nil {
		return nil, err
	}
	item := toAccessDeviceItem(repository.AccessDeviceRow{AccessDevice: dev, BranchName: row.BranchName})
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityAccessDevice, dev.ID, nil, map[string]any{"action": "ROTATE_KEY"})
	}
	return &v1.RotateDeviceKeyRes{
		Device:    item,
		RawAPIKey: rawKey,
	}, nil
}

func (s *deviceService) authenticateDevice(ctx context.Context, rawKey string) (*repository.AccessDeviceRow, error) {
	key := strings.TrimSpace(rawKey)
	if key == "" {
		return nil, rescode.New(rescode.DeviceUnauthorized, "Device key không hợp lệ hoặc thiết bị đã bị vô hiệu hóa")
	}
	dev, err := s.repo.FindByAPIKeyHash(ctx, hash.SHA256(key))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.DeviceUnauthorized, "Device key không hợp lệ hoặc thiết bị đã bị vô hiệu hóa")
		}
		return nil, err
	}
	if dev.Status == entity.DeviceStatusDisabled {
		return nil, rescode.New(rescode.DeviceUnauthorized, "Device key không hợp lệ hoặc thiết bị đã bị vô hiệu hóa")
	}
	return dev, nil
}

func (s *deviceService) DeviceHeartbeat(ctx context.Context, req *v1.DeviceHeartbeatReq) (*v1.DeviceHeartbeatRes, error) {
	dev, err := s.authenticateDevice(ctx, req.DeviceKey)
	if err != nil {
		return nil, err
	}
	now := s.now()
	if err := s.repo.TouchSeen(ctx, dev.ID, now); err != nil {
		return nil, err
	}
	dev.LastSeenAt = &now
	dev.Status = entity.DeviceStatusActive
	return &v1.DeviceHeartbeatRes{
		Device:     toAccessDeviceItem(*dev),
		ServerTime: now,
	}, nil
}

func (s *deviceService) DeviceCheckin(ctx context.Context, req *v1.DeviceCheckinReq) (*v1.DeviceCheckinRes, error) {
	dev, err := s.authenticateDevice(ctx, req.DeviceKey)
	if err != nil {
		return nil, err
	}
	now := s.now()
	if err := s.repo.TouchSeen(ctx, dev.ID, now); err != nil {
		return nil, err
	}
	if s.checkinSvc == nil {
		return nil, rescode.New(rescode.Internal, "Chưa cấu hình dịch vụ check-in")
	}

	// Thiết bị đã xác thực bằng device key: lượt check-in được ghi trực tiếp với method DEVICE + device_id
	// ngay trong transaction quyết định (không cập nhật lại bản ghi sau khi quét, tránh ghi đè nhầm lượt khác).
	scanRes, err := s.checkinSvc.ScanDevice(ctx, req.QRToken, dev.BranchID, dev.ID)
	if err != nil {
		return nil, err
	}

	allow := scanRes.Result == entity.CheckinAllowed
	msg := "Check-in thành công! Mời vào phòng tập."
	if !allow {
		msg = scanRes.DenyMessage
		if msg == "" {
			msg = CheckinDenyMessage(scanRes.DenyReason)
		}
	}
	out := &v1.DeviceCheckinRes{
		Allow:      allow,
		Result:     scanRes.Result,
		DenyReason: scanRes.DenyReason,
		Message:    msg,
		DeviceName: dev.Name,
		BranchName: dev.BranchName,
	}
	if scanRes.Member != nil {
		out.MemberName = scanRes.Member.FullName
		out.MemberCode = scanRes.Member.Code
		out.AvatarURL = scanRes.Member.AvatarURL
	}
	if scanRes.Subscription != nil {
		out.PlanName = scanRes.Subscription.PlanName
	}
	if out.BranchName == "" && scanRes.Checkin != nil {
		out.BranchName = scanRes.Checkin.BranchName
	}
	return out, nil
}

func (s *deviceService) CheckOfflineDevices(ctx context.Context, now time.Time) (int, error) {
	cutoff := now.Add(-deviceOfflineThreshold)
	stale, err := s.repo.ListStaleActiveDevices(ctx, cutoff)
	if err != nil {
		return 0, err
	}
	if len(stale) == 0 {
		return 0, nil
	}
	ids := make([]uint, 0, len(stale))
	byBranch := map[uint][]repository.AccessDeviceRow{}
	for _, d := range stale {
		ids = append(ids, d.ID)
		byBranch[d.BranchID] = append(byBranch[d.BranchID], d)
	}
	if err := s.repo.MarkDevicesOffline(ctx, ids); err != nil {
		return 0, err
	}
	if s.notifier == nil {
		return len(stale), nil
	}
	branchIDs := make([]uint, 0, len(byBranch))
	for bid := range byBranch {
		branchIDs = append(branchIDs, bid)
	}
	sort.Slice(branchIDs, func(i, j int) bool { return branchIDs[i] < branchIDs[j] })

	var firstErr error
	for _, bid := range branchIDs {
		devs := byBranch[bid]
		lines := make([]string, 0, len(devs))
		for _, d := range devs {
			lines = append(lines, fmt.Sprintf("%s - %s", d.Code, d.Name))
		}
		branchID := bid
		title := fmt.Sprintf("%d thiết bị kiểm soát mất kết nối (OFFLINE)", len(devs))
		link := fmt.Sprintf("/admin/access-devices?branch_id=%d", bid)
		if err := s.notifier.NotifyPermission(ctx, consts.PermDeviceManage, &branchID, NotifyDeviceOffline,
			title, strings.Join(lines, "\n"), link); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return len(stale), firstErr
}

func toAccessDeviceItem(r repository.AccessDeviceRow) *v1.AccessDeviceItem {
	return &v1.AccessDeviceItem{
		ID:         r.ID,
		BranchID:   r.BranchID,
		BranchName: r.BranchName,
		Code:       r.Code,
		Name:       r.Name,
		Type:       r.Type,
		Status:     r.Status,
		LastSeenAt: r.LastSeenAt,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}
