package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/job/v1"
	"gymmaster-api/internal/cron"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/mailer"
	"gymmaster-api/internal/repository"
)

// ---- fake ----

type jobFakeSubs struct {
	expireN   int
	expireErr error
	expiring  []repository.ExpiringSubscription
	gotDays   []int
}

func (f *jobFakeSubs) ExpireDue(context.Context, time.Time) (int, error) {
	return f.expireN, f.expireErr
}
func (f *jobFakeSubs) ApplyDueFreezes(context.Context, time.Time) (int, error) { return 2, nil }
func (f *jobFakeSubs) ListExpiring(_ context.Context, days []int, _ time.Time) ([]repository.ExpiringSubscription, error) {
	f.gotDays = days
	return f.expiring, nil
}

type jobFakeBookings struct{ at time.Time }

func (f *jobFakeBookings) MarkNoShows(_ context.Context, now time.Time) (int, error) {
	f.at = now
	return 4, nil
}

type jobFakeRepo struct {
	repository.JobRepository
	notices      []repository.JobExpiredNotice
	tokenBefore  time.Time
	lastRuns     map[string]entity.JobRun
	listRunsName string
}

func (f *jobFakeRepo) DBNow(context.Context) (time.Time, error) { return time.Now(), nil }
func (f *jobFakeRepo) ExpiredNotices(context.Context, time.Time) ([]repository.JobExpiredNotice, error) {
	return f.notices, nil
}
func (f *jobFakeRepo) DeleteStaleRefreshTokens(_ context.Context, before time.Time) (int64, error) {
	f.tokenBefore = before
	return 9, nil
}
func (f *jobFakeRepo) LastRuns(context.Context, []string) (map[string]entity.JobRun, error) {
	return f.lastRuns, nil
}
func (f *jobFakeRepo) ListRuns(_ context.Context, job, _ string, _ dto.Query) ([]entity.JobRun, int64, error) {
	f.listRunsName = job
	return []entity.JobRun{{ID: 1, Job: job, Status: "SUCCESS"}}, 1, nil
}

type jobNotif struct {
	userIDs  []uint
	perm     string
	branchID *uint
	typ      string
	title    string
	body     string
	link     string
}

type jobFakeNotifier struct{ sent []jobNotif }

func (f *jobFakeNotifier) Notify(_ context.Context, ids []uint, typ, title, body, link string) error {
	f.sent = append(f.sent, jobNotif{userIDs: ids, typ: typ, title: title, body: body, link: link})
	return nil
}
func (f *jobFakeNotifier) NotifyPermission(_ context.Context, perm string, b *uint, typ, title, body, link string) error {
	f.sent = append(f.sent, jobNotif{perm: perm, branchID: b, typ: typ, title: title, body: body, link: link})
	return nil
}

type jobFakeMailer struct {
	mu   sync.Mutex
	msgs []mailer.Message
}

func (f *jobFakeMailer) Send(_ context.Context, m mailer.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.msgs = append(f.msgs, m)
	return nil
}
func (f *jobFakeMailer) SendAsync(m mailer.Message) { _ = f.Send(context.Background(), m) }
func (f *jobFakeMailer) Wait()                      {}

type jobFakeSettings struct{ SettingService }

func (jobFakeSettings) IntSlice(_ context.Context, key string, def []int) []int {
	if key == SettingSubscriptionRemindBeforeDays {
		return []int{3, 0}
	}
	return def
}

func jobUint(v uint) *uint { return &v }

func jobTestTasks(subs *jobFakeSubs, repo *jobFakeRepo, n *jobFakeNotifier, m *jobFakeMailer) *JobTasks {
	t := NewJobTasks(subs, &jobFakeBookings{}, repo, jobFakeSettings{}, n, m)
	t.now = func() time.Time { return time.Date(2026, 10, 14, 8, 0, 0, 0, reportLoc) }
	return t
}

// ---- test ----

