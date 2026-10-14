// Package cron chạy các job định kỳ của hệ thống.
//
// Thiết kế registry: module nào cần job thì tự đăng ký khi lắp ráp (internal/module/<m>.go):
//
//	cron.Register(cron.Job{
//	    Name:        "expire_pt_contracts",           // snake_case, duy nhất, ≤ 50 ký tự (job_runs.job)
//	    Spec:        "15 0 * * *",                    // cú pháp cron 5 trường, giờ Asia/Ho_Chi_Minh
//	    Description: "Chuyển hợp đồng PT quá hạn sang EXPIRED",
//	    Run: func(ctx context.Context) (int, error) { // trả số bản ghi bị ảnh hưởng
//	        return ptService.ExpireContracts(ctx, time.Now())
//	    },
//	})
//
// cmd khởi động scheduler SAU khi router.Register (lúc đó mọi module đã đăng ký job) và chỉ khi CRON_ENABLED=true.
// Mỗi lần chạy (theo lịch hoặc thủ công qua API) đều ghi job_runs; chống chạy chồng bằng mutex của từng job
// trong tiến trình + pg_try_advisory_lock giữa nhiều instance.
package cron

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"sync"
	"time"

	rcron "github.com/robfig/cron/v3"
)

// Job là một tác vụ định kỳ. Run trả số bản ghi bị ảnh hưởng (affected) và lỗi (nếu có – kể cả khi đã xử lý một phần).
type Job struct {
	Name        string
	Spec        string
	Description string
	Run         func(ctx context.Context) (int, error)
	// Timeout cho mỗi lần chạy (mặc định 10 phút).
	Timeout time.Duration
}

var (
	jobNameRe  = regexp.MustCompile(`^[a-z][a-z0-9_]{1,49}$`)
	specParser = rcron.NewParser(rcron.Minute | rcron.Hour | rcron.Dom | rcron.Month | rcron.Dow | rcron.Descriptor)
)

// Registry giữ danh sách job theo thứ tự đăng ký.
type Registry struct {
	mu    sync.RWMutex
	jobs  []Job
	index map[string]int
}

func NewRegistry() *Registry {
	return &Registry{index: map[string]int{}}
}

// Validate kiểm tra tên, cú pháp lịch và hàm chạy.
func (j Job) Validate() error {
	if !jobNameRe.MatchString(j.Name) {
		return fmt.Errorf("cron: tên job %q không hợp lệ (snake_case, 2-50 ký tự)", j.Name)
	}
	if _, err := specParser.Parse(j.Spec); err != nil {
		return fmt.Errorf("cron: lịch %q của job %s không hợp lệ: %w", j.Spec, j.Name, err)
	}
	if j.Run == nil {
		return fmt.Errorf("cron: job %s thiếu hàm Run", j.Name)
	}
	return nil
}

// Add đăng ký job; trùng tên hoặc không hợp lệ → lỗi.
func (r *Registry) Add(j Job) error {
	if err := j.Validate(); err != nil {
		return err
	}
	if j.Timeout <= 0 {
		j.Timeout = 10 * time.Minute
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.index[j.Name]; dup {
		return fmt.Errorf("cron: job %s đã được đăng ký", j.Name)
	}
	r.index[j.Name] = len(r.jobs)
	r.jobs = append(r.jobs, j)
	return nil
}

// Get tìm job theo tên.
func (r *Registry) Get(name string) (Job, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	i, ok := r.index[name]
	if !ok {
		return Job{}, false
	}
	return r.jobs[i], true
}

// List trả các job theo thứ tự tên.
func (r *Registry) List() []Job {
	r.mu.RLock()
	out := append([]Job(nil), r.jobs...)
	r.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// defaultRegistry là registry dùng chung của tiến trình.
var defaultRegistry = NewRegistry()

// Register thêm job vào registry mặc định. Panic khi job không hợp lệ hoặc trùng tên (lỗi lập trình –
// phát hiện ngay khi khởi động).
func Register(j Job) {
	if err := defaultRegistry.Add(j); err != nil {
		panic(err)
	}
}

// Jobs trả danh sách job đã đăng ký trong registry mặc định.
func Jobs() []Job { return defaultRegistry.List() }

// DefaultRegistry dùng cho Runner mặc định.
func DefaultRegistry() *Registry { return defaultRegistry }
