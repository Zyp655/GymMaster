package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type JobRun struct {
	ID         uint       `json:"id"`
	Job        string     `json:"job"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	DurationMs *int64     `json:"duration_ms"`
	Status     string     `json:"status"` // RUNNING | SUCCESS | FAILED
	Affected   int        `json:"affected"`
	Message    string     `json:"message"`
}

type Job struct {
	Name        string     `json:"name"`
	Spec        string     `json:"spec"`
	Description string     `json:"description"`
	Scheduled   bool       `json:"scheduled"` // scheduler đang bật (CRON_ENABLED)
	Running     bool       `json:"running"`   // đang chạy trong instance này
	NextRunAt   *time.Time `json:"next_run_at"`
	LastRun     *JobRun    `json:"last_run"`
}

type ListReq struct {
	g.Meta `path:"/jobs" method:"get" tags:"Job định kỳ" summary:"Danh sách job + lần chạy gần nhất" perm:"job.view"`
}
type ListRes []*Job

type RunsReq struct {
	g.Meta `path:"/jobs/{name}/runs" method:"get" tags:"Job định kỳ" summary:"Lịch sử chạy của job" perm:"job.view"`
	dto.PageReq
	Name   string `json:"name"   in:"path"  v:"required"`
	Status string `json:"status" in:"query" v:"in:RUNNING,SUCCESS,FAILED#Trạng thái không hợp lệ"`
}
type RunsRes = dto.PageRes[*JobRun]

type RunReq struct {
	g.Meta `path:"/jobs/{name}/run" method:"post" tags:"Job định kỳ" summary:"Chạy job ngay (đồng bộ)" perm:"job.view"`
	Name   string `json:"name" in:"path" v:"required"`
}
type RunRes = JobRun

type TestEmailReq struct {
	g.Meta `path:"/jobs/test-email" method:"post" tags:"Job định kỳ" summary:"Gửi email thử (kiểm tra cấu hình SMTP)" perm:"job.view"`
	To     string `json:"to" v:"required|email#Vui lòng nhập email nhận|Email không hợp lệ"`
}
type TestEmailRes struct {
	Sent bool   `json:"sent"`
	To   string `json:"to"`
}