func TestJobExpireSubscriptionsNotifiesPortalMembers(t *testing.T) {
	subs := &jobFakeSubs{expireN: 3}
	repo := &jobFakeRepo{notices: []repository.JobExpiredNotice{
		{SubscriptionID: 1, MemberUserID: jobUint(70), PlanName: "Gói 3 tháng", BranchName: "Cầu Giấy", EndDate: time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)},
		{SubscriptionID: 2, MemberUserID: nil, PlanName: "Gói 1 tháng"}, // chưa có tài khoản cổng → bỏ qua
	}}
	n := &jobFakeNotifier{}
	tasks := jobTestTasks(subs, repo, n, &jobFakeMailer{})

	affected, err := tasks.ExpireSubscriptions(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, affected)
	require.Len(t, n.sent, 1)
	assert.Equal(t, []uint{70}, n.sent[0].userIDs)
	assert.Equal(t, NotifySubscriptionExpired, n.sent[0].typ)
	assert.Contains(t, n.sent[0].body, "13/10/2026")

	// Lỗi một phần vẫn trả affected + lỗi (job_runs ghi FAILED với số đã xử lý).
	subs.expireErr = errors.New("lỗi gói 5")
	affected, err = tasks.ExpireSubscriptions(context.Background())
	assert.Equal(t, 3, affected)
	assert.ErrorContains(t, err, "lỗi gói 5")
}

func TestJobRemindExpiring(t *testing.T) {
	end := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	subs := &jobFakeSubs{expiring: []repository.ExpiringSubscription{
		{SubscriptionID: 1, MemberUserID: jobUint(70), MemberName: "Lan", MemberEmail: "lan@x.vn", MemberPhone: "0901", PlanName: "Gói 3 tháng", BranchID: 1, BranchName: "Cầu Giấy", EndDate: end, DaysLeft: 3},
		{SubscriptionID: 2, MemberName: "Minh", MemberEmail: "", MemberPhone: "0902", PlanName: "Gói 1 tháng", BranchID: 1, BranchName: "Cầu Giấy", EndDate: end, DaysLeft: 3},
		{SubscriptionID: 3, MemberName: "Hoa", MemberEmail: " hoa@x.vn ", MemberPhone: "0903", PlanName: "Gói 6 tháng", BranchID: 2, BranchName: "Quận 1", EndDate: end, DaysLeft: 0},
	}}
	n, m := &jobFakeNotifier{}, &jobFakeMailer{}
	tasks := jobTestTasks(subs, &jobFakeRepo{}, n, m)

	affected, err := tasks.RemindExpiring(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, affected)
	assert.Equal(t, []int{3, 0}, subs.gotDays) // lấy từ settings

	// In-app chỉ cho hội viên có tài khoản cổng.
	var member, reception []jobNotif
	for _, s := range n.sent {
		if s.perm != "" {
			reception = append(reception, s)
		} else {
			member = append(member, s)
		}
	}
	require.Len(t, member, 1)
	assert.Equal(t, NotifySubscriptionExpiring, member[0].typ)
	assert.Contains(t, member[0].body, "còn 3 ngày")

	// Email cho hội viên có email (đã trim), nội dung HTML tiếng Việt.
	require.Len(t, m.msgs, 2)
	assert.Equal(t, []string{"lan@x.vn"}, m.msgs[0].To)
	assert.Contains(t, m.msgs[0].HTML, "17/10/2026")
	assert.Equal(t, []string{"hoa@x.vn"}, m.msgs[1].To)
	assert.Contains(t, m.msgs[1].Subject, "hết hạn hôm nay")

	// Tổng hợp cho lễ tân từng chi nhánh (quyền subscription.sell).
	require.Len(t, reception, 2)
	assert.Equal(t, "subscription.sell", reception[0].perm)
	assert.Equal(t, uint(1), *reception[0].branchID)
	assert.True(t, strings.HasPrefix(reception[0].title, "2 hội viên"))
	assert.Contains(t, reception[0].body, "Lan (0901")
	assert.Equal(t, "/admin/subscriptions?status=expiring&branch_id=1", reception[0].link)
	assert.Equal(t, uint(2), *reception[1].branchID)
}

