package portal

import (
	"context"

	v1 "gymmaster-api/api/portal/v1"
	"gymmaster-api/internal/service"
)

func (c *ControllerV1) GetProfile(ctx context.Context, _ *v1.GetProfileReq) (res *v1.GetProfileRes, err error) {
	return c.portalService.Profile(ctx)
}

func (c *ControllerV1) UpdateProfile(ctx context.Context, req *v1.UpdateProfileReq) (res *v1.UpdateProfileRes, err error) {
	return c.portalService.UpdateProfile(ctx, service.PortalProfileInput{
		Email:                 req.Email,
		Address:               req.Address,
		EmergencyContactName:  req.EmergencyContactName,
		EmergencyContactPhone: req.EmergencyContactPhone,
		AvatarURL:             req.AvatarURL,
	})
}

func (c *ControllerV1) Upload(ctx context.Context, req *v1.UploadReq) (res *v1.UploadRes, err error) {
	f, err := req.File.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	url, err := c.portalService.UploadAvatar(ctx, req.File.Size, f)
	if err != nil {
		return nil, err
	}
	return &v1.UploadRes{URL: url}, nil
}

func (c *ControllerV1) QR(ctx context.Context, _ *v1.QRReq) (res *v1.QRRes, err error) {
	return c.portalService.QR(ctx)
}
