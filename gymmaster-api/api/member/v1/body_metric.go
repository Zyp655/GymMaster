package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type BodyMetric struct {
	ID             uint      `json:"id"`
	MemberID       uint      `json:"member_id"`
	MeasuredAt     time.Time `json:"measured_at"`
	WeightKg       float64   `json:"weight_kg"`
	HeightCm       float64   `json:"height_cm"`
	BodyFatPct     *float64  `json:"body_fat_pct"`
	MuscleMassKg   *float64  `json:"muscle_mass_kg"`
	WaistCm        *float64  `json:"waist_cm"`
	ChestCm        *float64  `json:"chest_cm"`
	HipCm          *float64  `json:"hip_cm"`
	BMI            *float64  `json:"bmi" dc:"Tự tính trong DB = cân nặng / chiều cao²"`
	Note           string    `json:"note"`
	MeasuredBy     *uint     `json:"measured_by"`
	MeasuredByName string    `json:"measured_by_name"`
	CreatedAt      time.Time `json:"created_at"`
}

type ListBodyMetricsReq struct {
	g.Meta `path:"/members/{id}/body-metrics" method:"get" tags:"Hội viên" summary:"Lịch sử chỉ số cơ thể (tăng dần theo thời gian đo)" perm:"member.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type ListBodyMetricsRes []*BodyMetric

type CreateBodyMetricReq struct {
	g.Meta       `path:"/members/{id}/body-metrics" method:"post" tags:"Hội viên" summary:"Ghi chỉ số cơ thể" perm:"member.metric"`
	ID           uint       `json:"id"             in:"path" v:"required|min:1"`
	MeasuredAt   *time.Time `json:"measured_at"    dc:"RFC3339, mặc định = hiện tại; không được ở tương lai"`
	WeightKg     float64    `json:"weight_kg"      v:"required|between:1,500#Vui lòng nhập cân nặng|Cân nặng phải từ 1 đến 500 kg"`
	HeightCm     float64    `json:"height_cm"      v:"required|between:50,260#Vui lòng nhập chiều cao|Chiều cao phải từ 50 đến 260 cm"`
	BodyFatPct   *float64   `json:"body_fat_pct"   v:"between:0,100#Tỷ lệ mỡ phải từ 0 đến 100%"`
	MuscleMassKg *float64   `json:"muscle_mass_kg" v:"between:0,300#Khối lượng cơ không hợp lệ"`
	WaistCm      *float64   `json:"waist_cm"       v:"between:0,999#Vòng eo không hợp lệ"`
	ChestCm      *float64   `json:"chest_cm"       v:"between:0,999#Vòng ngực không hợp lệ"`
	HipCm        *float64   `json:"hip_cm"         v:"between:0,999#Vòng mông không hợp lệ"`
	Note         string     `json:"note"           v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}
type CreateBodyMetricRes = BodyMetric

type DeleteBodyMetricReq struct {
	g.Meta   `path:"/members/{id}/body-metrics/{metricId}" method:"delete" tags:"Hội viên" summary:"Xóa một lần đo" perm:"member.metric"`
	ID       uint `json:"id"       in:"path" v:"required|min:1"`
	MetricID uint `json:"metricId" in:"path" v:"required|min:1"`
}
type DeleteBodyMetricRes struct{}
