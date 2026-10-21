package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/staff/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/hash"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

// ---- fakes ------------------------------------------------------------------

var staffTestRoles = map[string]*entity.Role{
	consts.RoleOwner:        {BaseEntity: entity.BaseEntity{ID: 1}, Code: consts.RoleOwner, Name: "Chủ trung tâm"},
	consts.RoleManager:      {BaseEntity: entity.BaseEntity{ID: 2}, Code: consts.RoleManager, Name: "Quản lý"},
	consts.RoleReceptionist: {BaseEntity: entity.BaseEntity{ID: 3}, Code: consts.RoleReceptionist, Name: "Lễ tân"},
	consts.RoleTrainer:      {BaseEntity: entity.BaseEntity{ID: 4}, Code: consts.RoleTrainer, Name: "HLV"},
	consts.RoleMember:       {BaseEntity: entity.BaseEntity{ID: 5}, Code: consts.RoleMember, Name: "Hội viên"},
}

func staffRoleByID(id uint) entity.Role {
	for _, r := range staffTestRoles {
		if r.ID == id {
			return *r
		}
	}
	return entity.Role{}
}

// fakeStaffRoleRepo chỉ cài FindByCode; các method khác không dùng tới.
type fakeStaffRoleRepo struct {
	repository.RoleRepository
}

