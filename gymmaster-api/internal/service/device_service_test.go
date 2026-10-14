package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	checkinv1 "gymmaster-api/api/checkin/v1"
	v1 "gymmaster-api/api/device/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

type fakeDeviceRepo struct {
	nextID          uint
	devs            map[uint]*entity.AccessDevice
	attachedCheckin map[uint]uint
}

func newFakeDeviceRepo() *fakeDeviceRepo {
	return &fakeDeviceRepo{
		devs:            map[uint]*entity.AccessDevice{},
		attachedCheckin: map[uint]uint{},
	}
}

func (f *fakeDeviceRepo) List(_ context.Context, filter repository.AccessDeviceFilter) ([]repository.AccessDeviceRow, error) {
	var out []repository.AccessDeviceRow
	for _, d := range f.devs {
		if filter.Status != "" && d.Status != filter.Status {
			continue
		}
		out = append(out, repository.AccessDeviceRow{AccessDevice: *d, BranchName: "CN Quận 1"})
	}
	return out, nil
}

func (f *fakeDeviceRepo) FindByID(_ context.Context, id uint) (*repository.AccessDeviceRow, error) {
	if d, ok := f.devs[id]; ok {
		return &repository.AccessDeviceRow{AccessDevice: *d, BranchName: "CN Quận 1"}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeDeviceRepo) FindByAPIKeyHash(_ context.Context, keyHash string) (*repository.AccessDeviceRow, error) {
	for _, d := range f.devs {
		if d.APIKeyHash == keyHash {
			return &repository.AccessDeviceRow{AccessDevice: *d, BranchName: "CN Quận 1"}, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeDeviceRepo) ExistsCode(_ context.Context, code string, excludeID uint) (bool, error) {
	for _, d := range f.devs {
		if strings.EqualFold(d.Code, strings.TrimSpace(code)) && d.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeDeviceRepo) Create(_ context.Context, d *entity.AccessDevice) error {
	f.nextID++
	d.ID = f.nextID
	cp := *d
	f.devs[d.ID] = &cp
	return nil
}

func (f *fakeDeviceRepo) Update(_ context.Context, d *entity.AccessDevice) error {
	cp := *d
	f.devs[d.ID] = &cp
	return nil
}

func (f *fakeDeviceRepo) Delete(_ context.Context, id uint, _ *uint) error {
	delete(f.devs, id)
	return nil
}

func (f *fakeDeviceRepo) TouchSeen(_ context.Context, id uint, seenAt time.Time) error {
	if d, ok := f.devs[id]; ok {
		t := seenAt
		d.LastSeenAt = &t
		d.Status = entity.DeviceStatusActive
	}
	return nil
}

func (f *fakeDeviceRepo) AttachDeviceToCheckin(_ context.Context, checkinID, deviceID uint) error {
	f.attachedCheckin[checkinID] = deviceID
	return nil
}

func (f *fakeDeviceRepo) ListStaleActiveDevices(_ context.Context, cutoff time.Time) ([]repository.AccessDeviceRow, error) {
	var out []repository.AccessDeviceRow
	for _, d := range f.devs {
		if d.Status == entity.DeviceStatusActive && d.LastSeenAt != nil && d.LastSeenAt.Before(cutoff) {
			out = append(out, repository.AccessDeviceRow{AccessDevice: *d, BranchName: "CN Quận 1"})
		}
	}
	return out, nil
}

func (f *fakeDeviceRepo) MarkDevicesOffline(_ context.Context, ids []uint) error {
	for _, id := range ids {
		if d, ok := f.devs[id]; ok {
			d.Status = entity.DeviceStatusOffline
		}
	}
	return nil
}

type fakeDeviceCheckinSvc struct {
	lastToken    string
	lastBranchID uint
	result       *checkinv1.ScanResult
}

func (f *fakeDeviceCheckinSvc) IssueQR(context.Context, uint) (string, time.Time, error) {
	return "", time.Time{}, nil
}
func (f *fakeDeviceCheckinSvc) MemberQR(context.Context, uint) (*checkinv1.MemberQRRes, error) {
	return nil, nil
}
func (f *fakeDeviceCheckinSvc) Scan(_ context.Context, tok string, branchID uint) (*checkinv1.ScanResult, error) {
	f.lastToken = tok
	f.lastBranchID = branchID
	return f.result, nil
}
func (f *fakeDeviceCheckinSvc) Manual(context.Context, uint, uint, string) (*checkinv1.ScanResult, error) {
	return nil, nil
}
func (f *fakeDeviceCheckinSvc) Checkout(context.Context, uint) (*checkinv1.CheckinItem, error) {
	return nil, nil
}
func (f *fakeDeviceCheckinSvc) Present(context.Context, uint) (checkinv1.PresentRes, error) {
	return nil, nil
}
func (f *fakeDeviceCheckinSvc) List(context.Context, *checkinv1.ListReq) (*checkinv1.ListRes, error) {
	return nil, nil
}
func (f *fakeDeviceCheckinSvc) Stats(context.Context, uint, string) (*checkinv1.StatsRes, error) {
	return nil, nil
}
func (f *fakeDeviceCheckinSvc) ListMemberCheckins(context.Context, uint, dto.PageReq) (*dto.PageRes[*checkinv1.CheckinItem], error) {
	return nil, nil
}

func TestDeviceService_CreateRotateCheckinAndHeartbeat(t *testing.T) {
	repo := newFakeDeviceRepo()
	checkinFake := &fakeDeviceCheckinSvc{
		result: &checkinv1.ScanResult{
			Result: entity.CheckinAllowed,
			Member: &checkinv1.CheckinMember{
				ID:       10,
				Code:     "HV00010",
				FullName: "Nguyễn Văn A",
			},
			Subscription: &checkinv1.CheckinSubscription{
				ID:       5,
				PlanName: "Gói 12 Tháng VIP",
			},
			Checkin: &checkinv1.CheckinRef{
				ID:         99,
				BranchID:   1,
				BranchName: "CN Quận 1",
			},
		},
	}
	notif := &fakeEquipmentNotifier{}
	svc := NewDeviceService(repo, checkinFake, nil, nil, notif)
	fixedNow := time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC)
	svc.(*deviceService).now = func() time.Time { return fixedNow }

	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})

	// 1. Create device -> returns GMD- + 32 hex chars, stores SHA256
	created, err := svc.Create(ctx, &v1.CreateAccessDeviceReq{
		BranchID: 1,
		Code:     "KIOSK-Q1-01",
		Name:     "Kiosk Sảnh Chính Q1",
		Type:     "KIOSK",
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(created.RawAPIKey, "GMD-"))
	assert.Len(t, created.RawAPIKey, 36)
	assert.Equal(t, hash.SHA256(created.RawAPIKey), repo.devs[created.Device.ID].APIKeyHash)

	// 2. Rotate key -> new key works, old key rejected
	oldKey := created.RawAPIKey
	rotated, err := svc.RotateKey(ctx, created.Device.ID)
	require.NoError(t, err)
	assert.NotEqual(t, oldKey, rotated.RawAPIKey)

	_, err = svc.DeviceHeartbeat(context.Background(), &v1.DeviceHeartbeatReq{DeviceKey: oldKey})
	assert.Equal(t, rescode.DeviceUnauthorized.Code(), codeOf(err))

	// 3. Check-in with rotated key -> delegates to checkinSvc.Scan and updates LastSeenAt
	chkRes, err := svc.DeviceCheckin(context.Background(), &v1.DeviceCheckinReq{
		DeviceKey: rotated.RawAPIKey,
		QRToken:   "QR-TOKEN-123",
	})
	require.NoError(t, err)
	assert.True(t, chkRes.Allow)
	assert.Equal(t, entity.CheckinAllowed, chkRes.Result)
	assert.Equal(t, "Nguyễn Văn A", chkRes.MemberName)
	assert.Equal(t, "Gói 12 Tháng VIP", chkRes.PlanName)
	assert.Equal(t, uint(1), checkinFake.lastBranchID)
	assert.Equal(t, created.Device.ID, repo.attachedCheckin[99])

	// 4. Stale heartbeat check (>10m) marks OFFLINE and sends notification
	n, err := svc.CheckOfflineDevices(ctx, fixedNow.Add(15*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, entity.DeviceStatusOffline, repo.devs[created.Device.ID].Status)
	require.Len(t, notif.permCalls, 1)
	assert.Equal(t, consts.PermDeviceManage, notif.permCalls[0].Perm)
	assert.Equal(t, NotifyDeviceOffline, notif.permCalls[0].Type)

	// 5. Disabled device cannot check in
	_, err = svc.Update(ctx, &v1.UpdateAccessDeviceReq{
		ID:     created.Device.ID,
		Status: entity.DeviceStatusDisabled,
	})
	require.NoError(t, err)
	_, err = svc.DeviceCheckin(context.Background(), &v1.DeviceCheckinReq{
		DeviceKey: rotated.RawAPIKey,
		QRToken:   "QR-TOKEN-123",
	})
	assert.Equal(t, rescode.DeviceUnauthorized.Code(), codeOf(err))
}
