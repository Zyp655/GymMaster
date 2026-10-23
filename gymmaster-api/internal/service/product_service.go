package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/product/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

const (
	auditEntityProduct         = "product"
	auditEntityProductCategory = "product_category"
	auditEntitySupplier        = "supplier"
	AuditPriceChange           = "PRICE_CHANGE"
)

type (
	// ProductService: danh mục, nhà cung cấp, sản phẩm (perm product.view / product.manage).
	ProductService interface {
		ListCategories(ctx context.Context, keyword string) ([]*v1.Category, error)
		CreateCategory(ctx context.Context, in v1.CategoryInput) (*v1.Category, error)
		UpdateCategory(ctx context.Context, id uint, in v1.CategoryInput) (*v1.Category, error)
		DeleteCategory(ctx context.Context, id uint) error

		ListSuppliers(ctx context.Context, req *v1.SupplierListReq) (*v1.SupplierListRes, error)
		SupplierOptions(ctx context.Context) ([]*v1.SupplierOption, error)
		GetSupplier(ctx context.Context, id uint) (*v1.Supplier, error)
		CreateSupplier(ctx context.Context, in v1.SupplierInput) (*v1.Supplier, error)
		UpdateSupplier(ctx context.Context, id uint, in v1.SupplierInput) (*v1.Supplier, error)
		DeleteSupplier(ctx context.Context, id uint) error

		List(ctx context.Context, req *v1.ProductListReq) (*v1.ProductListRes, error)
		Get(ctx context.Context, id uint) (*v1.ProductDetail, error)
		Create(ctx context.Context, in v1.ProductInput) (*v1.ProductDetail, error)
		Update(ctx context.Context, id uint, in v1.ProductInput) (*v1.ProductDetail, error)
		Delete(ctx context.Context, id uint) error
	}

	productService struct {
		repo  repository.ProductRepository
		audit AuditService
	}
)

func NewProductService(repo repository.ProductRepository, audit AuditService) ProductService {
	return &productService{repo: repo, audit: audit}
}

var (
	errCategoryNotFound = rescode.New(rescode.NotFound, "Không tìm thấy danh mục sản phẩm")
	errSupplierNotFound = rescode.New(rescode.NotFound, "Không tìm thấy nhà cung cấp")
	errProductNotFound  = rescode.New(rescode.NotFound, "Không tìm thấy sản phẩm")
)

// ---- Danh mục -------------------------------------------------------------------

func toCategoryRes(r repository.CategoryRow) *v1.Category {
	return &v1.Category{ID: r.ID, Name: r.Name, SortOrder: r.SortOrder, ProductCount: r.ProductCount,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *productService) ListCategories(ctx context.Context, keyword string) ([]*v1.Category, error) {
	rows, err := s.repo.ListCategories(ctx, keyword)
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toCategoryRes), nil
}

func (s *productService) findCategory(ctx context.Context, id uint) (*entity.ProductCategory, error) {
	c, err := s.repo.FindCategory(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errCategoryNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *productService) checkCategoryName(ctx context.Context, name string, excludeID uint) error {
	exists, err := s.repo.ExistsCategoryName(ctx, name, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return rescode.New(rescode.Conflict, "Tên danh mục đã tồn tại")
	}
	return nil
}

func (s *productService) CreateCategory(ctx context.Context, in v1.CategoryInput) (*v1.Category, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên danh mục")
	}
	if err := s.checkCategoryName(ctx, name, 0); err != nil {
		return nil, err
	}
	c := &entity.ProductCategory{Name: name, SortOrder: in.SortOrder}
	if err := s.repo.CreateCategory(ctx, c); err != nil {
		return nil, mapDuplicate(err, "Tên danh mục đã tồn tại")
	}
	auditCatalog(ctx, s.audit, AuditCreate, auditEntityProductCategory, c.ID, nil, c)
	return toCategoryRes(repository.CategoryRow{ProductCategory: *c}), nil
}

func (s *productService) UpdateCategory(ctx context.Context, id uint, in v1.CategoryInput) (*v1.Category, error) {
	c, err := s.findCategory(ctx, id)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên danh mục")
	}
	if err := s.checkCategoryName(ctx, name, id); err != nil {
		return nil, err
	}
	before := *c
	c.Name, c.SortOrder = name, in.SortOrder
	if err := s.repo.UpdateCategory(ctx, c); err != nil {
		return nil, mapDuplicate(err, "Tên danh mục đã tồn tại")
	}
	auditCatalog(ctx, s.audit, AuditUpdate, auditEntityProductCategory, c.ID, before, c)
	n, _ := s.repo.CountProductsInCategory(ctx, id)
	return toCategoryRes(repository.CategoryRow{ProductCategory: *c, ProductCount: n}), nil
}