func (fakeStaffRoleRepo) FindByCode(_ context.Context, code string) (*entity.Role, error) {
	if r, ok := staffTestRoles[code]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

// FindByID trả vai trò kèm quyền: vai trò MANAGER (id 2) có "payroll.manage" để kiểm tra chống leo thang.
func (fakeStaffRoleRepo) FindByID(_ context.Context, id uint) (*entity.Role, error) {
	r := staffRoleByID(id)
	if r.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if r.Code == consts.RoleManager {
		r.Permissions = []entity.Permission{{Code: "payroll.manage"}}
	}
	return &r, nil
}

type fakeStaffAudit struct{ actions []string }

func (f *fakeStaffAudit) Log(_ context.Context, action, _ string, _ uint, _, _ any) {
	f.actions = append(f.actions, action)
}

type fakeStaffRepo struct {
	nextID      uint
	users       map[uint]*entity.StaffAccount
	branches    map[uint]string
	futureClass map[uint]int64
	futurePT    map[uint]int64
	status      map[uint]string
	deleted     []uint
}

func newFakeStaffRepo() *fakeStaffRepo {
	return &fakeStaffRepo{
		nextID:      100,
		users:       map[uint]*entity.StaffAccount{},
		branches:    map[uint]string{1: "CN 1", 2: "CN 2"},
		futureClass: map[uint]int64{},
		futurePT:    map[uint]int64{},
		status:      map[uint]string{},
	}
}

func (f *fakeStaffRepo) add(id uint, email string, roles ...entity.UserRole) {
	for i := range roles {
		roles[i].UserID = id
		roles[i].Role = staffRoleByID(roles[i].RoleID)
	}
	f.users[id] = &entity.StaffAccount{
		User:         entity.User{BaseEntity: entity.BaseEntity{ID: id}, Email: email, FullName: email, Status: consts.StatusActive, UserRoles: roles},
		StaffProfile: &entity.StaffProfile{UserID: id, StaffCode: "NV9999", Position: StaffPositionOther},
	}
}

func (f *fakeStaffRepo) List(context.Context, dto.Query, repository.StaffFilter) ([]entity.StaffAccount, int64, error) {
	return nil, 0, nil
}
func (f *fakeStaffRepo) FindByID(_ context.Context, id uint) (*entity.StaffAccount, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *u
	cp.UserRoles = append([]entity.UserRole{}, u.UserRoles...)
	return &cp, nil
}
func (f *fakeStaffRepo) EmailExists(_ context.Context, email string, excludeID uint) (bool, error) {
	for id, u := range f.users {
		if id != excludeID && strings.EqualFold(u.Email, email) {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeStaffRepo) NextStaffCode(context.Context) (string, error) { return "NV0001", nil }
func (f *fakeStaffRepo) CreateUser(_ context.Context, u *entity.User) error {
	f.nextID++
	u.ID = f.nextID
	f.users[u.ID] = &entity.StaffAccount{User: *u}
	return nil
}
func (f *fakeStaffRepo) UpdateUser(_ context.Context, u *entity.User) error {
	f.users[u.ID].Email, f.users[u.ID].FullName = u.Email, u.FullName
	return nil
}
func (f *fakeStaffRepo) SaveProfile(_ context.Context, p *entity.StaffProfile) error {
	cp := *p
	f.users[p.UserID].StaffProfile = &cp
	return nil
}
func (f *fakeStaffRepo) SaveTrainerProfile(_ context.Context, p *entity.TrainerProfile) error {
	cp := *p
	f.users[p.UserID].TrainerProfile = &cp
	return nil
}
func (f *fakeStaffRepo) DeleteTrainerProfile(_ context.Context, id uint) error {
	f.users[id].TrainerProfile = nil
	return nil
}
func (f *fakeStaffRepo) ReplaceRoles(_ context.Context, id uint, roles []entity.UserRole) error {
	out := make([]entity.UserRole, 0, len(roles))
	for _, r := range roles {
		r.UserID = id
		r.Role = staffRoleByID(r.RoleID)
		out = append(out, r)
	}
	f.users[id].UserRoles = out
	return nil
}
func (f *fakeStaffRepo) UpdateStatus(_ context.Context, id uint, status string, _ *uint) error {
	f.users[id].Status = status
	f.status[id] = status
	return nil
}
func (f *fakeStaffRepo) UpdatePassword(_ context.Context, id uint, h string, _ *uint) error {
	f.users[id].PasswordHash = h
	return nil
}
func (f *fakeStaffRepo) SoftDelete(_ context.Context, id uint, _ *uint) error {
	delete(f.users, id)
	f.deleted = append(f.deleted, id)
	return nil
}
func (f *fakeStaffRepo) CountFutureAssignments(_ context.Context, id uint, _ time.Time) (int64, int64, error) {
	return f.futureClass[id], f.futurePT[id], nil
}
func (f *fakeStaffRepo) DeleteFutureShifts(context.Context, uint, time.Time) error { return nil }
func (f *fakeStaffRepo) BranchNames(_ context.Context, ids []uint) (map[uint]string, error) {
	out := map[uint]string{}
	for _, id := range ids {
		if n, ok := f.branches[id]; ok {
			out[id] = n
		}
	}
	return out, nil
}
func (f *fakeStaffRepo) ActiveOptions(context.Context, []uint, bool) ([]entity.StaffAccount, error) {
	return nil, nil
}

// ---- fixture ----------------------------------------------------------------

func staffU(v uint) *uint { return &v }

type staffFixture struct {
	svc     *staffService
	repo    *fakeStaffRepo
	refresh *fakeRefreshRepo
	audit   *fakeStaffAudit
}

// Người dùng sẵn có:
//
//	1: OWNER (toàn hệ thống)
//	2: MANAGER chi nhánh 1
//	3: RECEPTIONIST chi nhánh 1
//	4: TRAINER chi nhánh 2
func newStaffFixture() *staffFixture {
	repo := newFakeStaffRepo()
	repo.add(1, "owner@gm.vn", entity.UserRole{RoleID: 1})
	repo.add(2, "manager@gm.vn", entity.UserRole{RoleID: 2, BranchID: staffU(1)})
	repo.add(3, "lt@gm.vn", entity.UserRole{RoleID: 3, BranchID: staffU(1)})
	repo.add(4, "hlv@gm.vn", entity.UserRole{RoleID: 4, BranchID: staffU(2)})
	refresh := &fakeRefreshRepo{tokens: map[string]*entity.RefreshToken{}}
	audit := &fakeStaffAudit{}
	svc := NewStaffService(repo, fakeStaffRoleRepo{}, refresh, noTx{}, audit).(*staffService)
	return &staffFixture{svc: svc, repo: repo, refresh: refresh, audit: audit}
}

func ownerCtx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
}

func managerCtx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{UserID: 2, Roles: []string{consts.RoleManager}, BranchIDs: []uint{1}})
}

func newStaffReq(email string, roles ...v1.RoleAssignment) *v1.CreateReq {
	return &v1.CreateReq{
		StaffInput: v1.StaffInput{FullName: "Nhân viên mới", Email: email, Position: StaffPositionReceptionist},
		Password:   "Secret@123",
		Roles:      roles,
	}
}

// ---- tests ------------------------------------------------------------------

