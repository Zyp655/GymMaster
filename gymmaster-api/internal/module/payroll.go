package module

import (
	"sync"

	"github.com/gogf/gf/v2/net/ghttp"

	payrollctrl "gymmaster-api/internal/controller/payroll"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

var (
	payrollMu       sync.Mutex
	payrollServices = map[*Deps]service.PayrollService{}
)

func PayrollServiceOf(d *Deps) service.PayrollService {
	payrollMu.Lock()
	defer payrollMu.Unlock()
	if s, ok := payrollServices[d]; ok {
		return s
	}
	s := service.NewPayrollService(
		repository.NewPayrollRepository(d.DB),
		d.Tx,
		d.Audit,
	)
	payrollServices[d] = s
	return s
}

// RegisterPayroll đăng ký các route chấm công nhân viên & bảng lương (/v1/admin).
func RegisterPayroll(d *Deps, admin *ghttp.RouterGroup) {
	svc := PayrollServiceOf(d)
	if admin != nil {
		admin.Bind(payrollctrl.NewV1(svc))
	}
}
