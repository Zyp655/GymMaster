export interface Role {
  id: number;
  code: string;
  name: string;
  description: string;
  is_system: boolean;
  /** Danh sách mã quyền, VD: ["branch.view"] */
  permissions: string[];
}

export interface Permission {
  id: number;
  code: string;
  module: string;
  name: string;
}

export interface CreateRolePayload {
  code: string;
  name: string;
  description: string;
  permissions: string[];
}

export type UpdateRolePayload = Omit<CreateRolePayload, 'code'>;
