package reqctx

import (
	"context"
	"slices"

	"gymmaster-api/internal/consts"
)

// AllowedBranches trả danh sách chi nhánh người dùng được thao tác.
// nil = toàn hệ thống (OWNER hoặc role không gắn chi nhánh).
func AllowedBranches(ctx context.Context) []uint {
	p := Auth(ctx)
	if p == nil || p.HasRole(consts.RoleOwner) || len(p.BranchIDs) == 0 {
		return nil
	}
	return p.BranchIDs
}

// CanAccessBranch kiểm tra người dùng có được thao tác trên chi nhánh branchID không.
func CanAccessBranch(ctx context.Context, branchID uint) bool {
	allowed := AllowedBranches(ctx)
	return allowed == nil || slices.Contains(allowed, branchID)
}

// ResolveBranchFilter dùng cho API danh sách có tham số branch_id tùy chọn:
//   - requested > 0 và được phép → []uint{requested}
//   - requested > 0 nhưng không được phép → ok=false (service trả 403)
//   - requested = 0 → AllowedBranches (nil = không lọc)
func ResolveBranchFilter(ctx context.Context, requested uint) (branchIDs []uint, ok bool) {
	if requested > 0 {
		if !CanAccessBranch(ctx, requested) {
			return nil, false
		}
		return []uint{requested}, true
	}
	return AllowedBranches(ctx), true
}

// HasRole / HasPermission tiện dụng cho service (OWNER luôn có mọi quyền).
func HasRole(ctx context.Context, role string) bool {
	p := Auth(ctx)
	return p != nil && p.HasRole(role)
}

func HasPermission(ctx context.Context, perm string) bool {
	p := Auth(ctx)
	return p != nil && (p.HasRole(consts.RoleOwner) || p.HasPermission(perm))
}
