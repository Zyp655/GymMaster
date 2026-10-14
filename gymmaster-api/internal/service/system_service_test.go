package service

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	auditv1 "gymmaster-api/api/auditlog/v1"
	jobv1 "gymmaster-api/api/job/v1"
	rolev1 "gymmaster-api/api/role/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/cron"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

// ---- 1. AuditLogService & auditDiff -----------------------------------------

type fakeAuditLogRepo struct {
	rows     map[uint]repository.AuditLogRow
	entities []string
	actions  []string
	lastFilt repository.AuditLogFilter
}

func (f *fakeAuditLogRepo) List(_ context.Context, flt repository.AuditLogFilter, _ dto.Query) ([]repository.AuditLogRow, int64, error) {
	f.lastFilt = flt
	out := make([]repository.AuditLogRow, 0, len(f.rows))
	for _, r := range f.rows {
		if flt.Entity != "" && r.Entity != flt.Entity {
			continue
		}
		if flt.Action != "" && r.Action != flt.Action {
			continue
		}
		out = append(out, r)
	}
	return out, int64(len(out)), nil
}

func (f *fakeAuditLogRepo) Get(_ context.Context, id uint) (*repository.AuditLogRow, error) {
	if r, ok := f.rows[id]; ok {
		cp := r
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeAuditLogRepo) Entities(context.Context) ([]string, error) { return f.entities, nil }
func (f *fakeAuditLogRepo) Actions(context.Context) ([]string, error)  { return f.actions, nil }

func TestAuditLogService(t *testing.T) {
	actorID := uint(9)
	entityID := uint(42)
	repo := &fakeAuditLogRepo{
		rows: map[uint]repository.AuditLogRow{
			1: {
				ID:         1,
				ActorID:    &actorID,
				ActorName:  "Nguyễn Quản Lý",
				ActorEmail: "quanly@gymmaster.vn",
				Action:     AuditUpdate,
				Entity:     "member",
				EntityID:   &entityID,
				Before:     datatypes.JSON(`{"status":"ACTIVE","phone":"0901112222","note":"cũ"}`),
				After:      datatypes.JSON(`{"status":"INACTIVE","phone":"0901112222","extra":true}`),
				IP:         "127.0.0.1",
				CreatedAt:  time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC),
			},
			2: {
				ID:        2,
				ActorID:   nil, // cron hệ thống
				Action:    "EXPIRE",
				Entity:    "subscription",
				CreatedAt: time.Date(2026, 10, 14, 1, 0, 0, 0, time.UTC),
			},
		},
	}
	svc := NewAuditLogService(repo)
	ctx := context.Background()

	// List với bộ lọc ngày hợp lệ và kiểm tra tên "Hệ thống" khi actor_id = nil
	res, err := svc.List(ctx, &auditv1.ListReq{
		PageReq: dto.PageReq{Page: 1, Limit: 20},
		From:    "2026-10-01",
		To:      "2026-10-14",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total)
	require.NotNil(t, repo.lastFilt.From)
	require.NotNil(t, repo.lastFilt.To)

	var sysRow *auditv1.AuditLog
	for _, it := range res.Items {
		if it.ID == 2 {
			sysRow = it
		}
	}
	require.NotNil(t, sysRow)
	assert.Equal(t, "Hệ thống", sysRow.ActorName)

	// Ngày lọc sai định dạng -> 400
	_, err = svc.List(ctx, &auditv1.ListReq{From: "14/10/2026"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	_, err = svc.List(ctx, &auditv1.ListReq{To: "invalid-date"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// Get chi tiết + tính toán diff (thêm extra, xóa note, đổi status, bỏ qua phone giữ nguyên)
	detail, err := svc.Get(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, "Nguyễn Quản Lý", detail.ActorName)
	require.Len(t, detail.Changes, 3)
	assert.Equal(t, "extra", detail.Changes[0].Field)
	assert.JSONEq(t, "null", string(detail.Changes[0].Before))
	assert.JSONEq(t, "true", string(detail.Changes[0].After))

	assert.Equal(t, "note", detail.Changes[1].Field)
	assert.JSONEq(t, `"cũ"`, string(detail.Changes[1].Before))
	assert.JSONEq(t, "null", string(detail.Changes[1].After))

	assert.Equal(t, "status", detail.Changes[2].Field)
	assert.JSONEq(t, `"ACTIVE"`, string(detail.Changes[2].Before))
	assert.JSONEq(t, `"INACTIVE"`, string(detail.Changes[2].After))

	// Get bản ghi không có before/after -> trả JSON null và changes rỗng
	emptyDetail, err := svc.Get(ctx, 2)
	require.NoError(t, err)
	assert.JSONEq(t, "null", string(emptyDetail.Before))
	assert.JSONEq(t, "null", string(emptyDetail.After))
	assert.Empty(t, emptyDetail.Changes)

	// Get ID không tồn tại -> 404
	_, err = svc.Get(ctx, 999)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))

	// Options luôn trả mảng không nil kể cả khi repo trả nil
	opts, err := svc.Options(ctx)
	require.NoError(t, err)
	assert.NotNil(t, opts.Entities)
	assert.NotNil(t, opts.Actions)
}

// ---- 2. JobService ----------------------------------------------------------

type fakeJobScheduler struct {
	jobs      []cron.Job
	scheduled bool
	running   map[string]bool
	nextAt    time.Time
	runErr    error
	runResult *entity.JobRun
}

func (f *fakeJobScheduler) Jobs() []cron.Job              { return f.jobs }
func (f *fakeJobScheduler) Scheduled() bool               { return f.scheduled }
func (f *fakeJobScheduler) IsRunning(name string) bool    { return f.running[name] }
func (f *fakeJobScheduler) NextRun(string) *time.Time     { return &f.nextAt }
func (f *fakeJobScheduler) NextBySpec(string) *time.Time  { return &f.nextAt }
func (f *fakeJobScheduler) Run(_ context.Context, name string) (*entity.JobRun, error) {
	if f.runErr != nil {
		return nil, f.runErr
	}
	return f.runResult, nil
}

type fakeJobRepo struct {
	repository.JobRepository
	runs     []entity.JobRun
	lastRuns map[string]entity.JobRun
}

func (f *fakeJobRepo) LastRuns(_ context.Context, _ []string) (map[string]entity.JobRun, error) {
	return f.lastRuns, nil
}

func (f *fakeJobRepo) ListRuns(_ context.Context, job, status string, _ dto.Query) ([]entity.JobRun, int64, error) {
	var out []entity.JobRun
	for _, r := range f.runs {
		if r.Job == job && (status == "" || r.Status == status) {
			out = append(out, r)
		}
	}
	return out, int64(len(out)), nil
}

func TestJobService_SchedulerStates(t *testing.T) {
	start := time.Date(2026, 10, 14, 0, 5, 0, 0, time.UTC)
	finish := start.Add(420 * time.Millisecond)
	next := start.Add(24 * time.Hour)

	sched := &fakeJobScheduler{
		jobs: []cron.Job{
			{Name: "subscriptions.daily", Spec: "0 5 0 * * *", Description: "Cập nhật gói tập hàng ngày"},
		},
		scheduled: true,
		running:   map[string]bool{"subscriptions.daily": false},
		nextAt:    next,
	}
	repo := &fakeJobRepo{
		runs: []entity.JobRun{
			{ID: 10, Job: "subscriptions.daily", StartedAt: start, FinishedAt: &finish, Status: "SUCCESS", Affected: 5, Message: "OK"},
		},
		lastRuns: map[string]entity.JobRun{
			"subscriptions.daily": {ID: 10, Job: "subscriptions.daily", StartedAt: start, FinishedAt: &finish, Status: "SUCCESS", Affected: 5, Message: "OK"},
		},
	}
	svc := NewJobService(sched, repo)
	ctx := context.Background()

	// List
	list, err := svc.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "subscriptions.daily", list[0].Name)
	assert.True(t, list[0].Scheduled)
	require.NotNil(t, list[0].LastRun)
	require.NotNil(t, list[0].LastRun.DurationMs)
	assert.Equal(t, int64(420), *list[0].LastRun.DurationMs)

	// Runs: job tồn tại vs không tồn tại
	runsRes, err := svc.Runs(ctx, &jobv1.RunsReq{
		PageReq: dto.PageReq{Page: 1, Limit: 10},
		Name:    "subscriptions.daily",
		Status:  "SUCCESS",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), runsRes.Total)

	_, err = svc.Runs(ctx, &jobv1.RunsReq{Name: "unknown.job"})
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))

	// Run thủ công: thành công, đang chạy (409), không tồn tại (404)
	sched.runResult = &entity.JobRun{ID: 11, Job: "subscriptions.daily", StartedAt: start, FinishedAt: &finish, Status: "SUCCESS", Affected: 3}
	runOut, err := svc.Run(ctx, "subscriptions.daily")
	require.NoError(t, err)
	assert.Equal(t, uint(11), runOut.ID)
	assert.Equal(t, 3, runOut.Affected)

	sched.runErr = cron.ErrJobRunning
	_, err = svc.Run(ctx, "subscriptions.daily")
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	sched.runErr = cron.ErrJobNotFound
	_, err = svc.Run(ctx, "unknown.job")
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}

