package mailer

import (
	"bytes"
	"fmt"
	"html/template"
)

// Tên template email (tiếng Việt, HTML đơn giản, inline CSS để hiển thị tốt trên mọi client).
const (
	TplSubscriptionExpiring = "subscription_expiring"
	TplSubscriptionExpired  = "subscription_expired"
	TplPasswordReset        = "password_reset"
	TplMemberRegisterOTP    = "member_register_otp"
	TplBirthdayGreeting     = "birthday_greeting"
	TplGeneric              = "generic"
)

// ExpiringData: nhắc gói sắp hết hạn.
type ExpiringData struct {
	MemberName string
	PlanName   string
	EndDate    string // dd/mm/yyyy
	DaysLeft   int
	BranchName string
}

// ExpiredData: gói đã hết hạn.
type ExpiredData struct {
	MemberName string
	PlanName   string
	EndDate    string
	BranchName string
}

// PasswordResetData: đặt lại mật khẩu.
type PasswordResetData struct {
	FullName   string
	ResetURL   string
	TTLMinutes int
}

// MemberOTPData: mã OTP xác thực liên kết tài khoản hội viên.
type MemberOTPData struct {
	FullName   string
	OTP        string
	TTLMinutes int
}

// BirthdayData: chúc mừng sinh nhật kèm mã voucher ưu đãi.
type BirthdayData struct {
	MemberName   string
	VoucherCode  string
	DiscountText string
	EndDate      string
}

// GenericData: thông báo chung (tiêu đề + các đoạn văn).
type GenericData struct {
	Title      string
	Greeting   string
	Paragraphs []string
}

const layout = `{{define "layout"}}<!doctype html>
<html lang="vi"><head><meta charset="utf-8"><title>{{block "title" .}}GymMaster{{end}}</title></head>
<body style="margin:0;padding:0;background:#f4f4f5;font-family:Arial,Helvetica,sans-serif;color:#18181b">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f4f4f5;padding:24px 0">
<tr><td align="center">
<table role="presentation" width="560" cellpadding="0" cellspacing="0" style="max-width:560px;width:100%;background:#ffffff;border-radius:8px;overflow:hidden">
<tr><td style="background:#f97316;padding:16px 24px;color:#ffffff;font-size:20px;font-weight:bold">GymMaster</td></tr>
<tr><td style="padding:24px;font-size:15px;line-height:1.6">{{block "body" .}}{{end}}</td></tr>
<tr><td style="padding:16px 24px;background:#fafafa;color:#71717a;font-size:12px">
Email được gửi tự động từ hệ thống GymMaster, vui lòng không trả lời email này.</td></tr>
</table></td></tr></table></body></html>{{end}}`

var templates = map[string]string{
	TplSubscriptionExpiring: `{{define "title"}}Gói tập sắp hết hạn{{end}}{{define "body"}}
<p>Xin chào <strong>{{.MemberName}}</strong>,</p>
<p>Gói tập của bạn – <strong>{{.PlanName}}</strong>{{if .BranchName}} tại <strong>{{.BranchName}}</strong>{{end}} – sẽ hết hạn vào ngày
<strong>{{.EndDate}}</strong>{{if gt .DaysLeft 0}} (còn {{.DaysLeft}} ngày){{else}} (hôm nay){{end}}.</p>
<p>Hãy gia hạn sớm tại quầy lễ tân hoặc trên cổng hội viên để việc tập luyện không bị gián đoạn.</p>
<p>Cảm ơn bạn đã đồng hành cùng GymMaster!</p>{{end}}`,

	TplSubscriptionExpired: `{{define "title"}}Gói tập đã hết hạn{{end}}{{define "body"}}
<p>Xin chào <strong>{{.MemberName}}</strong>,</p>
<p>Gói tập của bạn – <strong>{{.PlanName}}</strong>{{if .BranchName}} tại <strong>{{.BranchName}}</strong>{{end}} – đã hết hạn ngày <strong>{{.EndDate}}</strong>.</p>
<p>Liên hệ lễ tân để được tư vấn gói tập phù hợp và ưu đãi gia hạn.</p>{{end}}`,

	TplPasswordReset: `{{define "title"}}Đặt lại mật khẩu GymMaster{{end}}{{define "body"}}
<p>Xin chào <strong>{{.FullName}}</strong>,</p>
<p>Hệ thống nhận được yêu cầu đặt lại mật khẩu cho tài khoản của bạn. Nhấn vào đường dẫn bên dưới để tạo mật khẩu mới (hiệu lực trong <strong>{{.TTLMinutes}} phút</strong>):</p>
<p style="margin:20px 0"><a href="{{.ResetURL}}" style="display:inline-block;padding:10px 20px;background:#f97316;color:#ffffff;text-decoration:none;border-radius:6px;font-weight:bold">Đặt lại mật khẩu</a></p>
<p style="font-size:13px;color:#52525b">Hoặc sao chép đường dẫn: {{.ResetURL}}</p>
<p>Nếu bạn không yêu cầu đặt lại mật khẩu, vui lòng bỏ qua email này.</p>{{end}}`,

	TplMemberRegisterOTP: `{{define "title"}}Mã xác thực tài khoản hội viên GymMaster{{end}}{{define "body"}}
<p>Xin chào <strong>{{.FullName}}</strong>,</p>
<p>Mã xác thực (OTP) để liên kết hồ sơ hội viên của bạn với tài khoản cổng hội viên là:</p>
<p style="font-size:28px;font-weight:bold;letter-spacing:4px;color:#f97316;margin:16px 0">{{.OTP}}</p>
<p>Mã có hiệu lực trong <strong>{{.TTLMinutes}} phút</strong>. Vui lòng không chia sẻ mã này cho bất kỳ ai.</p>{{end}}`,

	TplBirthdayGreeting: `{{define "title"}}Chúc mừng sinh nhật từ GymMaster!{{end}}{{define "body"}}
<p>Xin chào <strong>{{.MemberName}}</strong>,</p>
<p>GymMaster chúc bạn một ngày sinh nhật thật nhiều sức khỏe, niềm vui và năng lượng tích cực!</p>
{{if .VoucherCode}}<p>Dành tặng riêng bạn mã ưu đãi sinh nhật: <strong style="color:#f97316;font-size:18px">{{.VoucherCode}}</strong> ({{.DiscountText}}), áp dụng đến ngày <strong>{{.EndDate}}</strong>.</p>{{end}}
<p>Cảm ơn bạn đã luôn đồng hành cùng GymMaster!</p>{{end}}`,

	TplGeneric: `{{define "title"}}{{.Title}}{{end}}{{define "body"}}
{{if .Greeting}}<p>{{.Greeting}}</p>{{end}}{{range .Paragraphs}}<p>{{.}}</p>{{end}}{{end}}`,
}

var parsed = func() map[string]*template.Template {
	out := make(map[string]*template.Template, len(templates))
	for name, body := range templates {
		t := template.Must(template.New(name).Parse(layout))
		out[name] = template.Must(t.Parse(body))
	}
	return out
}()

// Render dựng HTML cho template name với data tương ứng (ExpiringData, ExpiredData, GenericData).
func Render(name string, data any) (string, error) {
	t, ok := parsed[name]
	if !ok {
		return "", fmt.Errorf("mailer: không có template %q", name)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
