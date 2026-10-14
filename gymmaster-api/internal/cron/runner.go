package cron

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	rcron "github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
)

// Trạng thái job_runs.status.
const (
	StatusRunning = "RUNNING"
	StatusSuccess = "SUCCESS"
	StatusFailed  = "FAILED"
)

var (
	ErrJobNotFound = errors.New("cron: không tìm thấy job")
	// ErrJobRunning: job đang chạy (trong tiến trình này hoặc ở instance khác giữ advisory lock).
	ErrJobRunning = errors.New("cron: job đang chạy")
)

// Store lưu vết lần chạy và khóa phân tán. Bản mặc định dùng PostgreSQL (job_runs + advisory lock).
type Store interface {
	// TryLock giữ khóa toàn cục cho job; ok=false khi instance khác đang giữ. release() trả khóa.
	TryLock(ctx context.Context, job string) (release func(), ok bool, err error)
	Start(ctx context.Context, job string, at time.Time) (*entity.JobRun, error)
	Finish(ctx context.Context, run *entity.JobRun) error
	// FailStale đánh dấu FAILED các lần chạy RUNNING bắt đầu trước mốc before (tiến trình bị dừng giữa chừng).
	FailStale(ctx context.Context, before time.Time) (int64, error)
}

// Runner chạy job theo lịch và theo yêu cầu (API chạy thủ công).
type Runner struct {
	reg   *Registry
	store Store
	loc   *time.Location
	now   func() time.Time

	mu      sync.Mutex
	locks   map[string]*sync.Mutex
	running map[string]bool
	sched   *rcron.Cron
	entries map[string]rcron.EntryID
}

func NewRunner(reg *Registry, store Store) *Runner {
	loc, err := time.LoadLocation(consts.TimeZone)
	if err != nil {
		loc = time.FixedZone("ICT", 7*3600)
	}
	return &Runner{
		reg: reg, store: store, loc: loc, now: time.Now,
		locks: map[string]*sync.Mutex{}, running: map[string]bool{}, entries: map[string]rcron.EntryID{},
	}
}

func (r *Runner) Registry() *Registry { return r.reg }

func (r *Runner) jobLock(name string) *sync.Mutex {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.locks[name]
	if !ok {
		m = &sync.Mutex{}
		r.locks[name] = m
	}
	return m
}

// IsRunning: job đang chạy trong tiến trình này.
func (r *Runner) IsRunning(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running[name]
}

func (r *Runner) setRunning(name string, v bool) {
	r.mu.Lock()
	r.running[name] = v
	r.mu.Unlock()
}

// Scheduled: scheduler đang chạy (CRON_ENABLED=true và đã Start).
func (r *Runner) Scheduled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sched != nil
}

// NextRun: lần chạy kế tiếp theo lịch (nil nếu scheduler không chạy).
func (r *Runner) NextRun(name string) *time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sched == nil {
		return nil
	}
	id, ok := r.entries[name]
	if !ok {
		return nil
	}
	t := r.sched.Entry(id).Next
	if t.IsZero() {
		return nil
	}
	return &t
}

// NextBySpec tính lần chạy kế tiếp từ spec (dùng khi scheduler tắt, để hiển thị).
func (r *Runner) NextBySpec(spec string) *time.Time {
	s, err := specParser.Parse(spec)
	if err != nil {
		return nil
	}
	t := s.Next(r.now().In(r.loc))
	return &t
}

// Run chạy job ngay (đồng bộ) và trả bản ghi job_runs. Lỗi của job nằm trong run.Status/Message,
// error trả về chỉ dành cho: không tìm thấy job, đang chạy, lỗi hạ tầng (ghi job_runs, khóa).
func (r *Runner) Run(_ context.Context, name string) (*entity.JobRun, error) {
	job, ok := r.reg.Get(name)
	if !ok {
		return nil, ErrJobNotFound
	}
	m := r.jobLock(name)
	if !m.TryLock() {
		return nil, ErrJobRunning
	}
	defer m.Unlock()

	// Job luôn chạy với tư cách "hệ thống": context mới, không mang thông tin đăng nhập của request
	// (actor_id = NULL trong audit/subscription_events) và không bị hủy khi client ngắt kết nối.
	base := context.Background()
	release, locked, err := r.store.TryLock(base, name)
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, ErrJobRunning
	}
	defer release()

	run, err := r.store.Start(base, name, r.now())
	if err != nil {
		return nil, err
	}
	r.setRunning(name, true)
	defer r.setRunning(name, false)

	runCtx, cancel := context.WithTimeout(base, job.Timeout)
	affected, jobErr := r.safeRun(runCtx, job)
	cancel()

	fin := r.now()
	run.FinishedAt = &fin
	run.Affected = affected
	if jobErr != nil {
		run.Status = StatusFailed
		run.Message = jobErr.Error()
	} else {
		run.Status = StatusSuccess
	}
	if err := r.store.Finish(base, run); err != nil {
		return run, err
	}
	return run, nil
}