// ---- 3. RoleService ---------------------------------------------------------

type fakeRoleRepo struct {
	nextID    uint
	roles     map[uint]*entity.Role
	perms     []entity.Permission
	userCount map[uint]int64
}

func newFakeRoleRepo() *fakeRoleRepo {
	perms := []entity.Permission{
		{ID: 1, Code: "member.view", Module: "member", Name: "Xem hội viên"},
		{ID: 2, Code: "member.create", Module: "member", Name: "Thêm hội viên"},
		{ID: 3, Code: "role.update", Module: "role", Name: "Cập nhật vai trò"},
	}
	return &fakeRoleRepo{
		nextID: 2,
		roles: map[uint]*entity.Role{
			1: {
				BaseEntity:  entity.BaseEntity{ID: 1},
				Code:        consts.RoleOwner,
				Name:        "Chủ phòng tập",
				IsSystem:    true,
				Permissions: perms,
			},
			2: {
				BaseEntity:  entity.BaseEntity{ID: 2},
				Code:        consts.RoleReceptionist,
				Name:        "Lễ tân",
				IsSystem:    true,
				Permissions: perms[:1],
			},
		},
		perms:     perms,
		userCount: map[uint]int64{},
	}
}

func (f *fakeRoleRepo) List(context.Context) ([]entity.Role, error) {
	out := make([]entity.Role, 0, len(f.roles))
	for _, r := range f.roles {
		out = append(out, *r)
	}
	return out, nil
}