func (s *productService) DeleteCategory(ctx context.Context, id uint) error {
	c, err := s.findCategory(ctx, id)
	if err != nil {
		return err
	}
	n, err := s.repo.CountProductsInCategory(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return rescode.New(rescode.Conflict, fmt.Sprintf("Danh mục đang có %d sản phẩm, hãy chuyển hoặc xóa sản phẩm trước", n))
	}
	if err := s.repo.DeleteCategory(ctx, id); err != nil {
		return err
	}
	auditCatalog(ctx, s.audit, AuditDelete, auditEntityProductCategory, id, c, nil)
	return nil
}

// ---- Nhà cung cấp ---------------------------------------------------------------

func toSupplierRes(sp entity.Supplier) *v1.Supplier {
	return &v1.Supplier{ID: sp.ID, Name: sp.Name, Phone: sp.Phone, Email: sp.Email, Address: sp.Address, Note: sp.Note,
		CreatedAt: sp.CreatedAt, UpdatedAt: sp.UpdatedAt}
}

func (s *productService) ListSuppliers(ctx context.Context, req *v1.SupplierListReq) (*v1.SupplierListRes, error) {
	q := req.PageReq.ToQuery([]string{"id", "name", "created_at"}, "id")
	items, total, err := s.repo.ListSuppliers(ctx, q)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(items, toSupplierRes), total, q), nil
}

func (s *productService) SupplierOptions(ctx context.Context) ([]*v1.SupplierOption, error) {
	items, err := s.repo.SupplierOptions(ctx)
	if err != nil {
		return nil, err
	}
	return dto.Map(items, func(sp entity.Supplier) *v1.SupplierOption { return &v1.SupplierOption{ID: sp.ID, Name: sp.Name} }), nil
}

func (s *productService) findSupplier(ctx context.Context, id uint) (*entity.Supplier, error) {
	sp, err := s.repo.FindSupplier(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errSupplierNotFound
		}
		return nil, err
	}
	return sp, nil
}

func (s *productService) GetSupplier(ctx context.Context, id uint) (*v1.Supplier, error) {
	sp, err := s.findSupplier(ctx, id)
	if err != nil {
		return nil, err
	}
	return toSupplierRes(*sp), nil
}

func applySupplierInput(sp *entity.Supplier, in v1.SupplierInput) {
	sp.Name = strings.TrimSpace(in.Name)
	sp.Phone = strings.TrimSpace(in.Phone)
	sp.Email = strings.TrimSpace(in.Email)
	sp.Address = strings.TrimSpace(in.Address)
	sp.Note = strings.TrimSpace(in.Note)
}

func (s *productService) CreateSupplier(ctx context.Context, in v1.SupplierInput) (*v1.Supplier, error) {
	sp := &entity.Supplier{}
	applySupplierInput(sp, in)
	if sp.Name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên nhà cung cấp")
	}
	if err := s.repo.CreateSupplier(ctx, sp); err != nil {
		return nil, err
	}
	auditCatalog(ctx, s.audit, AuditCreate, auditEntitySupplier, sp.ID, nil, sp)
	return toSupplierRes(*sp), nil
}

func (s *productService) UpdateSupplier(ctx context.Context, id uint, in v1.SupplierInput) (*v1.Supplier, error) {
	sp, err := s.findSupplier(ctx, id)
	if err != nil {
		return nil, err
	}
	before := *sp
	applySupplierInput(sp, in)
	if sp.Name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên nhà cung cấp")
	}
	if err := s.repo.UpdateSupplier(ctx, sp); err != nil {
		return nil, err
	}
	auditCatalog(ctx, s.audit, AuditUpdate, auditEntitySupplier, sp.ID, before, sp)
	return s.GetSupplier(ctx, id)
}

