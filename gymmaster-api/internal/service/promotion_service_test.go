package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/promotion/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
)

type fakePromotionRepo struct {
	nextID uint
	items  map[uint]*entity.Promotion
}

func newFakePromotionRepo() *fakePromotionRepo {
	return &fakePromotionRepo{items: map[uint]*entity.Promotion{}}
}

func (f *fakePromotionRepo) List(context.Context, dto.Query, time.Time) ([]entity.Promotion, int64, error) {
	out := []entity.Promotion{}
	for _, p := range f.items {
		out = append(out, *p)
	}
	return out, int64(len(out)), nil
}
func (f *fakePromotionRepo) FindByID(_ context.Context, id uint) (*entity.Promotion, error) {
	if p, ok := f.items[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakePromotionRepo) FindByCode(_ context.Context, code string) (*entity.Promotion, error) {
	for _, p := range f.items {
		if strings.EqualFold(p.Code, code) {
			cp := *p
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakePromotionRepo) ExistsCode(_ context.Context, code string, excludeID uint) (bool, error) {
	for _, p := range f.items {
		if strings.EqualFold(p.Code, code) && p.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakePromotionRepo) Create(_ context.Context, p *entity.Promotion) error {
	f.nextID++
	p.ID = f.nextID
	cp := *p
	f.items[p.ID] = &cp
	return nil
}
func (f *fakePromotionRepo) Update(_ context.Context, p *entity.Promotion) error {
	cp := *p
	f.items[p.ID] = &cp
	return nil
}
func (f *fakePromotionRepo) Delete(_ context.Context, id uint, _ *uint) error {
	delete(f.items, id)
	return nil
}
func (f *fakePromotionRepo) IncrementUsage(_ context.Context, id uint) (int64, error) {
	p, ok := f.items[id]
	if !ok || (p.UsageLimit != nil && p.UsedCount >= *p.UsageLimit) {
		return 0, nil
	}
	p.UsedCount++
	return 1, nil
}

func catPtrF(v float64) *float64 { return &v }
func catPtrI(v int) *int         { return &v }

var promoNow = time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)

func basePromo() *entity.Promotion {
	return &entity.Promotion{
		Code: "GIAM10", Name: "Giảm 10%", Type: entity.PromotionPercent, Value: 10,
		AppliesTo: entity.AppliesAll, Status: "ACTIVE",
		StartAt: promoNow.Add(-24 * time.Hour), EndAt: promoNow.Add(24 * time.Hour),
	}
}

func TestCalcPromotionDiscount(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(p *entity.Promotion)
		amount float64
		want   float64
	}{
		{"phần trăm", nil, 1_000_000, 100_000},
		{"phần trăm làm tròn VND", func(p *entity.Promotion) { p.Value = 15 }, 333_333, 50_000},
		{"phần trăm bị giới hạn trần", func(p *entity.Promotion) { p.Value = 50; p.MaxDiscount = catPtrF(200_000) }, 1_000_000, 200_000},
		{"phần trăm dưới trần", func(p *entity.Promotion) { p.Value = 10; p.MaxDiscount = catPtrF(200_000) }, 1_000_000, 100_000},
		{"100%", func(p *entity.Promotion) { p.Value = 100 }, 500_000, 500_000},
		{"cố định", func(p *entity.Promotion) { p.Type = entity.PromotionFixed; p.Value = 150_000 }, 1_000_000, 150_000},
		{"cố định lớn hơn đơn", func(p *entity.Promotion) { p.Type = entity.PromotionFixed; p.Value = 150_000 }, 100_000, 100_000},
		{"đơn 0 đồng", nil, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := basePromo()
			if c.mutate != nil {
				c.mutate(p)
			}
			assert.Equal(t, c.want, CalcPromotionDiscount(p, c.amount))
		})
	}
}

func TestCheckPromotionApplicable(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(p *entity.Promotion)
		appliesTo string
		amount    float64
		wantMsg   string
	}{
		{"hợp lệ", nil, entity.AppliesMembership, 500_000, ""},
		{"ngừng áp dụng", func(p *entity.Promotion) { p.Status = "INACTIVE" }, entity.AppliesMembership, 500_000, "ngừng áp dụng"},
		{"chưa đến hạn", func(p *entity.Promotion) { p.StartAt = promoNow.Add(time.Hour) }, entity.AppliesMembership, 500_000, "chưa đến thời gian"},
		{"hết hạn", func(p *entity.Promotion) { p.EndAt = promoNow.Add(-time.Minute) }, entity.AppliesMembership, 500_000, "đã hết hạn"},
		{"đúng thời điểm kết thúc vẫn hợp lệ", func(p *entity.Promotion) { p.EndAt = promoNow }, entity.AppliesMembership, 500_000, ""},
		{"sai loại đơn", func(p *entity.Promotion) { p.AppliesTo = entity.AppliesPT }, entity.AppliesMembership, 500_000, "chỉ áp dụng cho gói PT"},
		{"khớp loại đơn", func(p *entity.Promotion) { p.AppliesTo = entity.AppliesPT }, entity.AppliesPT, 500_000, ""},
		{"chưa đạt tối thiểu", func(p *entity.Promotion) { p.MinAmount = 1_000_000 }, entity.AppliesMembership, 999_999, "tối thiểu 1.000.000 đ"},
		{"bằng tối thiểu", func(p *entity.Promotion) { p.MinAmount = 1_000_000 }, entity.AppliesMembership, 1_000_000, ""},
		{"hết lượt", func(p *entity.Promotion) { p.UsageLimit = catPtrI(5); p.UsedCount = 5 }, entity.AppliesMembership, 500_000, "hết lượt"},
		{"còn lượt", func(p *entity.Promotion) { p.UsageLimit = catPtrI(5); p.UsedCount = 4 }, entity.AppliesMembership, 500_000, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := basePromo()
			if c.mutate != nil {
				c.mutate(p)
			}
			err := CheckPromotionApplicable(p, c.appliesTo, c.amount, promoNow)
			if c.wantMsg == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
			assert.Contains(t, err.Error(), c.wantMsg)
		})
	}
}

