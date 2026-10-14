package upload

import (
	"gymmaster-api/api/upload"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	uploadService service.UploadService
}

func NewV1(uploadService service.UploadService) upload.IUploadV1 {
	return &ControllerV1{uploadService: uploadService}
}