func TestStaffCreate_Success(t *testing.T) {
	f := newStaffFixture()
	req := newStaffReq("  New@GM.vn ", v1.RoleAssignment{RoleCode: "receptionist", BranchID: staffU(1)})
	req.Position = StaffPositionTrainer
	req.TrainerProfile = &v1.TrainerProfileInput{Level: "SENIOR", Specialties: []string{"Yoga", " yoga ", "", "Boxing"}}

	res, err := f.svc.Create(managerCtx(), req)
	require.NoError(t, err)
	assert.Equal(t, "new@gm.vn", res.Email, "email được chuẩn hóa chữ thường")
	require.Len(t, res.Roles, 1)
	assert.Equal(t, consts.RoleReceptionist, res.Roles[0].RoleCode)
	assert.Equal(t, "CN 1", res.Roles[0].BranchName)
	require.NotNil(t, res.StaffProfile)
	assert.Equal(t, "NV0001", res.StaffProfile.StaffCode)
	require.NotNil(t, res.TrainerProfile, "position TRAINER phải có hồ sơ HLV")
	assert.Equal(t, []string{"Yoga", "Boxing"}, res.TrainerProfile.Specialties, "chuyên môn bỏ trống/trùng")

	u := f.repo.users[res.ID]
	assert.True(t, hash.CheckPassword(u.PasswordHash, "Secret@123"), "mật khẩu phải được băm bcrypt")
	assert.Contains(t, f.audit.actions, AuditCreate)
}

func TestStaffCreate_DuplicateEmailCaseInsensitive(t *testing.T) {
	f := newStaffFixture()
	_, err := f.svc.Create(ownerCtx(), newStaffReq("LT@gm.VN", v1.RoleAssignment{RoleCode: "TRAINER", BranchID: staffU(1)}))
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
}

func TestStaffCreate_RoleRules(t *testing.T) {
	cases := []struct {
		name string
		ctx  context.Context
		role v1.RoleAssignment
		code int
	}{
		{"không ai được gán MEMBER", ownerCtx(), v1.RoleAssignment{RoleCode: "MEMBER", BranchID: staffU(1)}, rescode.BadRequest.Code()},
		{"vai trò không tồn tại", ownerCtx(), v1.RoleAssignment{RoleCode: "NOPE", BranchID: staffU(1)}, rescode.BadRequest.Code()},
		{"chi nhánh không tồn tại", ownerCtx(), v1.RoleAssignment{RoleCode: "TRAINER", BranchID: staffU(99)}, rescode.BadRequest.Code()},
		{"chỉ OWNER gán được OWNER", managerCtx(), v1.RoleAssignment{RoleCode: "OWNER"}, rescode.PermissionDenied.Code()},
		{"non-OWNER phải chọn chi nhánh", managerCtx(), v1.RoleAssignment{RoleCode: "TRAINER"}, rescode.BadRequest.Code()},
		{"non-OWNER chỉ gán trong chi nhánh của mình", managerCtx(), v1.RoleAssignment{RoleCode: "TRAINER", BranchID: staffU(2)}, rescode.PermissionDenied.Code()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newStaffFixture()
			_, err := f.svc.Create(tc.ctx, newStaffReq("x@gm.vn", tc.role))
			require.Error(t, err)
			assert.Equal(t, tc.code, codeOf(err))
			assert.Len(t, f.repo.users, 4, "không được tạo user khi vai trò không hợp lệ")
		})
	}
}

func TestStaffCreate_NonOwnerCannotAssignRoleWithExtraPermissions(t *testing.T) {
	f := newStaffFixture()
	// Người thao tác không có payroll.manage → không được gán vai trò MANAGER (có payroll.manage).
	_, err := f.svc.Create(managerCtx(), newStaffReq("x@gm.vn", v1.RoleAssignment{RoleCode: "MANAGER", BranchID: staffU(1)}))
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	// Có đủ quyền thì gán được.
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 2, Roles: []string{consts.RoleManager},
		Permissions: []string{"payroll.manage"}, BranchIDs: []uint{1}})
	_, err = f.svc.Create(ctx, newStaffReq("y@gm.vn", v1.RoleAssignment{RoleCode: "MANAGER", BranchID: staffU(1)}))
	require.NoError(t, err)
}

func TestStaffCreate_OwnerMayAssignGlobalRoles(t *testing.T) {
	f := newStaffFixture()
	res, err := f.svc.Create(ownerCtx(), newStaffReq("x@gm.vn",
		v1.RoleAssignment{RoleCode: "OWNER", BranchID: staffU(1)}, // OWNER luôn toàn hệ thống
		v1.RoleAssignment{RoleCode: "MANAGER"},
	))
	require.NoError(t, err)
	require.Len(t, res.Roles, 2)
	for _, r := range res.Roles {
		assert.Nil(t, r.BranchID)
	}
}

