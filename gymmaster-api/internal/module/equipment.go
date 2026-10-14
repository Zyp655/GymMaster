package module

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"

	equipmentctrl "gymmaster-api/internal/controller/equipment"
	"gymmaster-api/internal/cron"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterEquipment: quản lý thiết bị phòng tập, báo hỏng & bảo trì (/v1/admin).
func RegisterEquipment(d *Deps, admin *ghttp.RouterGroup) {
	svc := service.NewEquipmentService(repository.NewEquipmentRepository(d.DB), d.Tx, d.Audit, d.Notifier)
	if admin != nil {
		admin.Bind(equipmentctrl.NewV1(svc))
	}

	if _, exists := cron.DefaultRegistry().Get(service.JobRemindMaintenance); !exists {
		cron.Register(cron.Job{
			Name:        service.JobRemindMaintenance,
			Spec:        "0 7 * * *",
			Description: "Nhắc thiết bị đến hạn bảo trì định kỳ cho quản lý chi nhánh",
			Run: func(ctx context.Context) (int, error) {
				return svc.RemindDueMaintenance(ctx, time.Now())
			},
		})
	}
}
