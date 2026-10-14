package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	planv1 "gymmaster-api/api/plan/v1"
	ptv1 "gymmaster-api/api/ptpackage/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
)

// ---- gói tập ---------------------------------------------------------------

type fakePlanRepo struct {
	nextID uint
	items  map[uint]*entity.MembershipPlan
	subs   map[uint]int64
}

func newFakePlanRepo() *fakePlanRepo {
	return &fakePlanRepo{items: map[uint]*entity.MembershipPlan{}, subs: map[uint]int64{}}
}

func (f *fakePlanRepo) List(context.Context, dto.Query) ([]entity.MembershipPlan, int64, error) {
	return nil, 0, nil
}
func (f *fakePlanRepo) ListActive(context.Context) ([]entity.MembershipPlan, error) {
	out := []entity.MembershipPlan{}
	for _, p := range f.items {
		if p.Status == "ACTIVE" {
			out = append(out, *p)
		}
	}
	return out, nil
}
func (f *fakePlanRepo) FindByID(_ context.Context, id uint) (*entity.MembershipPlan, error) {
	if p, ok := f.items[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakePlanRepo) ExistsCode(_ context.Context, code string, excludeID uint) (bool, error) {
	for _, p := range f.items {
		if p.Code == code && p.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakePlanRepo) Create(_ context.Context, p *entity.MembershipPlan) error {
	f.nextID++
	p.ID = f.nextID
	cp := *p
	f.items[p.ID] = &cp
	return nil
}
func (f *fakePlanRepo) Update(_ context.Context, p *entity.MembershipPlan) error {
	cp := *p
	f.items[p.ID] = &cp
	return nil
}
func (f *fakePlanRepo) UpdateStatus(_ context.Context, id uint, status string, _ *uint) error {
	f.items[id].Status = status
	return nil
}
func (f *fakePlanRepo) Delete(_ context.Context, id uint, _ *uint) error {
	delete(f.items, id)
	return nil
}
func (f *fakePlanRepo) CountSubscriptions(_ context.Context, id uint) (int64, error) {
	return f.subs[id], nil
}

func timePlan(code string) planv1.PlanInput {
	return planv1.PlanInput{Code: code, Name: "Gói " + code, Type: "TIME", DurationDays: 30, Price: 500_000}
}

func TestNormalizePlanInput(t *testing.T) {
	in := timePlan("m1")
	in.SessionCount = catPtrI(12)
	in.Price = 499_999.6
	out, err := normalizePlanInput(in)
	require.NoError(t, err)
	assert.Equal(t, "M1", out.Code)
	assert.Nil(t, out.SessionCount, "gói TIME không có số buổi")
	assert.Equal(t, 500_000.0, out.Price, "làm tròn VND")
	assert.Equal(t, "SINGLE_BRANCH", out.AccessScope)
	assert.Equal(t, "ACTIVE", out.Status)

	sess := timePlan("S12")
	sess.Type, sess.SessionCount = "SESSION", catPtrI(12)
	out, err = normalizePlanInput(sess)
	require.NoError(t, err)
	assert.Equal(t, 12, *out.SessionCount)

	bad := []struct {
		name   string
		mutate func(in *planv1.PlanInput)
	}{
		{"SESSION thiếu số buổi", func(in *planv1.PlanInput) { in.Type = "SESSION" }},
		{"SESSION số buổi 0", func(in *planv1.PlanInput) { in.Type, in.SessionCount = "SESSION", catPtrI(0) }},
		{"loại sai", func(in *planv1.PlanInput) { in.Type = "X" }},
		{"thời hạn 0", func(in *planv1.PlanInput) { in.DurationDays = 0 }},
		{"giá âm", func(in *planv1.PlanInput) { in.Price = -1 }},
		{"phí chuyển nhượng âm", func(in *planv1.PlanInput) { in.TransferFee = -1 }},
		{"bảo lưu ngày > 0 nhưng lần = 0", func(in *planv1.PlanInput) { in.MaxFreezeDays = 10 }},
		{"bảo lưu vượt thời hạn", func(in *planv1.PlanInput) { in.MaxFreezeDays, in.MaxFreezeTimes = 31, 1 }},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			in := timePlan("M1")
			c.mutate(&in)
			_, err := normalizePlanInput(in)
			assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
		})
	}
}

func TestMembershipPlanCRUD(t *testing.T) {
	repo := newFakePlanRepo()
	svc := NewMembershipPlanService(repo, nil)
	ctx := context.Background()

	p, err := svc.Create(ctx, timePlan("M1"))
	require.NoError(t, err)

	_, err = svc.Create(ctx, timePlan("M1"))
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err), "trùng mã")

	_, err = svc.Update(ctx, p.ID, timePlan("M1"))
	require.NoError(t, err, "giữ nguyên mã của chính nó")

	_, err = svc.Get(ctx, 999)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))

	// Đã có đăng ký → không xóa được, chỉ được ngừng bán.
	repo.subs[p.ID] = 1
	err = svc.Delete(ctx, p.ID)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	updated, err := svc.UpdateStatus(ctx, p.ID, "INACTIVE")
	require.NoError(t, err)
	assert.Equal(t, "INACTIVE", updated.Status)
	opts, err := svc.Options(ctx)
	require.NoError(t, err)
	assert.Empty(t, opts, "options chỉ trả gói ACTIVE")

	repo.subs[p.ID] = 0
	require.NoError(t, svc.Delete(ctx, p.ID))
}

