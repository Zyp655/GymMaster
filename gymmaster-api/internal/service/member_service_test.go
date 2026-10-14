package service

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/member/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

// ---- fakes (in-memory) ------------------------------------------------------

type memberFakeRepo struct {
	nextID      uint
	members     map[uint]*entity.Member
	branches    map[uint]*entity.Branch
	openSubs    map[uint]int64
	lockedUsers map[uint]bool
	created     int
}

func newMemberFakeRepo() *memberFakeRepo {
	return &memberFakeRepo{
		members: map[uint]*entity.Member{},
		branches: map[uint]*entity.Branch{
			1: {BaseEntity: entity.BaseEntity{ID: 1}, Code: "HN01", Name: "Hà Nội"},
			2: {BaseEntity: entity.BaseEntity{ID: 2}, Code: "HCM01", Name: "Hồ Chí Minh"},
		},
		openSubs:    map[uint]int64{},
		lockedUsers: map[uint]bool{},
	}
}

func (f *memberFakeRepo) List(context.Context, dto.Query, repository.MemberListFilter) ([]repository.MemberRow, int64, error) {
	return nil, 0, nil
}
func (f *memberFakeRepo) Options(context.Context, string, []uint, int) ([]entity.Member, error) {
	return nil, nil
}
func (f *memberFakeRepo) FindByID(_ context.Context, id uint) (*entity.Member, error) {
	if m, ok := f.members[id]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *memberFakeRepo) ExistsPhone(_ context.Context, phone string, excludeID uint) (bool, error) {
	for _, m := range f.members {
		if m.Phone == phone && m.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}
func (f *memberFakeRepo) ExistingPhones(_ context.Context, phones []string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, p := range phones {
		if ok, _ := f.ExistsPhone(context.Background(), p, 0); ok {
			out[p] = true
		}
	}
	return out, nil
}
func (f *memberFakeRepo) Create(_ context.Context, m *entity.Member) error {
	f.nextID++
	m.ID = f.nextID
	m.MemberCode = fmt.Sprintf("GM%06d", m.ID)
	cp := *m
	f.members[m.ID] = &cp
	f.created++
	return nil
}
func (f *memberFakeRepo) Update(_ context.Context, m *entity.Member) error {
	cp := *m
	f.members[m.ID] = &cp
	return nil
}
func (f *memberFakeRepo) Delete(_ context.Context, id uint, _ *uint) error {
	delete(f.members, id)
	return nil
}
func (f *memberFakeRepo) SetUserID(_ context.Context, id, userID uint, _ *uint) error {
	f.members[id].UserID = &userID
	return nil
}
func (f *memberFakeRepo) SetQRSecret(_ context.Context, id uint, secret string, _ *uint) error {
	f.members[id].QRSecret = secret
	return nil
}
func (f *memberFakeRepo) FindBranch(_ context.Context, id uint) (*entity.Branch, error) {
	if b, ok := f.branches[id]; ok {
		return b, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *memberFakeRepo) ActiveSubscription(context.Context, uint) (*repository.MemberSubscription, error) {
	return nil, nil
}
func (f *memberFakeRepo) CountOpenSubscriptions(_ context.Context, id uint) (int64, error) {
	return f.openSubs[id], nil
}
func (f *memberFakeRepo) CheckinStats(context.Context, uint) (repository.MemberCheckinStats, error) {
	return repository.MemberCheckinStats{}, nil
}
func (f *memberFakeRepo) RoleIDByCode(context.Context, string) (uint, error) { return 5, nil }
func (f *memberFakeRepo) UpdatePortalUser(_ context.Context, userID uint, fields map[string]any) error {
	if fields["status"] == consts.StatusLocked {
		f.lockedUsers[userID] = true
	}
	return nil
}
func (f *memberFakeRepo) RevokeUserTokens(context.Context, uint, time.Time) error { return nil }

type memberFakeMetricRepo struct{}

func (memberFakeMetricRepo) ListByMember(context.Context, uint) ([]repository.BodyMetricRow, error) {
	return nil, nil
}
func (memberFakeMetricRepo) Latest(context.Context, uint) (*repository.BodyMetricRow, error) {
	return nil, nil
}
func (memberFakeMetricRepo) FindByID(context.Context, uint) (*repository.BodyMetricRow, error) {
	return nil, gorm.ErrRecordNotFound
}
func (memberFakeMetricRepo) Create(context.Context, *entity.BodyMetric) error { return nil }
func (memberFakeMetricRepo) Delete(context.Context, uint) error               { return nil }

type memberFakeUserRepo struct {
	nextID uint
	users  map[uint]*entity.User
	roles  []entity.UserRole
}

func (f *memberFakeUserRepo) FindByEmail(_ context.Context, email string) (*entity.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *memberFakeUserRepo) FindByID(_ context.Context, id uint) (*entity.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *memberFakeUserRepo) Create(_ context.Context, u *entity.User) error {
	f.nextID++
	u.ID = f.nextID
	f.users[u.ID] = u
	return nil
}
func (f *memberFakeUserRepo) AssignRole(_ context.Context, ur *entity.UserRole) error {
	f.roles = append(f.roles, *ur)
	return nil
}
func (f *memberFakeUserRepo) UpdateLastLogin(context.Context, uint, time.Time) error { return nil }
func (f *memberFakeUserRepo) UpdatePassword(context.Context, uint, string) error     { return nil }
func (f *memberFakeUserRepo) PermissionCodes(context.Context, []uint) ([]string, error) {
	return nil, nil
}

type memberFakeAudit struct{ actions []string }

func (a *memberFakeAudit) Log(_ context.Context, action, _ string, _ uint, _, _ any) {
	a.actions = append(a.actions, action)
}

// ---- helpers ----------------------------------------------------------------

// memberTestNow: 14/10/2026 10:00 giờ Việt Nam.
var memberTestNow = time.Date(2026, 10, 14, 3, 0, 0, 0, time.UTC)

type memberFixture struct {
	svc   *memberService
	repo  *memberFakeRepo
	users *memberFakeUserRepo
	audit *memberFakeAudit
}

func newMemberFixture() memberFixture {
	repo := newMemberFakeRepo()
	users := &memberFakeUserRepo{nextID: 100, users: map[uint]*entity.User{
		1: {BaseEntity: entity.BaseEntity{ID: 1}, Email: "admin@gm.vn", Status: consts.StatusActive},
	}}
	audit := &memberFakeAudit{}
	svc := NewMemberService(repo, memberFakeMetricRepo{}, users, noTx{}, audit).(*memberService)
	svc.now = func() time.Time { return memberTestNow }
	return memberFixture{svc: svc, repo: repo, users: users, audit: audit}
}

func memberOwnerCtx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
}

// memberStaffCtx: nhân viên chỉ thuộc chi nhánh 1.
func memberStaffCtx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{
		UserID: 2, Roles: []string{consts.RoleReceptionist}, BranchIDs: []uint{1},
		Permissions: []string{"member.view", "member.create", "member.update"},
	})
}

