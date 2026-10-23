package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/systemext/v1"
	"gymmaster-api/internal/pkg/rescode"
)

func o2IntP(v int) *int { return &v }

// A5-01: plan_code mà thiếu ngày → lỗi dòng; A5-02: thông tin PT thiếu/sai → lỗi dòng (không bỏ qua im lặng).
func TestValidateLegacyImportRows_PlanCodeAndPT(t *testing.T) {
	rows := []v1.LegacyImportRow{
		{FullName: "A", Phone: "0912000001", PlanCode: "SAI-MA"},
		{FullName: "B", Phone: "0912000002", PTPackageCode: "PT10"},
		{FullName: "C", Phone: "0912000003", PTPackageCode: "PT10", PTRemainingSessions: o2IntP(0), PTTrainerEmail: "hlv@gm.vn"},
		{FullName: "D", Phone: "0912000004", PTPackageCode: "PT10", PTRemainingSessions: o2IntP(3)},
		{FullName: "E", Phone: "0912000005", PTPackageCode: "PT10", PTRemainingSessions: o2IntP(3), PTTrainerEmail: "hlv@gm.vn"},
		{FullName: "F", Phone: "0912000006"},
	}
	ok, errs := ValidateLegacyImportRows(rows)
	assert.Equal(t, 2, ok)
	require.Len(t, errs, 4)
	got := map[int]string{}
	for _, e := range errs {
		got[e.Row] = e.Message
	}
	assert.Contains(t, got[1], "thiếu ngày")
	assert.Contains(t, got[2], "mã gói PT và số buổi")
	assert.Contains(t, got[3], "lớn hơn 0")
	assert.Contains(t, got[4], "email HLV")
}

// REG-A9a: thông báo thao tác hàng loạt bằng tiếng Việt.
func TestBulkMemberActionLabel(t *testing.T) {
	for _, a := range []string{"NOTIFY", "TRANSFER_BRANCH", "ACTIVATE", "DEACTIVATE"} {
		assert.NotEqual(t, a, labelOr(bulkMemberActionLabel, a))
	}
	assert.Equal(t, "kích hoạt lại", bulkMemberActionLabel["ACTIVATE"])
}

// FIN-VNP-19: hai lần hoàn trong cùng giây có gateway_ref khác nhau.
func TestNewVNPayRefundRef_Unique(t *testing.T) {
	now := time.Date(2026, 10, 22, 11, 20, 44, 0, time.UTC)
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		ref := newVNPayRefundRef(47, now)
		assert.LessOrEqual(t, len(ref), 100)
		assert.False(t, seen[ref], ref)
		seen[ref] = true
	}
}

// UIA-073: tiền trong Ctrl+K có phân cách nghìn kiểu VN.
func TestSearchMoneyFormat(t *testing.T) {
	assert.Equal(t, "1.950.000 đ", formatMoneyVND(1950000))
}

// SW-02: wizard đổi giờ → phút cho booking.cancel_before_minutes, giới hạn 168 giờ.
func TestWizardCancelBeforeMinutes(t *testing.T) {
	m, err := wizardCancelBeforeMinutes(2)
	require.NoError(t, err)
	assert.Equal(t, 120, m)
	m, err = wizardCancelBeforeMinutes(0)
	require.NoError(t, err)
	assert.Equal(t, 0, m)
	_, err = wizardCancelBeforeMinutes(169)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	_, err = wizardCancelBeforeMinutes(-1)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
}
