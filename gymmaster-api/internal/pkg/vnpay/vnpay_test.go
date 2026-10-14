package vnpay

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gymmaster-api/internal/config"
)

func TestBuildPaymentURLAndVerifyQuery(t *testing.T) {
	cfg := config.VNPayCfg{
		TmnCode:    "GYMMASTR",
		HashSecret: "GYMMASTER_VNPAY_SANDBOX_SECRET_KEY_32",
		PaymentURL: "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html",
		ReturnURL:  "http://localhost:3000/payment/result",
	}
	now := time.Date(2026, 10, 14, 10, 30, 0, 0, time.UTC)

	payURL, err := BuildPaymentURL(cfg, BuildParams{
		TxnRef:     "VNP26101417300001",
		Amount:     1350000,
		OrderInfo:  "Thanh toan hoa don HD26101400001",
		ClientIP:   "127.0.0.1",
		CreateDate: now,
		ExpireDate: now.Add(15 * time.Minute),
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(payURL, cfg.PaymentURL+"?"))

	parsed, err := url.Parse(payURL)
	require.NoError(t, err)
	params := map[string]string{}
	for k, vals := range parsed.Query() {
		if len(vals) > 0 {
			params[k] = vals[0]
		}
	}

	assert.Equal(t, "135000000", params["vnp_Amount"])
	assert.Equal(t, "VNP26101417300001", params["vnp_TxnRef"])
	assert.Equal(t, "GYMMASTR", params["vnp_TmnCode"])
	assert.NotEmpty(t, params["vnp_SecureHash"])

	// Chữ ký hợp lệ (kể cả viết hoa).
	assert.True(t, VerifyQuery(cfg.HashSecret, params))
	paramsUpper := map[string]string{}
	for k, v := range params {
		paramsUpper[k] = v
	}
	paramsUpper["vnp_SecureHash"] = strings.ToUpper(params["vnp_SecureHash"])
	assert.True(t, VerifyQuery(cfg.HashSecret, paramsUpper))

	// Sửa số tiền → chữ ký không còn hợp lệ.
	paramsTampered := map[string]string{}
	for k, v := range params {
		paramsTampered[k] = v
	}
	paramsTampered["vnp_Amount"] = "10000"
	assert.False(t, VerifyQuery(cfg.HashSecret, paramsTampered))

	// Thiếu vnp_SecureHash → false.
	delete(paramsTampered, "vnp_SecureHash")
	assert.False(t, VerifyQuery(cfg.HashSecret, paramsTampered))
}
