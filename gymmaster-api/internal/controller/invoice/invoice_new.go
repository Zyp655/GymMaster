package invoice

import (
	"gymmaster-api/api/invoice"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	svc service.InvoiceService
}

func NewV1(svc service.InvoiceService) invoice.IInvoiceV1 {
	return &ControllerV1{svc: svc}
}