func validMember(phone string, branchID uint) v1.MemberInput {
	return v1.MemberInput{FullName: "Nguyễn Văn A", Phone: phone, HomeBranchID: branchID}
}

// ---- tests ------------------------------------------------------------------

func TestNormalizeVNPhone(t *testing.T) {
	cases := map[string]string{
		"0912345678":      "0912345678",
		"091 234 5678":    "0912345678",
		"0912.345.678":    "0912345678",
		"+84 912 345 678": "0912345678",
		"84912345678":     "0912345678",
		"0381234567":      "0381234567",
		"02438123456":     "02438123456", // máy bàn 11 số
	}
	for in, want := range cases {
		got, ok := NormalizeVNPhone(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, bad := range []string{"", "12345", "0112345678", "091234567", "09123456789", "abc0912345678", "0612345678"} {
		_, ok := NormalizeVNPhone(bad)
		assert.False(t, ok, bad)
	}
}

func TestMemberCreate_ValidatesPhoneAndDob(t *testing.T) {
	fx := newMemberFixture()
	ctx := memberOwnerCtx()

	in := validMember("+84 912 345 678", 1)
	in.Dob = "1995-08-15"
	in.Email = " An@Example.COM "
	m, err := fx.svc.Create(ctx, in)
	require.NoError(t, err)
	assert.Equal(t, "0912345678", m.Phone, "SĐT được chuẩn hóa")
	assert.Equal(t, "an@example.com", m.Email)
	assert.Equal(t, "1995-08-15", *m.Dob)
	assert.Equal(t, entity.MemberGenderOther, m.Gender, "mặc định OTHER")
	assert.Equal(t, entity.MemberSourceWalkIn, m.Source)
	assert.Equal(t, entity.MemberStatusActive, m.Status)
	assert.Equal(t, []string{AuditCreate}, fx.audit.actions)

	_, err = fx.svc.Create(ctx, validMember("12345", 1))
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "SĐT không hợp lệ")

	future := validMember("0987000111", 1)
	future.Dob = "2026-10-15" // ngày mai
	_, err = fx.svc.Create(ctx, future)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "ngày sinh ở tương lai")

	today := validMember("0987000111", 1)
	today.Dob = "2026-10-14"
	_, err = fx.svc.Create(ctx, today)
	assert.NoError(t, err, "ngày sinh = hôm nay vẫn hợp lệ")

	badEmergency := validMember("0987000222", 1)
	badEmergency.EmergencyContactPhone = "999"
	_, err = fx.svc.Create(ctx, badEmergency)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
}

