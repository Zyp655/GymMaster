package i18n

import (
	"context"
	"strings"
	"testing"

	"github.com/gogf/gf/v2/frame/g"
)

func TestInit_VietnameseValidationMessages(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	type req struct {
		Code        string  `v:"required"`
		OpeningCash float64 `v:"min:0"`
	}
	err := g.Validator().Data(&req{OpeningCash: -5}).Run(context.Background())
	if err == nil {
		t.Fatal("expected validation error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "bắt buộc") {
		t.Fatalf("expected Vietnamese message, got %q", msg)
	}
	err = g.Validator().Data(&req{Code: "X", OpeningCash: -5}).Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "lớn hơn hoặc bằng 0") {
		t.Fatalf("expected Vietnamese min message, got %v", err)
	}
}
