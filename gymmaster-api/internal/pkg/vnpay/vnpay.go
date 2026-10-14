// Package vnpay xây dựng URL thanh toán và xác thực chữ ký HMAC-SHA512 theo chuẩn VNPay 2.1.0.
package vnpay

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"gymmaster-api/internal/config"
	"gymmaster-api/internal/consts"
)

const (
	Version    = "2.1.0"
	CommandPay = "pay"
	CurrVND    = "VND"
	LocaleVN   = "vn"
	OrderOther = "other"
	TimeLayout = "20060102150405"
)

var vnLoc = func() *time.Location {
	loc, err := time.LoadLocation(consts.TimeZone)
	if err != nil {
		return time.FixedZone("ICT", 7*3600)
	}
	return loc
}()

// BuildParams là đầu vào để tạo URL thanh toán VNPay.
type BuildParams struct {
	TxnRef     string
	Amount     float64
	OrderInfo  string
	ClientIP   string
	CreateDate time.Time
	ExpireDate time.Time
	ReturnURL  string
}

// BuildPaymentURL tạo URL thanh toán VNPay Sandbox đã ký HMAC-SHA512.
func BuildPaymentURL(cfg config.VNPayCfg, in BuildParams) (string, error) {
	if strings.TrimSpace(in.TxnRef) == "" {
		return "", errors.New("vnpay: thiếu mã giao dịch TxnRef")
	}
	if in.Amount <= 0 {
		return "", errors.New("vnpay: số tiền thanh toán phải lớn hơn 0")
	}
	createAt := in.CreateDate
	if createAt.IsZero() {
		createAt = time.Now()
	}
	expireAt := in.ExpireDate
	if expireAt.IsZero() {
		expireAt = createAt.Add(15 * time.Minute)
	}
	ip := strings.TrimSpace(in.ClientIP)
	if ip == "" {
		ip = "127.0.0.1"
	}
	orderInfo := strings.TrimSpace(in.OrderInfo)
	if orderInfo == "" {
		orderInfo = "Thanh toan don hang " + in.TxnRef
	}
	returnURL := strings.TrimSpace(in.ReturnURL)
	if returnURL == "" {
		returnURL = cfg.ReturnURL
	}
	tmnCode := strings.TrimSpace(cfg.TmnCode)
	if tmnCode == "" {
		tmnCode = "GYMMASTR"
	}
	baseURL := strings.TrimSpace(cfg.PaymentURL)
	if baseURL == "" {
		baseURL = "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html"
	}

	vnpAmount := int64(math.Round(in.Amount * 100))
	params := map[string]string{
		"vnp_Version":    Version,
		"vnp_Command":    CommandPay,
		"vnp_TmnCode":    tmnCode,
		"vnp_Amount":     strconv.FormatInt(vnpAmount, 10),
		"vnp_CurrCode":   CurrVND,
		"vnp_TxnRef":     in.TxnRef,
		"vnp_OrderInfo":  orderInfo,
		"vnp_OrderType":  OrderOther,
		"vnp_Locale":     LocaleVN,
		"vnp_ReturnUrl":  returnURL,
		"vnp_IpAddr":     ip,
		"vnp_CreateDate": createAt.In(vnLoc).Format(TimeLayout),
		"vnp_ExpireDate": expireAt.In(vnLoc).Format(TimeLayout),
	}

	query := canonicalQuery(params)
	secureHash := hmacSHA512(cfg.HashSecret, query)
	return fmt.Sprintf("%s?%s&vnp_SecureHash=%s", baseURL, query, secureHash), nil
}

// VerifyQuery kiểm tra chữ ký vnp_SecureHash (HMAC-SHA512, không phân biệt hoa thường).
func VerifyQuery(hashSecret string, params map[string]string) bool {
	if len(params) == 0 || hashSecret == "" {
		return false
	}
	gotHash := strings.TrimSpace(params["vnp_SecureHash"])
	if gotHash == "" {
		return false
	}
	expected := SignParams(hashSecret, params)
	return strings.EqualFold(expected, gotHash)
}

// SignParams tính vnp_SecureHash cho tập tham số (bỏ qua vnp_SecureHash, vnp_SecureHashType và giá trị rỗng).
func SignParams(hashSecret string, params map[string]string) string {
	return hmacSHA512(hashSecret, canonicalQuery(params))
}

func canonicalQuery(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "vnp_SecureHash" || k == "vnp_SecureHashType" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(k))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(params[k]))
	}
	return b.String()
}

func hmacSHA512(secret, data string) string {
	mac := hmac.New(sha512.New, []byte(secret))
	_, _ = mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