func (s *productService) DeleteSupplier(ctx context.Context, id uint) error {
	sp, err := s.findSupplier(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteSupplier(ctx, id); err != nil {
		return err
	}
	auditCatalog(ctx, s.audit, AuditDelete, auditEntitySupplier, id, sp, nil)
	return nil
}

// ---- Sản phẩm -------------------------------------------------------------------

var productOrderable = []string{"id", "sku", "name", "sell_price", "cost_price", "min_stock", "status", "created_at"}

func toProductRes(r repository.ProductRow, withBranch bool) *v1.Product {
	out := &v1.Product{ID: r.ID, SKU: r.SKU, Barcode: r.Barcode, Name: r.Name, CategoryID: r.CategoryID,
		CategoryName: r.CategoryName, Unit: r.Unit, CostPrice: r.CostPrice, SellPrice: r.SellPrice,
		MemberPrice: r.MemberPrice, ImageURL: r.ImageURL, MinStock: r.MinStock, Status: r.Status,
		TotalStock: r.TotalStock, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	if withBranch {
		stock, low := r.Stock, r.LowStock
		out.Stock, out.LowStock = &stock, &low
	}
	return out
}

func (s *productService) List(ctx context.Context, req *v1.ProductListReq) (*v1.ProductListRes, error) {
	if req.BranchID > 0 && !reqctx.CanAccessBranch(ctx, req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu chi nhánh này")
	}
	if req.LowStock && req.BranchID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Lọc tồn thấp cần chọn chi nhánh")
	}
	q := req.PageReq.ToQuery(productOrderable, "id")
	q.OrderBy = "p." + q.OrderBy
	f := repository.ProductFilter{CategoryID: req.CategoryID, Status: req.Status, BranchID: req.BranchID, LowStock: req.LowStock}
	rows, total, err := s.repo.ListProducts(ctx, q, f)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, func(r repository.ProductRow) *v1.Product { return toProductRes(r, req.BranchID > 0) }), total, q), nil
}

func (s *productService) Get(ctx context.Context, id uint) (*v1.ProductDetail, error) {
	row, err := s.repo.FindProductRow(ctx, id, 0)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errProductNotFound
		}
		return nil, err
	}
	stocks, err := s.repo.ProductStocks(ctx, id, reqctx.AllowedBranches(ctx))
	if err != nil {
		return nil, err
	}
	return &v1.ProductDetail{
		Product: *toProductRes(*row, false),
		Stocks: dto.Map(stocks, func(r repository.ProductStockRow) *v1.ProductStock {
			return &v1.ProductStock{BranchID: r.BranchID, BranchName: r.BranchName, Quantity: r.Quantity, LowStock: r.LowStock}
		}),
	}, nil
}

// productValidate kiểm tra quy tắc giá & danh mục (hàm có I/O nhẹ).
func (s *productService) validate(ctx context.Context, in *v1.ProductInput, excludeID uint) error {
	in.SKU = strings.ToUpper(strings.TrimSpace(in.SKU))
	in.Barcode = strings.TrimSpace(in.Barcode)
	in.Name = strings.TrimSpace(in.Name)
	in.Unit = strings.TrimSpace(in.Unit)
	if in.Unit == "" {
		in.Unit = "cái"
	}
	if in.Status == "" {
		in.Status = entity.ProductStatusActive
	}
	if in.Name == "" || in.SKU == "" {
		return rescode.New(rescode.BadRequest, "Vui lòng nhập SKU và tên sản phẩm")
	}
	if err := productCheckPrices(in.CostPrice, in.SellPrice, in.MemberPrice); err != nil {
		return err
	}
	if in.MinStock < 0 {
		return rescode.New(rescode.BadRequest, "Tồn tối thiểu không được âm")
	}
	if in.MinStock > stockMaxLevel {
		return rescode.New(rescode.BadRequest, "Tồn tối thiểu tối đa 100.000.000")
	}
	in.ImageURL = strings.TrimSpace(in.ImageURL)
	if !productValidImageURL(in.ImageURL) {
		return rescode.New(rescode.BadRequest, "URL ảnh không hợp lệ (chỉ chấp nhận http(s):// hoặc /uploads/...)")
	}
	if in.CategoryID != nil && *in.CategoryID > 0 {
		if _, err := s.findCategory(ctx, *in.CategoryID); err != nil {
			if errors.Is(err, errCategoryNotFound) {
				return rescode.New(rescode.BadRequest, "Danh mục sản phẩm không tồn tại")
			}
			return err
		}
	} else {
		in.CategoryID = nil
	}
	exists, err := s.repo.ExistsSKU(ctx, in.SKU, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return rescode.New(rescode.Conflict, "Mã SKU đã tồn tại")
	}
	return nil
}

// productCheckPrices: giá không âm, giá hội viên ≤ giá bán.
// productMaxPrice: giá tối đa vừa cột NUMERIC(14,2).
const productMaxPrice = 999_999_999_999

// productValidImageURL: ảnh sản phẩm chỉ chấp nhận URL http(s) tuyệt đối hoặc đường dẫn tương đối /uploads/...
// (chặn javascript:, data:, // … để tránh XSS khi hiển thị).
func productValidImageURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	if strings.ContainsAny(raw, " \t\r\n\\") {
		return false
	}
	if strings.HasPrefix(raw, "/uploads/") {
		_, ok := safeUploadPath(raw)
		return ok
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return (scheme == "http" || scheme == "https") && u.Host != ""
}

