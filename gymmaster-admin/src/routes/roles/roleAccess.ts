import { ROLE } from '@/constants/permissions';

/**
 * Quy tắc chống leo thang đặc quyền – sao chép từ `gymmaster-api/internal/service/role_service.go`
 * (`ensureCanManageRole` / `ensureCanGrantPermissions`) để ẩn nút trước khi API trả 403.
 */
export interface RoleActor {
  roles?: readonly string[] | null;
  permissions?: readonly string[] | null;
}

export interface RoleTarget {
  code: string;
  permissions?: readonly string[] | null;
}

export interface ManageCheck {
  ok: boolean;
  /** Lý do không được sửa/xóa (thông điệp giống API); `null` khi được phép. */
  reason: string | null;
}

const isOwnerActor = (actor: RoleActor | null | undefined) => !!actor?.roles?.includes(ROLE.OWNER);

/** Các mã quyền trong `perms` mà người thao tác KHÔNG có (OWNER → luôn rỗng). */
export function missingPermissions(
  actor: RoleActor | null | undefined,
  perms: readonly string[] | null | undefined,
): string[] {
  if (isOwnerActor(actor)) return [];
  const owned = new Set(actor?.permissions ?? []);
  return (perms ?? []).filter((p) => !owned.has(p));
}

/** OWNER sửa/xóa mọi vai trò; người khác không được đụng OWNER, vai trò mình giữ, vai trò quyền cao hơn. */
export function canManageRole(
  actor: RoleActor | null | undefined,
  role: RoleTarget | null | undefined,
): ManageCheck {
  if (!actor || !role) return { ok: false, reason: null };
  if (isOwnerActor(actor)) return { ok: true, reason: null };
  if (role.code === ROLE.OWNER) {
    return { ok: false, reason: 'Chỉ chủ trung tâm mới được thay đổi vai trò chủ trung tâm' };
  }
  if (actor.roles?.includes(role.code)) {
    return { ok: false, reason: 'Không thể tự thay đổi vai trò mà chính mình đang giữ' };
  }
  if (missingPermissions(actor, role.permissions).length > 0) {
    return { ok: false, reason: 'Không thể thay đổi vai trò có quyền cao hơn quyền của bạn' };
  }
  return { ok: true, reason: null };
}

/**
 * Các quyền người thao tác được phép cấp (OWNER: tất cả; người khác: chỉ quyền mình đang có),
 * giữ nguyên thứ tự `allCodes`.
 */
export function grantablePermissions(
  actor: RoleActor | null | undefined,
  allCodes: readonly string[],
): string[] {
  if (isOwnerActor(actor)) return [...allCodes];
  const owned = new Set(actor?.permissions ?? []);
  return allCodes.filter((c) => owned.has(c));
}