func TestStaffCreate_RequiresRoles(t *testing.T) {
	f := newStaffFixture()
	_, err := f.svc.Create(ownerCtx(), newStaffReq("x@gm.vn"))
	require.Error(t, err)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
}

func TestStaffGet_BranchScope(t *testing.T) {
	f := newStaffFixture()
	_, err := f.svc.Get(managerCtx(), 3)
	require.NoError(t, err, "cùng chi nhánh")

	_, err = f.svc.Get(managerCtx(), 4)
	require.Error(t, err, "HLV chi nhánh 2 không hiển thị với quản lý chi nhánh 1")
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))

	_, err = f.svc.Get(ownerCtx(), 4)
	require.NoError(t, err)
}

func TestStaffLock_RevokesTokensAndForbidsSelf(t *testing.T) {
	f := newStaffFixture()
	f.refresh.tokens["h1"] = &entity.RefreshToken{ID: 1, UserID: 3, ExpiresAt: time.Now().Add(time.Hour)}
	f.refresh.tokens["h2"] = &entity.RefreshToken{ID: 2, UserID: 3, ExpiresAt: time.Now().Add(time.Hour)}

	res, err := f.svc.Lock(managerCtx(), 3)
	require.NoError(t, err)
	assert.Equal(t, consts.StatusLocked, res.Status)
	assert.Equal(t, 0, f.refresh.activeCount(3), "khóa phải thu hồi mọi refresh token")
	assert.Contains(t, f.audit.actions, auditStaffLock)

	_, err = f.svc.Lock(managerCtx(), 2)
	require.Error(t, err)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "không được tự khóa chính mình")

	res, err = f.svc.Unlock(managerCtx(), 3)
	require.NoError(t, err)
	assert.Equal(t, consts.StatusActive, res.Status)
}

func TestStaffManage_NonOwnerCannotTouchOwner(t *testing.T) {
	f := newStaffFixture()
	// Quản lý toàn hệ thống (không gắn chi nhánh) vẫn không thao tác được trên OWNER.
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 2, Roles: []string{consts.RoleManager}})
	_, err := f.svc.Lock(ctx, 1)
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	err = f.svc.ResetPassword(ctx, 1, "Another@123")
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
}

func TestStaffDelete(t *testing.T) {
	t.Run("không được tự xóa", func(t *testing.T) {
		f := newStaffFixture()
		err := f.svc.Delete(ownerCtx(), 1)
		require.Error(t, err)
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	})
	t.Run("còn lịch lớp tương lai → 409", func(t *testing.T) {
		f := newStaffFixture()
		f.repo.futureClass[4] = 2
		err := f.svc.Delete(ownerCtx(), 4)
		require.Error(t, err)
		assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
		assert.Empty(t, f.repo.deleted)
	})
	t.Run("còn buổi PT tương lai → 409", func(t *testing.T) {
		f := newStaffFixture()
		f.repo.futurePT[4] = 1
		err := f.svc.Delete(ownerCtx(), 4)
		require.Error(t, err)
		assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	})
	t.Run("xóa mềm + thu hồi token", func(t *testing.T) {
		f := newStaffFixture()
		f.refresh.tokens["h"] = &entity.RefreshToken{ID: 1, UserID: 4, ExpiresAt: time.Now().Add(time.Hour)}
		require.NoError(t, f.svc.Delete(ownerCtx(), 4))
		assert.Equal(t, []uint{4}, f.repo.deleted)
		assert.Equal(t, 0, f.refresh.activeCount(4))
		assert.Contains(t, f.audit.actions, AuditDelete)
	})
}

