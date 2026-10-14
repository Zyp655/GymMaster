package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/promotion/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

type (
	// PromotionService quản lý mã khuyến mãi và cung cấp nghiệp vụ áp mã cho các module bán hàng
	// (bán gói tập, bán PT, POS).
	//
	// Cách dùng trong module bán hàng:
	//
	//	promo, discount, err := promotions.Validate(ctx, req.PromotionCode, entity.AppliesMembership, subtotal, time.Now())
	//	if err != nil { return err }                 // lỗi 400 với thông điệp tiếng Việt
	//	err = tx.WithTx(ctx, func(ctx context.Context) error {
	//	    ... tạo invoice (promotion_id = promo.ID, discount = discount) ...
	//	    return promotions.Consume(ctx, promo.ID) // tăng used_count nguyên tử trong cùng transaction
	//	})
	PromotionService interface {
		List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error)
		Get(ctx context.Context, id uint) (*v1.Promotion, error)
		Create(ctx context.Context, in v1.PromotionInput) (*v1.Promotion, error)
		Update(ctx context.Context, id uint, in v1.PromotionInput) (*v1.Promotion, error)
		Delete(ctx context.Context, id uint) error
		// Check dùng cho màn hình bán hàng: kiểm tra mã và trả số tiền giảm, số tiền còn phải trả.
		Check(ctx context.Context, code, appliesTo string, amount float64) (*v1.CheckRes, error)

		// Validate kiểm tra mã khuyến mãi và tính số tiền giảm cho một đơn có giá trị amount
		// thuộc loại appliesTo (MEMBERSHIP|PT|PRODUCT). Số tiền giảm đã làm tròn VND và không vượt amount.
		// Mọi lỗi nghiệp vụ trả rescode.BadRequest với thông điệp tiếng Việt.
		Validate(ctx context.Context, code string, appliesTo string, amount float64, now time.Time) (*entity.Promotion, float64, error)
		// Consume tăng used_count nguyên tử (chỉ khi còn lượt). Gọi BÊN TRONG transaction bán hàng.
		// Hết lượt → rescode.BadRequest "Mã khuyến mãi đã hết lượt sử dụng".
		Consume(ctx context.Context, promotionID uint) error
	}

	promotionService struct {
		repo  repository.PromotionRepository
		audit AuditService
		now   func() time.Time
	}
)

// NewPromotionService tạo service khuyến mãi. audit có thể nil.
// Module khác dùng: service.NewPromotionService(repository.NewPromotionRepository(d.DB), d.Audit).
func NewPromotionService(repo repository.PromotionRepository, audit AuditService) PromotionService {
	return &promotionService{repo: repo, audit: audit, now: time.Now}
}

const auditEntityPromotion = "promotion"

var promotionOrderable = []string{"id", "code", "name", "type", "value", "start_at", "end_at", "used_count", "status", "created_at"}

func (s *promotionService) List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error) {
	q := req.PageReq.ToQuery(promotionOrderable, "id")
	if req.Status != "" {
		q.Filters["status"] = req.Status
	}
	if req.AppliesTo != "" {
		q.Filters["applies_to"] = req.AppliesTo
	}
	if req.Validity != "" {
		q.Filters["validity"] = req.Validity
	}
	now := s.now()
	items, total, err := s.repo.List(ctx, q, now)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(items, func(p entity.Promotion) *v1.Promotion { return toPromotionRes(p, now) }), total, q), nil
}

func (s *promotionService) Get(ctx context.Context, id uint) (*v1.Promotion, error) {
	p, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	return toPromotionRes(*p, s.now()), nil
}

func (s *promotionService) Create(ctx context.Context, in v1.PromotionInput) (*v1.Promotion, error) {
	in, err := normalizePromotionInput(in)
	if err != nil {
		return nil, err
	}
	if err := s.checkCode(ctx, in.Code, 0); err != nil {
		return nil, err
	}
	p := &entity.Promotion{}
	applyPromotionInput(p, in)
	p.CreatedBy = reqctx.UserID(ctx)
	p.UpdatedBy = p.CreatedBy
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, mapDuplicate(err, "Mã khuyến mãi đã tồn tại")
	}
	res := toPromotionRes(*p, s.now())
	auditCatalog(ctx, s.audit, AuditCreate, auditEntityPromotion, p.ID, nil, res)
	return res, nil
}

func (s *promotionService) Update(ctx context.Context, id uint, in v1.PromotionInput) (*v1.Promotion, error) {
	p, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	in, err = normalizePromotionInput(in)
	if err != nil {
		return nil, err
	}
	if in.UsageLimit != nil && *in.UsageLimit < p.UsedCount {
		return nil, rescode.New(rescode.BadRequest, fmt.Sprintf("Giới hạn lượt dùng không được nhỏ hơn số lượt đã dùng (%d)", p.UsedCount))
	}
	if err := s.checkCode(ctx, in.Code, id); err != nil {
		return nil, err
	}
	before := toPromotionRes(*p, s.now())
	applyPromotionInput(p, in)
	p.UpdatedBy = reqctx.UserID(ctx)
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, mapDuplicate(err, "Mã khuyến mãi đã tồn tại")
	}
	res, err := s.Get(ctx, id)
	if err == nil {
		auditCatalog(ctx, s.audit, AuditUpdate, auditEntityPromotion, id, before, res)
	}
	return res, err
}