// ---- gói PT ----------------------------------------------------------------

type fakePTRepo struct {
	nextID    uint
	items     map[uint]*entity.PTPackage
	contracts map[uint]int64
}

func (f *fakePTRepo) List(context.Context, dto.Query) ([]entity.PTPackage, int64, error) {
	return nil, 0, nil
}
func (f *fakePTRepo) ListActive(context.Context) ([]entity.PTPackage, error) { return nil, nil }
func (f *fakePTRepo) FindByID(_ context.Context, id uint) (*entity.PTPackage, error) {
	if p, ok := f.items[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakePTRepo) ExistsCode(_ context.Context, code string, excludeID uint) (bool, error) {
	for _, p := range f.items {
		if p.Code == code && p.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakePTRepo) Create(_ context.Context, p *entity.PTPackage) error {
	f.nextID++
	p.ID = f.nextID
	cp := *p
	f.items[p.ID] = &cp
	return nil
}
func (f *fakePTRepo) Update(_ context.Context, p *entity.PTPackage) error {
	cp := *p
	f.items[p.ID] = &cp
	return nil
}
func (f *fakePTRepo) UpdateStatus(_ context.Context, id uint, status string, _ *uint) error {
	f.items[id].Status = status
	return nil
}
func (f *fakePTRepo) Delete(_ context.Context, id uint, _ *uint) error {
	delete(f.items, id)
	return nil
}
func (f *fakePTRepo) CountContracts(_ context.Context, id uint) (int64, error) {
	return f.contracts[id], nil
}

func TestPTPackage(t *testing.T) {
	repo := &fakePTRepo{items: map[uint]*entity.PTPackage{}, contracts: map[uint]int64{}}
	svc := NewPTPackageService(repo, nil)
	ctx := context.Background()

	in := ptv1.PTPackageInput{Code: "pt10", Name: "10 buổi", SessionCount: 10, ValidityDays: 60, Price: 3_000_000}
	p, err := svc.Create(ctx, in)
	require.NoError(t, err)
	assert.Equal(t, "PT10", p.Code)
	assert.Equal(t, 60, p.SessionMinutes, "mặc định 60 phút")
	assert.Equal(t, 300_000.0, p.PricePerSession)

	for name, mutate := range map[string]func(*ptv1.PTPackageInput){
		"số buổi 0":      func(in *ptv1.PTPackageInput) { in.SessionCount = 0 },
		"thời lượng 10":  func(in *ptv1.PTPackageInput) { in.SessionMinutes = 10 },
		"thời lượng 300": func(in *ptv1.PTPackageInput) { in.SessionMinutes = 300 },
		"thời hạn 0":     func(in *ptv1.PTPackageInput) { in.ValidityDays = 0 },
		"giá âm":         func(in *ptv1.PTPackageInput) { in.Price = -1 },
	} {
		bad := in
		bad.Code = "X1"
		mutate(&bad)
		_, err := svc.Create(ctx, bad)
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), name)
	}

	repo.contracts[p.ID] = 2
	assert.Equal(t, rescode.Conflict.Code(), codeOf(svc.Delete(ctx, p.ID)), "đã có hợp đồng thì không xóa")
}

// ---- cấu hình --------------------------------------------------------------

func TestValidateSettingValue(t *testing.T) {
	ok := []struct {
		key, in, want string
	}{
		{SettingCheckinQRTTLSeconds, "60", "60"},
		{SettingBookingAdvanceDays, "7.0", "7"},
		{SettingSubscriptionRemindBeforeDays, "[1, 7, 3, 7]", "[7,3,1]"},
	}
	for _, c := range ok {
		got, err := ValidateSettingValue(c.key, json.RawMessage(c.in))
		require.NoError(t, err, c.key)
		assert.JSONEq(t, c.want, string(got))
	}

	bad := []struct {
		key, in string
	}{
		{SettingBookingAdvanceDays, "0"},
		{SettingBookingAdvanceDays, "-3"},
		{SettingBookingAdvanceDays, "1.5"},
		{SettingBookingAdvanceDays, `"7"`},
		{SettingBookingAdvanceDays, "[7]"},
		{SettingCheckinQRTTLSeconds, "5"},
		{SettingSubscriptionRemindBeforeDays, "7"},
		{SettingSubscriptionRemindBeforeDays, "[]"},
		{SettingSubscriptionRemindBeforeDays, "[7, 0]"},
		{SettingSubscriptionRemindBeforeDays, "[1,2,3,4,5,6,7,8,9,10,11]"},
	}
	for _, c := range bad {
		_, err := ValidateSettingValue(c.key, json.RawMessage(c.in))
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), c.key+"="+c.in)
	}

	_, err := ValidateSettingValue("khong.ton.tai", json.RawMessage("1"))
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}