func TestStaffAssignRoles(t *testing.T) {
	t.Run("không tự đổi vai trò của mình", func(t *testing.T) {
		f := newStaffFixture()
		_, err := f.svc.AssignRoles(managerCtx(), 2, []v1.RoleAssignment{{RoleCode: "TRAINER", BranchID: staffU(1)}})
		require.Error(t, err)
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	})
	t.Run("quản lý chi nhánh giữ nguyên vai trò ngoài phạm vi", func(t *testing.T) {
		f := newStaffFixture()
		// Lễ tân 3: lễ tân chi nhánh 1 + HLV chi nhánh 2.
		f.repo.users[3].UserRoles = append(f.repo.users[3].UserRoles,
			entity.UserRole{UserID: 3, RoleID: 4, BranchID: staffU(2), Role: staffRoleByID(4)})

		res, err := f.svc.AssignRoles(managerCtx(), 3, []v1.RoleAssignment{{RoleCode: "TRAINER", BranchID: staffU(1)}})
		require.NoError(t, err)
		got := []string{}
		for _, r := range res.Roles {
			got = append(got, fmt.Sprintf("%s@%d", r.RoleCode, *r.BranchID))
		}
		// HLV CN2 ngoài phạm vi được giữ nguyên; lễ tân CN1 bị thay bằng HLV CN1.
		assert.ElementsMatch(t, []string{"TRAINER@2", "TRAINER@1"}, got)
		assert.Contains(t, f.audit.actions, auditStaffRoles)
	})
	t.Run("OWNER thay toàn bộ", func(t *testing.T) {
		f := newStaffFixture()
		res, err := f.svc.AssignRoles(ownerCtx(), 4, []v1.RoleAssignment{{RoleCode: "MANAGER", BranchID: staffU(1)}, {RoleCode: "MANAGER", BranchID: staffU(1)}})
		require.NoError(t, err)
		require.Len(t, res.Roles, 1, "dòng trùng bị gộp")
		assert.Equal(t, consts.RoleManager, res.Roles[0].RoleCode)
	})
}

func TestStaffUpdate_PositionChangeDropsTrainerProfile(t *testing.T) {
	f := newStaffFixture()
	f.repo.users[4].StaffProfile.Position = StaffPositionTrainer
	f.repo.users[4].TrainerProfile = &entity.TrainerProfile{UserID: 4, Level: "SENIOR"}

	res, err := f.svc.Update(ownerCtx(), 4, v1.StaffInput{FullName: "HLV", Email: "hlv@gm.vn", Position: StaffPositionOther})
	require.NoError(t, err)
	assert.Nil(t, res.TrainerProfile)
	assert.Nil(t, f.repo.users[4].TrainerProfile)
	assert.Equal(t, "NV9999", res.StaffProfile.StaffCode, "giữ nguyên mã nhân viên")

	_, err = f.svc.Update(ownerCtx(), 4, v1.StaffInput{FullName: "HLV", Email: "LT@gm.vn", Position: StaffPositionOther})
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err), "email trùng người khác")
}

func TestStaffResetPassword(t *testing.T) {
	f := newStaffFixture()
	f.refresh.tokens["h"] = &entity.RefreshToken{ID: 1, UserID: 3, ExpiresAt: time.Now().Add(time.Hour)}

	err := f.svc.ResetPassword(managerCtx(), 3, "short")
	require.Error(t, err)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	require.NoError(t, f.svc.ResetPassword(managerCtx(), 3, "NewPass@123"))
	assert.True(t, hash.CheckPassword(f.repo.users[3].PasswordHash, "NewPass@123"))
	assert.Equal(t, 0, f.refresh.activeCount(3))
}

func TestStaffTextArrayRoundTrip(t *testing.T) {
	in := entity.StaffTextArray{"Yoga", `Giảm cân, "cardio"`, `a\b`}
	v, err := in.Value()
	require.NoError(t, err)

	var out entity.StaffTextArray
	require.NoError(t, out.Scan(v))
	assert.Equal(t, in, out)

	require.NoError(t, out.Scan("{}"))
	assert.Empty(t, out)
	require.NoError(t, out.Scan("{Yoga,Boxing}"))
	assert.Equal(t, entity.StaffTextArray{"Yoga", "Boxing"}, out)
}