func (s *promotionService) Delete(ctx context.Context, id uint) error {
	p, err := s.find(ctx, id)
	if err != nil {
		return err
	}
	if p.UsedCount > 0 {
		return rescode.New(rescode.Conflict, "Mã khuyến mãi đã được sử dụng nên không thể xóa, hãy chuyển sang trạng thái Ngừng áp dụng")
	}
	if err := s.repo.Delete(ctx, id, reqctx.UserID(ctx)); err != nil {
		return err
	}
	auditCatalog(ctx, s.audit, AuditDelete, auditEntityPromotion, id, toPromotionRes(*p, s.now()), nil)
	return nil
}

func (s *promotionService) Check(ctx context.Context, code, appliesTo string, amount float64) (*v1.CheckRes, error) {
	now := s.now()
	p, discount, err := s.Validate(ctx, code, appliesTo, amount, now)
	if err != nil {
		return nil, err
	}
	return &v1.CheckRes{
		Promotion:   toPromotionRes(*p, now),
		Discount:    discount,
		FinalAmount: roundMoneyVND(amount) - discount,
	}, nil
}

func (s *promotionService) Validate(ctx context.Context, code string, appliesTo string, amount float64, now time.Time) (*entity.Promotion, float64, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, 0, rescode.New(rescode.BadRequest, "Vui lòng nhập mã khuyến mãi")
	}
	switch appliesTo {
	case entity.AppliesMembership, entity.AppliesPT, entity.AppliesProduct:
	default:
		return nil, 0, rescode.New(rescode.BadRequest, "Loại đơn áp dụng khuyến mãi không hợp lệ")
	}
	if amount < 0 {
		return nil, 0, rescode.New(rescode.BadRequest, "Giá trị đơn không được âm")
	}
	p, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, rescode.New(rescode.BadRequest, "Mã khuyến mãi không tồn tại")
		}
		return nil, 0, err
	}
	if err := CheckPromotionApplicable(p, appliesTo, amount, now); err != nil {
		return nil, 0, err
	}
	return p, CalcPromotionDiscount(p, amount), nil
}

func (s *promotionService) Consume(ctx context.Context, promotionID uint) error {
	n, err := s.repo.IncrementUsage(ctx, promotionID)
	if err != nil {
		return err
	}
	if n == 0 {
		return rescode.New(rescode.BadRequest, "Mã khuyến mãi đã hết lượt sử dụng")
	}
	return nil
}

func (s *promotionService) find(ctx context.Context, id uint) (*entity.Promotion, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy khuyến mãi")
		}
		return nil, err
	}
	return p, nil
}

func (s *promotionService) checkCode(ctx context.Context, code string, excludeID uint) error {
	exists, err := s.repo.ExistsCode(ctx, code, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return rescode.New(rescode.Conflict, "Mã khuyến mãi đã tồn tại")
	}
	return nil
}

// CheckPromotionApplicable kiểm tra mã có áp dụng được cho đơn (loại appliesTo, giá trị amount) tại thời điểm now.
// Thứ tự kiểm tra: trạng thái → thời gian → loại đơn → giá trị tối thiểu → số lượt.
func CheckPromotionApplicable(p *entity.Promotion, appliesTo string, amount float64, now time.Time) error {
	switch {
	case p.Status != consts.StatusActive:
		return rescode.New(rescode.BadRequest, "Mã khuyến mãi đã ngừng áp dụng")
	case now.Before(p.StartAt):
		return rescode.New(rescode.BadRequest, "Mã khuyến mãi chưa đến thời gian áp dụng")
	case now.After(p.EndAt):
		return rescode.New(rescode.BadRequest, "Mã khuyến mãi đã hết hạn")
	case p.AppliesTo != entity.AppliesAll && p.AppliesTo != appliesTo:
		return rescode.New(rescode.BadRequest, "Mã khuyến mãi chỉ áp dụng cho "+appliesToLabel(p.AppliesTo))
	case amount < p.MinAmount:
		return rescode.New(rescode.BadRequest, "Đơn hàng chưa đạt giá trị tối thiểu "+formatMoneyVND(p.MinAmount)+" để dùng mã này")
	case p.UsageLimit != nil && p.UsedCount >= *p.UsageLimit:
		return rescode.New(rescode.BadRequest, "Mã khuyến mãi đã hết lượt sử dụng")
	}
	return nil
}

