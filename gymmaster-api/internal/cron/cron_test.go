package cron

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gymmaster-api/internal/entity"
)

type jobFakeStore struct {
	mu       sync.Mutex
	locked   map[string]bool
	runs     []*entity.JobRun
	finished []entity.JobRun
	failLock bool
}

func newJobFakeStore() *jobFakeStore { return &jobFakeStore{locked: map[string]bool{}} }

func (s *jobFakeStore) TryLock(_ context.Context, job string) (func(), bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failLock || s.locked[job] {
		return func() {}, false, nil
	}
	s.locked[job] = true
	return func() { s.mu.Lock(); delete(s.locked, job); s.mu.Unlock() }, true, nil
}

func (s *jobFakeStore) Start(_ context.Context, job string, at time.Time) (*entity.JobRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := &entity.JobRun{ID: uint(len(s.runs) + 1), Job: job, StartedAt: at, Status: StatusRunning}
	s.runs = append(s.runs, r)
	return r, nil
}

func (s *jobFakeStore) Finish(_ context.Context, r *entity.JobRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.finished = append(s.finished, *r)
	return nil
}

func (s *jobFakeStore) FailStale(context.Context, time.Time) (int64, error) { return 0, nil }

func TestRegistryValidation(t *testing.T) {
	r := NewRegistry()
	run := func(context.Context) (int, error) { return 0, nil }
	require.NoError(t, r.Add(Job{Name: "b_job", Spec: "*/15 * * * *", Run: run}))
	require.NoError(t, r.Add(Job{Name: "a_job", Spec: "@daily", Run: run}))
	assert.Error(t, r.Add(Job{Name: "a_job", Spec: "@daily", Run: run}), "trùng tên")
	assert.Error(t, r.Add(Job{Name: "Bad-Name", Spec: "@daily", Run: run}))
	assert.Error(t, r.Add(Job{Name: "c_job", Spec: "61 * * * *", Run: run}))
	assert.Error(t, r.Add(Job{Name: "d_job", Spec: "@daily"}))

	list := r.List()
	require.Len(t, list, 2)
	assert.Equal(t, "a_job", list[0].Name)
	assert.Equal(t, 10*time.Minute, list[0].Timeout)
}

func TestRunnerRecordsRunsAndPreventsOverlap(t *testing.T) {
	reg := NewRegistry()
	started, release := make(chan struct{}), make(chan struct{})
	require.NoError(t, reg.Add(Job{Name: "slow", Spec: "@daily", Run: func(context.Context) (int, error) {
		close(started)
		<-release
		return 3, nil
	}}))
	require.NoError(t, reg.Add(Job{Name: "bad", Spec: "@daily", Run: func(context.Context) (int, error) {
		return 1, errors.New("hỏng")
	}}))
	require.NoError(t, reg.Add(Job{Name: "boom", Spec: "@daily", Run: func(context.Context) (int, error) {
		panic("nổ")
	}}))
	store := newJobFakeStore()
	r := NewRunner(reg, store)

	done := make(chan *entity.JobRun)
	go func() {
		run, err := r.Run(context.Background(), "slow")
		assert.NoError(t, err)
		done <- run
	}()
	<-started
	assert.True(t, r.IsRunning("slow"))
	_, err := r.Run(context.Background(), "slow")
	assert.ErrorIs(t, err, ErrJobRunning)
	close(release)
	run := <-done
	assert.Equal(t, StatusSuccess, run.Status)
	assert.Equal(t, 3, run.Affected)
	assert.NotNil(t, run.FinishedAt)
	assert.False(t, r.IsRunning("slow"))

	run, err = r.Run(context.Background(), "bad")
	require.NoError(t, err)
	assert.Equal(t, StatusFailed, run.Status)
	assert.Equal(t, "hỏng", run.Message)
	assert.Equal(t, 1, run.Affected)

	run, err = r.Run(context.Background(), "boom")
	require.NoError(t, err)
	assert.Equal(t, StatusFailed, run.Status)
	assert.Contains(t, run.Message, "panic")

	_, err = r.Run(context.Background(), "none")
	assert.ErrorIs(t, err, ErrJobNotFound)

	// Instance khác đang giữ advisory lock → không chạy, không ghi job_runs.
	store.failLock = true
	before := len(store.runs)
	_, err = r.Run(context.Background(), "bad")
	assert.ErrorIs(t, err, ErrJobRunning)
	assert.Len(t, store.runs, before)
	assert.Len(t, store.finished, 3)
}

func TestRunnerSchedule(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.Add(Job{Name: "nightly", Spec: "5 0 * * *", Run: func(context.Context) (int, error) { return 0, nil }}))
	r := NewRunner(reg, newJobFakeStore())
	r.now = func() time.Time { return time.Date(2026, 10, 14, 10, 0, 0, 0, r.loc) }

	next := r.NextBySpec("5 0 * * *")
	require.NotNil(t, next)
	assert.Equal(t, time.Date(2026, 10, 15, 0, 5, 0, 0, r.loc), *next)
	assert.Nil(t, r.NextRun("nightly"))
	assert.False(t, r.Scheduled())

	require.NoError(t, r.Start(context.Background()))
	defer r.Stop()
	assert.True(t, r.Scheduled())
	nr := r.NextRun("nightly")
	require.NotNil(t, nr)
	assert.Equal(t, 0, nr.In(r.loc).Hour())
	assert.Equal(t, 5, nr.In(r.loc).Minute())
}