func TestMemberCreate_DuplicatePhone(t *testing.T) {
	fx := newMemberFixture()
	ctx := memberOwnerCtx()
	a, err := fx.svc.Create(ctx, validMember("0912345678", 1))
	require.NoError(t, err)

	// Cùng SĐT dù khác cách viết → 409.
	_, err = fx.svc.Create(ctx, validMember("+84912345678", 2))
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	// Sửa chính hội viên đó, giữ SĐT cũ → hợp lệ.
	in := validMember("0912345678", 1)
	in.FullName = "Tên mới"
	updated, err := fx.svc.Update(ctx, a.ID, in)
	require.NoError(t, err)
	assert.Equal(t, "Tên mới", updated.FullName)

	// Sửa hội viên khác sang SĐT đã dùng → 409.
	b, err := fx.svc.Create(ctx, validMember("0987654321", 1))
	require.NoError(t, err)
	_, err = fx.svc.Update(ctx, b.ID, validMember("0912345678", 1))
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
}

func TestMember_BranchScope(t *testing.T) {
	fx := newMemberFixture()
	staff := memberStaffCtx()

	// Chi nhánh không tồn tại.
	_, err := fx.svc.Create(memberOwnerCtx(), validMember("0900000001", 99))
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// Nhân viên chi nhánh 1 không được tạo hội viên cho chi nhánh 2.
	_, err = fx.svc.Create(staff, validMember("0900000002", 2))
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	mine, err := fx.svc.Create(staff, validMember("0900000003", 1))
	require.NoError(t, err)
	other, err := fx.svc.Create(memberOwnerCtx(), validMember("0900000004", 2))
	require.NoError(t, err)

	// Xem / sửa / xóa hội viên chi nhánh khác → 403.
	_, err = fx.svc.Get(staff, other.ID)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
	_, err = fx.svc.Update(staff, other.ID, validMember("0900000004", 2))
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(fx.svc.Delete(staff, other.ID)))
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(fx.svc.ResetQR(staff, other.ID)))

	// Chuyển hội viên của mình sang chi nhánh ngoài phạm vi → 403.
	_, err = fx.svc.Update(staff, mine.ID, validMember("0900000003", 2))
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	// Lọc danh sách theo chi nhánh ngoài phạm vi → 403.
	_, err = fx.svc.List(staff, &v1.ListReq{PageReq: dto.PageReq{Page: 1, Limit: 10}, BranchID: 2})
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
	_, err = fx.svc.Options(staff, "", 2)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	// Không tìm thấy.
	_, err = fx.svc.Get(staff, 999)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}

func TestMemberDelete_BlockedWhenHasOpenSubscription(t *testing.T) {
	fx := newMemberFixture()
	ctx := memberOwnerCtx()
	m, err := fx.svc.Create(ctx, validMember("0912345678", 1))
	require.NoError(t, err)

	fx.repo.openSubs[m.ID] = 1
	assert.Equal(t, rescode.Conflict.Code(), codeOf(fx.svc.Delete(ctx, m.ID)))
	assert.Contains(t, fx.repo.members, m.ID)

	fx.repo.openSubs[m.ID] = 0
	uid := uint(55)
	fx.repo.members[m.ID].UserID = &uid
	require.NoError(t, fx.svc.Delete(ctx, m.ID))
	assert.NotContains(t, fx.repo.members, m.ID)
	assert.True(t, fx.repo.lockedUsers[uid], "xóa hội viên thì khóa tài khoản cổng")
	assert.Contains(t, fx.audit.actions, AuditDelete)
}

