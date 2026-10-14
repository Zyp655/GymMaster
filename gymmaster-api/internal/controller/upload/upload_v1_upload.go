package upload

import (
	"context"

	v1 "gymmaster-api/api/upload/v1"
)

func (c *ControllerV1) Upload(ctx context.Context, req *v1.UploadReq) (res *v1.UploadRes, err error) {
	f, err := req.File.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	url, err := c.uploadService.SaveImage(ctx, req.File.Size, f)
	if err != nil {
		return nil, err
	}
	return &v1.UploadRes{URL: url}, nil
}