func TestPromotionValidate(t *testing.T) {
	repo := newFakePromotionRepo()
	svc := NewPromotionService(repo, nil)
	ctx := context.Background()
	p := basePromo()
	p.MaxDiscount = catPtrF(50_000)
	require.NoError(t, repo.Create(ctx, p))

	got, discount, err := svc.Validate(ctx, " giam10 ", entity.AppliesMembership, 1_000_000, promoNow)
	require.NoError(t, err, "mã không phân biệt hoa thường, bỏ khoảng trắng")
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, 50_000.0, discount)

	_, _, err = svc.Validate(ctx, "KHONGCO", entity.AppliesMembership, 1_000_000, promoNow)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	assert.Contains(t, err.Error(), "không tồn tại")

	_, _, err = svc.Validate(ctx, "", entity.AppliesMembership, 1_000_000, promoNow)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	_, _, err = svc.Validate(ctx, "GIAM10", "ALL", 1_000_000, promoNow)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "loại đơn phải là MEMBERSHIP|PT|PRODUCT")

	_, _, err = svc.Validate(ctx, "GIAM10", entity.AppliesPT, 1_000_000, promoNow.Add(48*time.Hour))
	assert.Contains(t, err.Error(), "đã hết hạn")
}

func TestPromotionConsume(t *testing.T) {
	repo := newFakePromotionRepo()
	svc := NewPromotionService(repo, nil)
	ctx := context.Background()
	p := basePromo()
	p.UsageLimit = catPtrI(2)
	require.NoError(t, repo.Create(ctx, p))

	require.NoError(t, svc.Consume(ctx, p.ID))
	require.NoError(t, svc.Consume(ctx, p.ID))
	err := svc.Consume(ctx, p.ID)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	assert.Contains(t, err.Error(), "hết lượt")
	assert.Equal(t, 2, repo.items[p.ID].UsedCount)

	unlimited := basePromo()
	unlimited.Code = "VOHAN"
	require.NoError(t, repo.Create(ctx, unlimited))
	for range 5 {
		require.NoError(t, svc.Consume(ctx, unlimited.ID))
	}
}