func TestMemberPortalAccount(t *testing.T) {
	fx := newMemberFixture()
	ctx := memberOwnerCtx()
	m, err := fx.svc.Create(ctx, validMember("0912345678", 1))
	require.NoError(t, err)

	// Email trùng tài khoản khác → 409.
	_, err = fx.svc.CreatePortalAccount(ctx, m.ID, "admin@gm.vn", "Member@123")
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	acc, err := fx.svc.CreatePortalAccount(ctx, m.ID, "Hv@GM.vn", "Member@123")
	require.NoError(t, err)
	assert.Equal(t, "hv@gm.vn", acc.Email)
	require.NotNil(t, fx.repo.members[m.ID].UserID)
	assert.Equal(t, acc.UserID, *fx.repo.members[m.ID].UserID)
	require.Len(t, fx.users.roles, 1)
	assert.Nil(t, fx.users.roles[0].BranchID, "role MEMBER không gắn chi nhánh")

	// Đã có tài khoản đang hoạt động → 409.
	_, err = fx.svc.CreatePortalAccount(ctx, m.ID, "khac@gm.vn", "Member@123")
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	require.NoError(t, fx.svc.LockPortalAccount(ctx, m.ID))
	assert.True(t, fx.repo.lockedUsers[acc.UserID])

	// Hội viên chưa có tài khoản → khóa trả 404.
	m2, err := fx.svc.Create(ctx, validMember("0987654321", 1))
	require.NoError(t, err)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(fx.svc.LockPortalAccount(ctx, m2.ID)))
}

func TestMemberResetQR(t *testing.T) {
	fx := newMemberFixture()
	ctx := memberOwnerCtx()
	m, err := fx.svc.Create(ctx, validMember("0912345678", 1))
	require.NoError(t, err)
	fx.repo.members[m.ID].QRSecret = "old"
	require.NoError(t, fx.svc.ResetQR(ctx, m.ID))
	assert.Len(t, fx.repo.members[m.ID].QRSecret, 64)
	assert.NotEqual(t, "old", fx.repo.members[m.ID].QRSecret)
}

func TestParseMemberImportRow(t *testing.T) {
	now := memberTestNow
	row, err := parseMemberImportRow([]string{" Trần Thị B ", "912345678", "B@Example.com", "Nữ", "20/05/1998", "001198012345", "Hà Nội", "Giới thiệu"}, now)
	require.NoError(t, err)
	assert.Equal(t, "Trần Thị B", row.FullName)
	assert.Equal(t, "0912345678", row.Phone, "ô số mất số 0 đầu được bổ sung")
	assert.Equal(t, "b@example.com", row.Email)
	assert.Equal(t, entity.MemberGenderFemale, row.Gender)
	assert.Equal(t, "1998-05-20", row.Dob.Format(consts.DateLayout))
	assert.Equal(t, entity.MemberSourceReferral, row.Source)

	// Dòng tối thiểu: chỉ họ tên + SĐT, các cột sau thiếu hẳn.
	row, err = parseMemberImportRow([]string{"Lê C", "0987654321"}, now)
	require.NoError(t, err)
	assert.Equal(t, entity.MemberGenderOther, row.Gender)
	assert.Equal(t, entity.MemberSourceWalkIn, row.Source)
	assert.Nil(t, row.Dob)

	// Ngày sinh dạng số serial của Excel (35935 = 20/05/1998).
	row, err = parseMemberImportRow([]string{"D", "0987654321", "", "nam", "35935"}, now)
	require.NoError(t, err)
	assert.Equal(t, "1998-05-20", row.Dob.Format(consts.DateLayout))
	assert.Equal(t, entity.MemberGenderMale, row.Gender)

	bad := map[string][]string{
		"thiếu họ tên":     {"", "0912345678"},
		"thiếu SĐT":        {"A", ""},
		"SĐT sai":          {"A", "12345"},
		"email sai":        {"A", "0912345678", "abc@"},
		"giới tính sai":    {"A", "0912345678", "", "XYZ"},
		"ngày không có":    {"A", "0912345678", "", "", "31/02/2000"},
		"ngày tương lai":   {"A", "0912345678", "", "", "01/01/2099"},
		"CCCD sai":         {"A", "0912345678", "", "", "", "12AB"},
		"nguồn không biết": {"A", "0912345678", "", "", "", "", "", "TikTok"},
	}
	for name, cells := range bad {
		_, err := parseMemberImportRow(cells, now)
		assert.Error(t, err, name)
	}
}

