package report

import (
	"context"
	"fmt"
	"net/url"

	"github.com/gogf/gf/v2/frame/g"

	v1 "gymmaster-api/api/report/v1"
)

// Export ghi thẳng file .xlsx vào response (middleware Response bỏ qua khi handler đã tự ghi).
func (c *ControllerV1) Export(ctx context.Context, req *v1.ExportReq) (res *v1.ExportRes, err error) {
	data, filename, err := c.reportService.Export(ctx, req)
	if err != nil {
		return nil, err
	}
	r := g.RequestFromCtx(ctx)
	h := r.Response.Header()
	h.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, filename, url.PathEscape(filename)))
	h.Set("Access-Control-Expose-Headers", "Content-Disposition")
	h.Set("Cache-Control", "no-store")
	r.Response.Write(data)
	return nil, nil
}
