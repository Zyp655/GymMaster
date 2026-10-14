package job

import (
	"context"

	v1 "gymmaster-api/api/job/v1"
	"gymmaster-api/internal/pkg/rescode"
)

func (c *ControllerV1) TestEmail(ctx context.Context, req *v1.TestEmailReq) (res *v1.TestEmailRes, err error) {
	if err := c.mail.SendTestEmail(ctx, req.To); err != nil {
		return nil, rescode.Wrap(err, rescode.BadRequest, "Gửi email thất bại: "+err.Error())
	}
	return &v1.TestEmailRes{Sent: true, To: req.To}, nil
}