func (f *fakeRoleRepo) FindByID(_ context.Context, id uint) (*entity.Role, error) {
	if r, ok := f.roles[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRoleRepo) FindByCode(_ context.Context, code string) (*entity.Role, error) {
	for _, r := range f.roles {
		if r.Code == code {
			cp := *r
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRoleRepo) ExistsCode(_ context.Context, code string) (bool, error) {
	for _, r := range f.roles {
		if r.Code == code {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRoleRepo) Create(_ context.Context, role *entity.Role) error {
	f.nextID++
	role.ID = f.nextID
	cp := *role
	f.roles[role.ID] = &cp
	return nil
}

func (f *fakeRoleRepo) Update(_ context.Context, role *entity.Role) error {
	if existing, ok := f.roles[role.ID]; ok {
		existing.Name = role.Name
		existing.Description = role.Description
		existing.UpdatedBy = role.UpdatedBy
		return nil
	}
	return gorm.ErrRecordNotFound
}

func (f *fakeRoleRepo) ReplacePermissions(_ context.Context, role *entity.Role, perms []entity.Permission) error {
	if existing, ok := f.roles[role.ID]; ok {
		cp := make([]entity.Permission, len(perms))
		copy(cp, perms)
		existing.Permissions = cp
	}
	return nil
}

func (f *fakeRoleRepo) Delete(_ context.Context, id uint) error {
	delete(f.roles, id)
	return nil
}

func (f *fakeRoleRepo) CountUsers(_ context.Context, id uint) (int64, error) {
	return f.userCount[id], nil
}

func (f *fakeRoleRepo) ListPermissions(context.Context) ([]entity.Permission, error) {
	return f.perms, nil
}

func (f *fakeRoleRepo) FindPermissionsByCodes(_ context.Context, codes []string) ([]entity.Permission, error) {
	var out []entity.Permission
	for _, code := range codes {
		for _, p := range f.perms {
			if p.Code == code {
				out = append(out, p)
			}
		}
	}
	return out, nil
}

func TestRoleService(t *testing.T) {
	repo := newFakeRoleRepo()
	svc := NewRoleService(repo, subNoTx{})
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})

	// ListPermissions & List
	perms, err := svc.ListPermissions(ctx)
	require.NoError(t, err)
	assert.Len(t, perms, 3)

	roles, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Len(t, roles, 2)

	// Create vai trò tùy chỉnh mới (khử trùng mã quyền)
	created, err := svc.Create(ctx, &rolev1.CreateReq{
		Code:        "ACCOUNTANT",
		Name:        "Kế toán",
		Description: "Phụ trách đối soát",
		Permissions: []string{"member.view", "member.view", "member.create"},
	})
	require.NoError(t, err)
	assert.Equal(t, "ACCOUNTANT", created.Code)
	assert.Equal(t, []string{"member.create", "member.view"}, created.Permissions)

	// Create trùng mã -> 409
	_, err = svc.Create(ctx, &rolev1.CreateReq{Code: "ACCOUNTANT", Name: "Trùng"})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	// Create với mã quyền không tồn tại -> 400
	_, err = svc.Create(ctx, &rolev1.CreateReq{
		Code:        "MARKETING",
		Name:        "Marketing",
		Permissions: []string{"member.view", "perm.khong_ton_tai"},
	})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// Update OWNER: luôn giữ toàn bộ quyền trong hệ thống dù request truyền mảng rỗng
	ownerUpdated, err := svc.Update(ctx, &rolev1.UpdateReq{
		ID:          1,
		Name:        "Chủ hệ thống GymMaster",
		Permissions: []string{},
	})
	require.NoError(t, err)
	assert.Equal(t, "Chủ hệ thống GymMaster", ownerUpdated.Name)
	assert.Len(t, ownerUpdated.Permissions, 3, "OWNER không bao giờ bị mất quyền")

	// Delete vai trò hệ thống (IsSystem=true) -> 409
	err = svc.Delete(ctx, 2)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	// Delete vai trò đang có người dùng -> 409
	repo.userCount[created.ID] = 2
	err = svc.Delete(ctx, created.ID)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	// Delete vai trò tùy chỉnh khi không còn người dùng -> thành công
	repo.userCount[created.ID] = 0
	require.NoError(t, svc.Delete(ctx, created.ID))
	_, err = svc.Get(ctx, created.ID)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}

// ---- 4. UploadService -------------------------------------------------------

func TestUploadService_MagicBytesAndLimits(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUploadService(tmpDir, "/uploads/", 1) // giới hạn 1 MB

	memberCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 10, Roles: []string{consts.RoleMember}})
	staffCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 2, Roles: []string{consts.RoleReceptionist}})

	// 1. Chưa đăng nhập hoặc chỉ có role MEMBER -> 403
	_, err := svc.SaveImage(context.Background(), 10, strings.NewReader("abc"))
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
	_, err = svc.SaveImage(memberCtx, 10, strings.NewReader("abc"))
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	// 2. Tệp rỗng -> 400
	_, err = svc.SaveImage(staffCtx, 0, bytes.NewReader(nil))
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// 3. Khai báo vượt dung lượng hoặc nội dung thực vượt dung lượng -> 400
	_, err = svc.SaveImage(staffCtx, 2<<20, strings.NewReader("abc"))
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	bigBuf := bytes.Repeat([]byte{0x89, 'P', 'N', 'G'}, (1<<20)/4+10)
	_, err = svc.SaveImage(staffCtx, 100, bytes.NewReader(bigBuf))
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// 4. Sai định dạng ảnh (text/plain) -> 400
	_, err = svc.SaveImage(staffCtx, 16, strings.NewReader("hello not an image"))
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// 5. Ảnh PNG hợp lệ (magic bytes 89 50 4E 47 0D 0A 1A 0A) -> lưu thành công
	pngBytes := append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, make([]byte, 32)...)
	url, err := svc.SaveImage(staffCtx, int64(len(pngBytes)), bytes.NewReader(pngBytes))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(url, "/uploads/"))
	assert.True(t, strings.HasSuffix(url, ".png"))

	relPath := strings.TrimPrefix(url, "/uploads/")
	saved, err := os.ReadFile(filepath.Join(tmpDir, filepath.FromSlash(relPath)))
	require.NoError(t, err)
	assert.Equal(t, pngBytes, saved)
}