func buildMemberXLSX(t *testing.T, rows [][]any) *bytes.Reader {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	require.NoError(t, f.SetSheetRow("Sheet1", "A1", &[]any{"Họ tên*", "SĐT*"}))
	for i, r := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		require.NoError(t, f.SetSheetRow("Sheet1", cell, &r))
	}
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))
	return bytes.NewReader(buf.Bytes())
}

func TestMemberImport(t *testing.T) {
	fx := newMemberFixture()
	ctx := memberOwnerCtx()
	_, err := fx.svc.Create(ctx, validMember("0900000000", 1))
	require.NoError(t, err)
	fx.repo.created = 0

	file := buildMemberXLSX(t, [][]any{
		{"Hợp lệ 1", "0911111111", "", "Nam", "01/01/1990"},
		{"Hợp lệ 2", 922222222}, // ô số, mất số 0 đầu
		{"Trùng DB", "0900000000"},
		{"Trùng file", "0911111111"},
		{}, // dòng trống – bỏ qua im lặng
		{"SĐT sai", "123"},
		{"", "0933333333"}, // thiếu họ tên
	})
	res, err := fx.svc.Import(ctx, 1, file)
	require.NoError(t, err)
	assert.Equal(t, 2, res.Created)
	assert.Equal(t, 4, res.Skipped)
	assert.Equal(t, 2, fx.repo.created)
	rows := []int{}
	for _, e := range res.Errors {
		rows = append(rows, e.Row)
	}
	assert.Equal(t, []int{4, 5, 7, 8}, rows, "số dòng Excel, tăng dần")
	assert.Contains(t, res.Errors[0].Message, "đã tồn tại")
	assert.Contains(t, res.Errors[1].Message, "trùng với dòng 2")
	assert.Contains(t, fx.audit.actions, AuditImport)

	// Chi nhánh ngoài phạm vi.
	_, err = fx.svc.Import(memberStaffCtx(), 2, buildMemberXLSX(t, [][]any{{"A", "0944444444"}}))
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	// File không phải Excel / không có dữ liệu.
	_, err = fx.svc.Import(ctx, 1, bytes.NewReader([]byte("not excel")))
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	_, err = fx.svc.Import(ctx, 1, buildMemberXLSX(t, nil))
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
}

func TestMemberImportTemplate(t *testing.T) {
	fx := newMemberFixture()
	data, err := fx.svc.ImportTemplate()
	require.NoError(t, err)
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	rows, err := f.GetRows(memberImportSheet)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, memberImportHeaders, rows[0])
}

func TestUploadService(t *testing.T) {
	dir := t.TempDir()
	svc := NewUploadService(dir, "http://x/uploads/", 1).(*uploadService)
	svc.now = func() time.Time { return memberTestNow }
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00")

	url, err := svc.SaveImage(memberOwnerCtx(), int64(len(png)), bytes.NewReader(png))
	require.NoError(t, err)
	assert.Regexp(t, `^http://x/uploads/2026/10/[0-9a-f-]{36}\.png$`, url)

	_, err = svc.SaveImage(memberOwnerCtx(), 10, bytes.NewReader([]byte("<?php echo 1;")))
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "nội dung không phải ảnh")

	big := append(append([]byte{}, png...), make([]byte, 1<<20)...)
	_, err = svc.SaveImage(memberOwnerCtx(), 100, bytes.NewReader(big))
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "vượt dung lượng dù khai báo size nhỏ")

	memberOnly := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 9, Roles: []string{consts.RoleMember}})
	_, err = svc.SaveImage(memberOnly, int64(len(png)), bytes.NewReader(png))
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
}
