package service

import (
	"context"
	"errors"
	"time"

	v1 "gymmaster-api/api/job/v1"
	"gymmaster-api/internal/cron"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

// JobScheduler: phần cron.Runner mà API job dùng (để test bằng fake).
type JobScheduler interface {
	Jobs() []cron.Job
	Run(ctx context.Context, name string) (*entity.JobRun, error)
	Scheduled() bool
	IsRunning(name string) bool
	NextRun(name string) *time.Time
	NextBySpec(spec string) *time.Time
}

// RunnerScheduler bọc *cron.Runner thành JobScheduler.
type RunnerScheduler struct{ *cron.Runner }

func (r RunnerScheduler) Jobs() []cron.Job { return r.Registry().List() }

// JobService: xem danh sách job, lịch sử chạy, chạy thủ công.
type JobService interface {
	List(ctx context.Context) (v1.ListRes, error)
	Runs(ctx context.Context, req *v1.RunsReq) (*v1.RunsRes, error)
	Run(ctx context.Context, name string) (*v1.JobRun, error)
}

type jobService struct {
	sched JobScheduler
	repo  repository.JobRepository
}

func NewJobService(sched JobScheduler, repo repository.JobRepository) JobService {
	return &jobService{sched: sched, repo: repo}
}

func jobRunToDTO(r entity.JobRun) *v1.JobRun {
	out := &v1.JobRun{
		ID: r.ID, Job: r.Job, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt,
		Status: r.Status, Affected: r.Affected, Message: r.Message,
	}
	if r.FinishedAt != nil {
		ms := r.FinishedAt.Sub(r.StartedAt).Milliseconds()
		out.DurationMs = &ms
	}
	return out
}

func (s *jobService) find(name string) (cron.Job, bool) {
	for _, j := range s.sched.Jobs() {
		if j.Name == name {
			return j, true
		}
	}
	return cron.Job{}, false
}

func (s *jobService) List(ctx context.Context) (v1.ListRes, error) {
	jobs := s.sched.Jobs()
	names := make([]string, len(jobs))
	for i, j := range jobs {
		names[i] = j.Name
	}
	last, err := s.repo.LastRuns(ctx, names)
	if err != nil {
		return nil, err
	}
	scheduled := s.sched.Scheduled()
	out := make(v1.ListRes, 0, len(jobs))
	for _, j := range jobs {
		item := &v1.Job{
			Name: j.Name, Spec: j.Spec, Description: j.Description, Scheduled: scheduled, Running: s.sched.IsRunning(j.Name),
		}
		if scheduled {
			item.NextRunAt = s.sched.NextRun(j.Name)
		} else {
			item.NextRunAt = s.sched.NextBySpec(j.Spec)
		}
		if r, ok := last[j.Name]; ok {
			item.LastRun = jobRunToDTO(r)
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *jobService) Runs(ctx context.Context, req *v1.RunsReq) (*v1.RunsRes, error) {
	if _, ok := s.find(req.Name); !ok {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy job")
	}
	q := req.ToQuery([]string{"started_at", "id"}, "started_at")
	rows, total, err := s.repo.ListRuns(ctx, req.Name, req.Status, q)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, func(r entity.JobRun) *v1.JobRun { return jobRunToDTO(r) }), total, q), nil
}

func (s *jobService) Run(ctx context.Context, name string) (*v1.JobRun, error) {
	run, err := s.sched.Run(ctx, name)
	switch {
	case errors.Is(err, cron.ErrJobNotFound):
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy job")
	case errors.Is(err, cron.ErrJobRunning):
		return nil, rescode.New(rescode.Conflict, "Job đang chạy, vui lòng thử lại sau")
	case err != nil:
		return nil, err
	}
	return jobRunToDTO(*run), nil
}