// ---- 5. LandingService & SettingAdminService --------------------------------

type fakeSettingStore struct {
	rows map[string]entity.Setting
}

func (f *fakeSettingStore) Int(context.Context, string, int) int             { return 0 }
func (f *fakeSettingStore) IntSlice(context.Context, string, []int) []int    { return nil }
func (f *fakeSettingStore) All(context.Context) ([]entity.Setting, error) {
	out := make([]entity.Setting, 0, len(f.rows))
	for _, r := range f.rows {
		out = append(out, r)
	}
	return out, nil
}
func (f *fakeSettingStore) Set(_ context.Context, key string, value json.RawMessage, _ *uint) error {
	st, ok := f.rows[key]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	st.Value = datatypes.JSON(value)
	f.rows[key] = st
	return nil
}

func TestLandingAndSettingAdminService(t *testing.T) {
	planRepo := newFakePlanRepo()
	_ = planRepo.Create(context.Background(), &entity.MembershipPlan{Code: "M1", Name: "Gói 1 tháng", Status: consts.StatusActive, Price: 500_000})
	_ = planRepo.Create(context.Background(), &entity.MembershipPlan{Code: "OLD", Name: "Gói ngừng bán", Status: consts.StatusInactive, Price: 300_000})

	ptRepo := &fakePTRepo{items: map[uint]*entity.PTPackage{}, contracts: map[uint]int64{}}
	branchRepo := newFakeBranchRepo()
	_ = branchRepo.Create(context.Background(), &entity.Branch{Code: "HN01", Name: "Cầu Giấy", Status: consts.StatusActive})

	landing := NewLandingService(planRepo, ptRepo, branchRepo)
	plansRes, err := landing.Plans(context.Background())
	require.NoError(t, err)
	require.Len(t, plansRes.MembershipPlans, 1)
	assert.Equal(t, "M1", plansRes.MembershipPlans[0].Code)

	branchesRes, err := landing.Branches(context.Background())
	require.NoError(t, err)
	require.Len(t, branchesRes, 1)
	assert.Equal(t, "Cầu Giấy", branchesRes[0].Name)

	// SettingAdminService: List, Update hợp lệ (ghi audit) và Update khóa không tồn tại
	store := &fakeSettingStore{
		rows: map[string]entity.Setting{
			SettingCheckinQRTTLSeconds: {
				Key:         SettingCheckinQRTTLSeconds,
				Value:       datatypes.JSON("60"),
				Description: "Thời hạn mã QR (giây)",
			},
		},
	}
	audit := &fakeStaffAudit{}
	settingAdmin := NewSettingAdminService(store, audit)

	list, err := settingAdmin.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)

	updated, err := settingAdmin.Update(context.Background(), SettingCheckinQRTTLSeconds, 90)
	require.NoError(t, err)
	assert.JSONEq(t, "90", string(updated.Value))
	assert.Equal(t, []string{AuditUpdate}, audit.actions)

	_, err = settingAdmin.Update(context.Background(), SettingCheckinQRTTLSeconds, 5) // < min 15s
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	_, err = settingAdmin.Update(context.Background(), "khong.ton.tai", 10)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}