func (r *Runner) safeRun(ctx context.Context, job Job) (n int, err error) {
	defer func() {
		if p := recover(); p != nil {
			g.Log().Errorf(ctx, "cron: job %s panic: %v\n%s", job.Name, p, debug.Stack())
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	return job.Run(ctx)
}

// Start bật scheduler cho mọi job đã đăng ký (giờ Asia/Ho_Chi_Minh). Gọi một lần sau khi các module đã Register.
func (r *Runner) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.sched != nil {
		r.mu.Unlock()
		return nil
	}
	sched := rcron.New(rcron.WithLocation(r.loc), rcron.WithParser(specParser))
	r.mu.Unlock()

	if n, err := r.store.FailStale(ctx, r.now().Add(-6*time.Hour)); err != nil {
		g.Log().Warningf(ctx, "cron: dọn job_runs treo: %v", err)
	} else if n > 0 {
		g.Log().Warningf(ctx, "cron: đánh dấu FAILED %d lần chạy bị gián đoạn", n)
	}

	entries := map[string]rcron.EntryID{}
	for _, job := range r.reg.List() {
		name := job.Name
		id, err := sched.AddFunc(job.Spec, func() {
			run, err := r.Run(context.Background(), name)
			switch {
			case errors.Is(err, ErrJobRunning):
				g.Log().Infof(context.Background(), "cron: bỏ qua %s – đang chạy ở nơi khác", name)
			case err != nil:
				g.Log().Errorf(context.Background(), "cron: %s lỗi hạ tầng: %v", name, err)
			case run.Status == StatusFailed:
				g.Log().Warningf(context.Background(), "cron: %s FAILED (affected=%d): %s", name, run.Affected, run.Message)
			default:
				g.Log().Infof(context.Background(), "cron: %s SUCCESS (affected=%d)", name, run.Affected)
			}
		})
		if err != nil {
			return fmt.Errorf("cron: lên lịch %s: %w", name, err)
		}
		entries[name] = id
	}
	r.mu.Lock()
	r.sched, r.entries = sched, entries
	r.mu.Unlock()
	sched.Start()
	g.Log().Infof(ctx, "cron: scheduler đã bật với %d job (%s)", len(entries), r.loc)
	return nil
}

// Stop dừng scheduler và chờ các job đang chạy kết thúc.
func (r *Runner) Stop() {
	r.mu.Lock()
	sched := r.sched
	r.sched = nil
	r.mu.Unlock()
	if sched != nil {
		<-sched.Stop().Done()
	}
}

// ---------------------------------------------------------------------------
// Store PostgreSQL
// ---------------------------------------------------------------------------

type pgStore struct {
	db *gorm.DB
}

func NewPGStore(db *gorm.DB) Store { return &pgStore{db: db} }

// advisoryKey: khóa advisory theo tên job (hashtext → int4, đủ để phân biệt các job).
const advisoryKey = "gymmaster.job."

func (s *pgStore) TryLock(ctx context.Context, job string) (func(), bool, error) {
	sqlDB, err := s.db.DB()
	if err != nil {
		return nil, false, err
	}
	// Advisory lock gắn với session → giữ riêng một kết nối cho tới khi nhả khóa.
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, false, err
	}
	var ok bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(hashtext($1))", advisoryKey+job).Scan(&ok); err != nil {
		_ = conn.Close()
		return nil, false, err
	}
	if !ok {
		_ = conn.Close()
		return func() {}, false, nil
	}
	return func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock(hashtext($1))", advisoryKey+job)
		_ = conn.Close()
	}, true, nil
}

func (s *pgStore) Start(ctx context.Context, job string, at time.Time) (*entity.JobRun, error) {
	run := &entity.JobRun{Job: job, StartedAt: at, Status: StatusRunning}
	if err := s.db.WithContext(ctx).Create(run).Error; err != nil {
		return nil, err
	}
	return run, nil
}

func (s *pgStore) Finish(ctx context.Context, run *entity.JobRun) error {
	return s.db.WithContext(ctx).Model(&entity.JobRun{}).Where("id = ?", run.ID).Updates(map[string]any{
		"finished_at": run.FinishedAt, "status": run.Status, "affected": run.Affected, "message": run.Message,
	}).Error
}

func (s *pgStore) FailStale(ctx context.Context, before time.Time) (int64, error) {
	res := s.db.WithContext(ctx).Model(&entity.JobRun{}).
		Where("status = ? AND started_at < ?", StatusRunning, before).
		Updates(map[string]any{"status": StatusFailed, "finished_at": time.Now(), "message": "Bị gián đoạn (tiến trình dừng khi đang chạy)"})
	return res.RowsAffected, res.Error
}

// ---------------------------------------------------------------------------
// Runner mặc định của tiến trình
// ---------------------------------------------------------------------------

var (
	defaultOnce   sync.Once
	defaultRunner *Runner
)

// Default trả Runner dùng chung (registry mặc định + store PostgreSQL). Module job API và cmd dùng chung đối tượng này.
func Default(db *gorm.DB) *Runner {
	defaultOnce.Do(func() { defaultRunner = NewRunner(defaultRegistry, NewPGStore(db)) })
	return defaultRunner
}