func productCheckPrices(cost, sell float64, member *float64) error {
	if cost < 0 || sell < 0 {
		return rescode.New(rescode.BadRequest, "Giá không được âm")
	}
	if cost > productMaxPrice || sell > productMaxPrice {
		return rescode.New(rescode.BadRequest, "Giá vượt quá giới hạn cho phép (tối đa 999.999.999.999đ)")
	}
	if member != nil {
		if *member < 0 {
			return rescode.New(rescode.BadRequest, "Giá hội viên không được âm")
		}
		if *member > productMaxPrice {
			return rescode.New(rescode.BadRequest, "Giá hội viên vượt quá giới hạn cho phép")
		}
		if *member > sell {
			return rescode.New(rescode.BadRequest, "Giá hội viên không được lớn hơn giá bán")
		}
	}
	return nil
}

func applyProductInput(p *entity.Product, in v1.ProductInput) {
	p.SKU, p.Barcode, p.Name, p.CategoryID, p.Unit = in.SKU, in.Barcode, in.Name, in.CategoryID, in.Unit
	p.CostPrice = stockRoundCost(in.CostPrice)
	p.SellPrice = roundMoneyVND(in.SellPrice)
	p.MemberPrice = nil
	if in.MemberPrice != nil {
		v := roundMoneyVND(*in.MemberPrice)
		p.MemberPrice = &v
	}
	p.ImageURL, p.MinStock, p.Status = strings.TrimSpace(in.ImageURL), in.MinStock, in.Status
}

func (s *productService) Create(ctx context.Context, in v1.ProductInput) (*v1.ProductDetail, error) {
	if err := s.validate(ctx, &in, 0); err != nil {
		return nil, err
	}
	p := &entity.Product{}
	applyProductInput(p, in)
	p.CreatedBy = reqctx.UserID(ctx)
	p.UpdatedBy = p.CreatedBy
	if err := s.repo.CreateProduct(ctx, p); err != nil {
		return nil, mapDuplicate(err, "Mã SKU đã tồn tại")
	}
	auditCatalog(ctx, s.audit, AuditCreate, auditEntityProduct, p.ID, nil, p)
	return s.Get(ctx, p.ID)
}

func productPriceSnapshot(p *entity.Product) map[string]any {
	return map[string]any{"cost_price": p.CostPrice, "sell_price": p.SellPrice, "member_price": p.MemberPrice}
}

func productPriceChanged(a, b *entity.Product) bool {
	if a.CostPrice != b.CostPrice || a.SellPrice != b.SellPrice {
		return true
	}
	if (a.MemberPrice == nil) != (b.MemberPrice == nil) {
		return true
	}
	return a.MemberPrice != nil && *a.MemberPrice != *b.MemberPrice
}

func (s *productService) Update(ctx context.Context, id uint, in v1.ProductInput) (*v1.ProductDetail, error) {
	p, err := s.repo.FindProduct(ctx, id, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errProductNotFound
		}
		return nil, err
	}
	if err := s.validate(ctx, &in, id); err != nil {
		return nil, err
	}
	before := *p
	applyProductInput(p, in)
	// Giá vốn là bình quân gia quyền do phiếu nhập kho tính (stockWeightedCost) → KHÔNG cho client ghi đè khi
	// cập nhật (trước đây PUT gửi lại giá vốn cũ trên form làm mất giá vốn mới vừa nhập kho). Repo cũng không ghi cột này.
	p.CostPrice = before.CostPrice
	p.UpdatedBy = reqctx.UserID(ctx)
	if err := s.repo.UpdateProduct(ctx, p); err != nil {
		return nil, mapDuplicate(err, "Mã SKU đã tồn tại")
	}
	auditCatalog(ctx, s.audit, AuditUpdate, auditEntityProduct, p.ID, before, p)
	if productPriceChanged(&before, p) {
		auditCatalog(ctx, s.audit, AuditPriceChange, auditEntityProduct, p.ID, productPriceSnapshot(&before), productPriceSnapshot(p))
	}
	return s.Get(ctx, id)
}

func (s *productService) Delete(ctx context.Context, id uint) error {
	p, err := s.repo.FindProduct(ctx, id, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errProductNotFound
		}
		return err
	}
	total, err := s.repo.TotalStock(ctx, id)
	if err != nil {
		return err
	}
	if total > 0 {
		return rescode.New(rescode.Conflict, fmt.Sprintf("Sản phẩm còn tồn %d %s, hãy điều chỉnh tồn về 0 trước khi xóa", total, p.Unit))
	}
	if err := s.repo.DeleteProduct(ctx, id, reqctx.UserID(ctx)); err != nil {
		return err
	}
	auditCatalog(ctx, s.audit, AuditDelete, auditEntityProduct, id, p, nil)
	return nil
}
