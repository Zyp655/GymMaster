package member

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	v1 "gymmaster-api/api/member/v1"
)

// ImportTemplate ghi thẳng file .xlsx vào response (middleware Response bỏ qua khi handler đã tự ghi).
func (c *ControllerV1) ImportTemplate(ctx context.Context, req *v1.ImportTemplateReq) (res *v1.ImportTemplateRes, err error) {
	data, err := c.memberService.ImportTemplate()
	if err != nil {
		return nil, err
	}
	r := g.RequestFromCtx(ctx)
	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", `attachment; filename="mau-nhap-hoi-vien.xlsx"`)
	r.Response.Write(data)
	return nil, nil
}
