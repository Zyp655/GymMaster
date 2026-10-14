package einvoice

import (
	"context"
	"fmt"
	"time"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
)

type IssuedEInvoice struct {
	InvoiceID  uint      `json:"invoice_id"`
	InvoiceNo  string    `json:"invoice_no"`
	EInvoiceNo string    `json:"einvoice_no"`
	LookupCode string    `json:"lookup_code"`
	PDFURL     string    `json:"pdf_url"`
	Provider   string    `json:"provider"`
	Total      float64   `json:"total"`
	IssuedAt   time.Time `json:"issued_at"`
}

type Provider interface {
	IssueInvoice(ctx context.Context, inv *entity.Invoice) (*IssuedEInvoice, error)
}

type MockProvider struct {
	now func() time.Time
}

func NewMockProvider() *MockProvider {
	return &MockProvider{now: time.Now}
}

func (p *MockProvider) IssueInvoice(_ context.Context, inv *entity.Invoice) (*IssuedEInvoice, error) {
	now := p.now()
	einvNo := fmt.Sprintf("EINV-%d-%06d", now.Year(), inv.ID)
	lookup := hash.SHA256(fmt.Sprintf("%d:%s:%d", inv.ID, inv.InvoiceNo, now.Unix()))
	if len(lookup) > 12 {
		lookup = lookup[:12]
	}
	return &IssuedEInvoice{
		InvoiceID:  inv.ID,
		InvoiceNo:  inv.InvoiceNo,
		EInvoiceNo: einvNo,
		LookupCode: fmt.Sprintf("TC-%s", lookup),
		PDFURL:     fmt.Sprintf("https://einvoice.gymmaster.vn/pdf/%s.pdf", einvNo),
		Provider:   "MOCK_EINVOICE",
		Total:      inv.Total,
		IssuedAt:   now,
	}, nil
}