// Hồi quy QA vòng 2: chiếm tài khoản nhân viên có quyền cao hơn qua đặt lại mật khẩu/đổi email/khóa.
func TestStaffManage_TargetPermissionsMustBeSubsetOfCaller(t *testing.T) {
	// Quản lý 5 cùng chi nhánh 1: vai trò MANAGER có payroll.manage – quản lý 2 (không có quyền này) không được đụng tới.
	newF := func() *staffFixture {
		f := newStaffFixture()
		f.repo.add(5, "ql2@gm.vn", entity.UserRole{RoleID: 2, BranchID: staffU(1)})
		return f
	}
	actions := map[string]func(f *staffFixture, ctx context.Context) error{
		"reset-password": func(f *staffFixture, ctx context.Context) error { return f.svc.ResetPassword(ctx, 5, "Takeover123") },
		"lock":           func(f *staffFixture, ctx context.Context) error { _, err := f.svc.Lock(ctx, 5); return err },
		"unlock":         func(f *staffFixture, ctx context.Context) error { _, err := f.svc.Unlock(ctx, 5); return err },
		"update-email": func(f *staffFixture, ctx context.Context) error {
			_, err := f.svc.Update(ctx, 5, v1.StaffInput{FullName: "QL2", Email: "attacker@gm.vn", Position: StaffPositionManager})
			return err
		},
		"delete": func(f *staffFixture, ctx context.Context) error { return f.svc.Delete(ctx, 5) },
		"assign-roles": func(f *staffFixture, ctx context.Context) error {
			_, err := f.svc.AssignRoles(ctx, 5, []v1.RoleAssignment{{RoleCode: "TRAINER", BranchID: staffU(1)}})
			return err
		},
	}
	for name, act := range actions {
		t.Run(name, func(t *testing.T) {
			f := newF()
			err := act(f, managerCtx())
			require.Error(t, err)
			assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
			assert.Equal(t, "ql2@gm.vn", f.repo.users[5].Email)
			assert.Empty(t, f.repo.users[5].PasswordHash)
			assert.Empty(t, f.audit.actions)

			// Vai trò khác (không đồng cấp) có đủ quyền của nhân viên đích → được phép.
			hr := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 2, Roles: []string{"HR_ADMIN"},
				Permissions: []string{"payroll.manage"}, BranchIDs: []uint{1}})
			require.NoError(t, act(newF(), hr))

			// Quản lý đồng cấp, dù có đủ quyền, cũng không được thao tác trên quản lý khác.
			peer := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 2, Roles: []string{consts.RoleManager},
				Permissions: []string{"payroll.manage"}, BranchIDs: []uint{1}})
			err = act(newF(), peer)
			assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
		})
	}
}

func TestStaffManage_BranchScopeAndSelf(t *testing.T) {
	t.Run("nhân viên có vai trò ở chi nhánh khác → không reset/khóa được, gán vai trò trong chi nhánh vẫn được", func(t *testing.T) {
		f := newStaffFixture()
		f.repo.users[3].UserRoles = append(f.repo.users[3].UserRoles,
			entity.UserRole{UserID: 3, RoleID: 4, BranchID: staffU(2), Role: staffRoleByID(4)})
		err := f.svc.ResetPassword(managerCtx(), 3, "Takeover123")
		require.Error(t, err)
		assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
		_, err = f.svc.Lock(managerCtx(), 3)
		assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
		_, err = f.svc.AssignRoles(managerCtx(), 3, []v1.RoleAssignment{{RoleCode: "TRAINER", BranchID: staffU(1)}})
		require.NoError(t, err)
		// OWNER không bị giới hạn.
		require.NoError(t, f.svc.ResetPassword(ownerCtx(), 3, "NewPass123"))
	})
	t.Run("không tự thao tác trên chính mình qua màn quản trị", func(t *testing.T) {
		f := newStaffFixture()
		err := f.svc.ResetPassword(managerCtx(), 2, "NewPass123")
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
		err = f.svc.ResetPassword(ownerCtx(), 1, "NewPass123")
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "OWNER cũng phải dùng đổi mật khẩu (cần mật khẩu cũ)")
		_, err = f.svc.Unlock(managerCtx(), 2)
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
		_, err = f.svc.Update(managerCtx(), 2, v1.StaffInput{FullName: "QL", Email: "manager@gm.vn", Position: StaffPositionManager})
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
		_, err = f.svc.Update(ownerCtx(), 1, v1.StaffInput{FullName: "Chủ", Email: "owner@gm.vn", Position: StaffPositionManager})
		require.NoError(t, err, "OWNER được tự sửa hồ sơ")
	})
	t.Run("mật khẩu mới phải có chữ và số", func(t *testing.T) {
		f := newStaffFixture()
		err := f.svc.ResetPassword(ownerCtx(), 3, "chiluchuthoi")
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
		req := newStaffReq("z@gm.vn", v1.RoleAssignment{RoleCode: "TRAINER", BranchID: staffU(1)})
		req.Password = "12345678"
		_, err = f.svc.Create(ownerCtx(), req)
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	})
}