func TestPromotionCheck_FinalAmount(t *testing.T) {
	repo := newFakePromotionRepo()
	svc := NewPromotionService(repo, nil).(*promotionService)
	svc.now = func() time.Time { return promoNow }
	p := basePromo()
	p.Type, p.Value = entity.PromotionFixed, 200_000
	require.NoError(t, repo.Create(context.Background(), p))

	res, err := svc.Check(context.Background(), "giam10", entity.AppliesProduct, 150_000)
	require.NoError(t, err)
	assert.Equal(t, 150_000.0, res.Discount)
	assert.Equal(t, 0.0, res.FinalAmount)
	assert.Equal(t, v1.ValidityActive, res.Promotion.Validity)
}

func validPromotionInput() v1.PromotionInput {
	return v1.PromotionInput{
		Code: "he2026", Name: "Hè 2026", Type: entity.PromotionPercent, Value: 20,
		StartAt: promoNow, EndAt: promoNow.Add(30 * 24 * time.Hour),
	}
}

func TestNormalizePromotionInput(t *testing.T) {
	in, err := normalizePromotionInput(validPromotionInput())
	require.NoError(t, err)
	assert.Equal(t, "HE2026", in.Code, "mã lưu IN HOA")
	assert.Equal(t, "ALL", in.AppliesTo)
	assert.Equal(t, "ACTIVE", in.Status)

	bad := []struct {
		name   string
		mutate func(in *v1.PromotionInput)
	}{
		{"phần trăm > 100", func(in *v1.PromotionInput) { in.Value = 101 }},
		{"giá trị 0", func(in *v1.PromotionInput) { in.Value = 0 }},
		{"kết thúc trước bắt đầu", func(in *v1.PromotionInput) { in.EndAt = in.StartAt.Add(-time.Hour) }},
		{"kết thúc bằng bắt đầu", func(in *v1.PromotionInput) { in.EndAt = in.StartAt }},
		{"trần giảm 0", func(in *v1.PromotionInput) { in.MaxDiscount = catPtrF(0) }},
		{"tối thiểu âm", func(in *v1.PromotionInput) { in.MinAmount = -1 }},
		{"giới hạn lượt 0", func(in *v1.PromotionInput) { in.UsageLimit = catPtrI(0) }},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			in := validPromotionInput()
			c.mutate(&in)
			_, err := normalizePromotionInput(in)
			assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
		})
	}

	fixed := validPromotionInput()
	fixed.Type, fixed.Value, fixed.MaxDiscount = entity.PromotionFixed, 150_000.4, catPtrF(10)
	out, err := normalizePromotionInput(fixed)
	require.NoError(t, err, "FIXED được phép > 100")
	assert.Nil(t, out.MaxDiscount, "FIXED bỏ trần giảm")
	assert.Equal(t, 150_000.0, out.Value, "làm tròn VND")
}

func TestPromotionCreateUpdateDelete(t *testing.T) {
	repo := newFakePromotionRepo()
	svc := NewPromotionService(repo, nil)
	ctx := context.Background()

	p, err := svc.Create(ctx, validPromotionInput())
	require.NoError(t, err)
	assert.Equal(t, "HE2026", p.Code)

	dup := validPromotionInput()
	dup.Code = "He2026"
	_, err = svc.Create(ctx, dup)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err), "trùng mã không phân biệt hoa thường")

	repo.items[p.ID].UsedCount = 3
	in := validPromotionInput()
	in.UsageLimit = catPtrI(2)
	_, err = svc.Update(ctx, p.ID, in)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "giới hạn < số lượt đã dùng")

	err = svc.Delete(ctx, p.ID)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err), "đã dùng thì không được xóa")

	repo.items[p.ID].UsedCount = 0
	require.NoError(t, svc.Delete(ctx, p.ID))
}

func TestPromotionValidity(t *testing.T) {
	p := basePromo()
	assert.Equal(t, v1.ValidityActive, PromotionValidity(*p, promoNow))
	assert.Equal(t, v1.ValidityUpcoming, PromotionValidity(*p, promoNow.Add(-48*time.Hour)))
	assert.Equal(t, v1.ValidityExpired, PromotionValidity(*p, promoNow.Add(48*time.Hour)))
	p.UsageLimit, p.UsedCount = catPtrI(1), 1
	assert.Equal(t, v1.ValidityExhausted, PromotionValidity(*p, promoNow))
	p.Status = "INACTIVE"
	assert.Equal(t, v1.ValidityInactive, PromotionValidity(*p, promoNow))
}
