package module

import (
	"context"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/gogf/gf/v2/net/ghttp"

	devicectrl "gymmaster-api/internal/controller/device"
	"gymmaster-api/internal/cron"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

var (
	deviceMu       sync.Mutex
	deviceServices = map[*Deps]service.DeviceService{}
)

func DeviceServiceOf(d *Deps) service.DeviceService {
	deviceMu.Lock()
	defer deviceMu.Unlock()
	if s, ok := deviceServices[d]; ok {
		return s
	}
	checkinSvc := service.NewCheckinService(repository.NewCheckinRepository(d.DB), d.Tx, d.Settings)
	s := service.NewDeviceService(
		repository.NewDeviceRepository(d.DB),
		checkinSvc,
		d.Tx,
		d.Audit,
		d.Notifier,
	)
	deviceServices[d] = s

	if _, exists := cron.DefaultRegistry().Get(service.JobDeviceHeartbeatCheck); !exists {
		cron.Register(cron.Job{
			Name:        service.JobDeviceHeartbeatCheck,
			Spec:        "*/5 * * * *",
			Description: "Đánh dấu Mất kết nối các thiết bị kiểm soát ra vào không gửi tín hiệu quá 10 phút",
			Run: func(ctx context.Context) (int, error) {
				return s.CheckOfflineDevices(ctx, time.Now())
			},
		})
	}
	return s
}

func parentRouterGroup(g *ghttp.RouterGroup) *ghttp.RouterGroup {
	if g == nil {
		return nil
	}
	v := reflect.ValueOf(g)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil
	}
	elem := v.Elem()
	prefixField := elem.FieldByName("prefix")
	if prefixField.IsValid() && prefixField.String() != "/public" {
		return nil
	}
	parentField := elem.FieldByName("parent")
	if !parentField.IsValid() || parentField.IsNil() {
		return nil
	}
	return (*ghttp.RouterGroup)(unsafe.Pointer(parentField.Pointer()))
}

// RegisterDevice đăng ký route quản trị thiết bị (/v1/admin), route thiết bị tại /v1/public/device/*
// và tạo thêm alias trực tiếp tại /v1/device/*.
func RegisterDevice(d *Deps, admin, public *ghttp.RouterGroup, extra ...*ghttp.RouterGroup) {
	svc := DeviceServiceOf(d)
	if admin != nil {
		admin.Bind(devicectrl.NewAdminV1(svc))
	}
	pubCtrl := devicectrl.NewPublicV1(svc)
	if public != nil {
		public.Bind(pubCtrl)
	}
	if len(extra) > 0 && extra[0] != nil {
		extra[0].Bind(pubCtrl)
	} else if v1Group := parentRouterGroup(public); v1Group != nil {
		v1Group.Bind(pubCtrl)
	}
}
