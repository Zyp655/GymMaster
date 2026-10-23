package v1

import (
	"context"
	"strings"
	"testing"

	"github.com/gogf/gf/v2/frame/g"
)

// C8-01: lỗi kích cỡ tủ phải nói về tủ, không dùng nhãn chung "kích thước trang" của tham số phân trang.
func TestCreateLockerReq_SizeMessage(t *testing.T) {
	req := &CreateLockerReq{BranchID: 1, Code: "A01", Size: "HUGE", Status: "AVAILABLE"}
	err := g.Validator().Data(req).Run(context.Background())
	if err == nil {
		t.Fatal("size không hợp lệ phải báo lỗi")
	}
	msg := err.FirstError().Error()
	if !strings.Contains(msg, "Kích cỡ tủ") || strings.Contains(msg, "trang") {
		t.Fatalf("thông báo sai ngữ cảnh: %q", msg)
	}
}