// CalcPromotionDiscount tính số tiền giảm (làm tròn VND):
//   - PERCENT: amount × value / 100, không vượt max_discount (nếu có);
//   - FIXED  : min(value, amount).
//
// Kết quả luôn nằm trong [0, amount].
func CalcPromotionDiscount(p *entity.Promotion, amount float64) float64 {
	amount = roundMoneyVND(amount)
	if amount <= 0 {
		return 0
	}
	var d float64
	switch p.Type {
	case entity.PromotionPercent:
		d = roundMoneyVND(amount * p.Value / 100)
		if p.MaxDiscount != nil && d > *p.MaxDiscount {
			d = roundMoneyVND(*p.MaxDiscount)
		}
	case entity.PromotionFixed:
		d = roundMoneyVND(p.Value)
	}
	return math.Max(0, math.Min(d, amount))
}

// PromotionValidity trả trạng thái hiệu lực của mã tại thời điểm now
// (inactive | upcoming | expired | exhausted | active).
func PromotionValidity(p entity.Promotion, now time.Time) string {
	switch {
	case p.Status != consts.StatusActive:
		return v1.ValidityInactive
	case now.Before(p.StartAt):
		return v1.ValidityUpcoming
	case now.After(p.EndAt):
		return v1.ValidityExpired
	case p.UsageLimit != nil && p.UsedCount >= *p.UsageLimit:
		return v1.ValidityExhausted
	}
	return v1.ValidityActive
}

// normalizePromotionInput kiểm tra ràng buộc (khớp CHECK của migration 000004) và chuẩn hóa:
// mã IN HOA, max_discount chỉ dùng cho PERCENT, tiền làm tròn VND.
func normalizePromotionInput(in v1.PromotionInput) (v1.PromotionInput, error) {
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	if in.Status == "" {
		in.Status = consts.StatusActive
	}
	if in.AppliesTo == "" {
		in.AppliesTo = entity.AppliesAll
	}
	if in.Value <= 0 {
		return in, rescode.New(rescode.BadRequest, "Giá trị giảm phải lớn hơn 0")
	}
	switch in.Type {
	case entity.PromotionPercent:
		if in.Value > 100 {
			return in, rescode.New(rescode.BadRequest, "Phần trăm giảm không được vượt quá 100%")
		}
		if in.MaxDiscount != nil {
			if *in.MaxDiscount <= 0 {
				return in, rescode.New(rescode.BadRequest, "Mức giảm tối đa phải lớn hơn 0")
			}
			v := roundMoneyVND(*in.MaxDiscount)
			in.MaxDiscount = &v
		}
	case entity.PromotionFixed:
		in.Value = roundMoneyVND(in.Value)
		in.MaxDiscount = nil
	default:
		return in, rescode.New(rescode.BadRequest, "Loại giảm giá không hợp lệ")
	}
	if in.MinAmount < 0 {
		return in, rescode.New(rescode.BadRequest, "Giá trị đơn tối thiểu không được âm")
	}
	in.MinAmount = roundMoneyVND(in.MinAmount)
	if in.StartAt.IsZero() || in.EndAt.IsZero() {
		return in, rescode.New(rescode.BadRequest, "Vui lòng chọn thời gian áp dụng")
	}
	if !in.EndAt.After(in.StartAt) {
		return in, rescode.New(rescode.BadRequest, "Thời gian kết thúc phải sau thời gian bắt đầu")
	}
	if in.UsageLimit != nil && *in.UsageLimit <= 0 {
		return in, rescode.New(rescode.BadRequest, "Giới hạn lượt dùng phải lớn hơn 0")
	}
	return in, nil
}

func applyPromotionInput(p *entity.Promotion, in v1.PromotionInput) {
	p.Code = in.Code
	p.Name = in.Name
	p.Type = in.Type
	p.Value = in.Value
	p.MaxDiscount = in.MaxDiscount
	p.MinAmount = in.MinAmount
	p.AppliesTo = in.AppliesTo
	p.StartAt = in.StartAt
	p.EndAt = in.EndAt
	p.UsageLimit = in.UsageLimit
	p.Status = in.Status
}

func toPromotionRes(p entity.Promotion, now time.Time) *v1.Promotion {
	return &v1.Promotion{
		ID:          p.ID,
		Code:        p.Code,
		Name:        p.Name,
		Type:        p.Type,
		Value:       p.Value,
		MaxDiscount: p.MaxDiscount,
		MinAmount:   p.MinAmount,
		AppliesTo:   p.AppliesTo,
		StartAt:     p.StartAt,
		EndAt:       p.EndAt,
		UsageLimit:  p.UsageLimit,
		UsedCount:   p.UsedCount,
		Status:      p.Status,
		Validity:    PromotionValidity(p, now),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func appliesToLabel(v string) string {
	switch v {
	case entity.AppliesMembership:
		return "gói tập"
	case entity.AppliesPT:
		return "gói PT"
	case entity.AppliesProduct:
		return "sản phẩm"
	}
	return "mọi loại đơn"
}

// formatMoneyVND định dạng số tiền kiểu 1.500.000 đ cho thông điệp lỗi.
func formatMoneyVND(v float64) string {
	s := fmt.Sprintf("%d", int64(math.Round(v)))
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String() + " đ"
	}
	return b.String() + " đ"
}