func TestJobOtherTasks(t *testing.T) {
	repo := &jobFakeRepo{}
	b := &jobFakeBookings{}
	tasks := jobTestTasks(&jobFakeSubs{}, repo, &jobFakeNotifier{}, &jobFakeMailer{})
	tasks.bookings = b

	n, err := tasks.CleanupTokens(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 9, n)
	assert.Equal(t, tasks.now().Add(-30*24*time.Hour), repo.tokenBefore)

	n, _ = tasks.MarkClassNoShows(context.Background())
	assert.Equal(t, 4, n)
	assert.Equal(t, tasks.now(), b.at)

	n, _ = tasks.ApplyFreezes(context.Background())
	assert.Equal(t, 2, n)
}

// ---- JobService ----

type jobFakeScheduler struct {
	jobs    []cron.Job
	runErr  error
	ranName string
}

func (f *jobFakeScheduler) Jobs() []cron.Job { return f.jobs }
func (f *jobFakeScheduler) Run(_ context.Context, name string) (*entity.JobRun, error) {
	f.ranName = name
	if f.runErr != nil {
		return nil, f.runErr
	}
	fin := time.Date(2026, 10, 14, 0, 5, 2, 0, time.UTC)
	return &entity.JobRun{ID: 3, Job: name, StartedAt: fin.Add(-2 * time.Second), FinishedAt: &fin, Status: "SUCCESS", Affected: 5}, nil
}
func (f *jobFakeScheduler) Scheduled() bool           { return false }
func (f *jobFakeScheduler) IsRunning(string) bool     { return false }
func (f *jobFakeScheduler) NextRun(string) *time.Time { return nil }
func (f *jobFakeScheduler) NextBySpec(string) *time.Time {
	t := time.Date(2026, 10, 15, 0, 5, 0, 0, reportLoc)
	return &t
}

func TestJobService(t *testing.T) {
	sched := &jobFakeScheduler{jobs: []cron.Job{{Name: "a_job", Spec: "5 0 * * *"}, {Name: "b_job", Spec: "@daily"}}}
	repo := &jobFakeRepo{lastRuns: map[string]entity.JobRun{"a_job": {ID: 1, Job: "a_job", Status: "FAILED", Message: "x"}}}
	svc := NewJobService(sched, repo)

	list, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.NotNil(t, list[0].LastRun)
	assert.Equal(t, "FAILED", list[0].LastRun.Status)
	assert.Nil(t, list[1].LastRun)
	assert.NotNil(t, list[0].NextRunAt) // scheduler tắt → tính theo spec

	run, err := svc.Run(context.Background(), "a_job")
	require.NoError(t, err)
	assert.Equal(t, int64(2000), *run.DurationMs)
	assert.Equal(t, 5, run.Affected)

	sched.runErr = cron.ErrJobRunning
	_, err = svc.Run(context.Background(), "a_job")
	assert.Equal(t, 40900, reportErrCode(err))
	sched.runErr = cron.ErrJobNotFound
	_, err = svc.Run(context.Background(), "zzz")
	assert.Equal(t, 40400, reportErrCode(err))

	_, err = svc.Runs(context.Background(), &v1.RunsReq{Name: "zzz"})
	assert.Equal(t, 40400, reportErrCode(err))
	res, err := svc.Runs(context.Background(), &v1.RunsReq{Name: "b_job", PageReq: dto.PageReq{Page: 1, Limit: 10}})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Equal(t, "b_job", repo.listRunsName)
}

// ---- Audit diff ----

func TestAuditDiff(t *testing.T) {
	ch := auditDiff(json.RawMessage(`{"name":"A","price":100,"tags":[1,2],"same":{"x":1}}`),
		json.RawMessage(`{"price":120, "same":{"x":1},"tags":[1,2],"new":true}`))
	require.Len(t, ch, 3)
	assert.Equal(t, "name", ch[0].Field)
	assert.JSONEq(t, `null`, string(ch[0].After))
	assert.Equal(t, "new", ch[1].Field)
	assert.Equal(t, "price", ch[2].Field)
	assert.JSONEq(t, `120`, string(ch[2].After))

	assert.Empty(t, auditDiff(auditNull, auditNull))
	ch = auditDiff(auditNull, json.RawMessage(`{"a":1}`))
	require.Len(t, ch, 1)
	assert.Equal(t, "a", ch[0].Field)
	ch = auditDiff(json.RawMessage(`"x"`), json.RawMessage(`"y"`))
	require.Len(t, ch, 1)
	assert.Equal(t, "", ch[0].Field)
}
