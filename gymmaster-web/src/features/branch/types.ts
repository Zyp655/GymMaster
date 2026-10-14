import type { ActiveStatus } from '@/components/common/StatusBadge';
import type { ListParams } from '@/types/api';

export interface Branch {
  id: number;
  code: string;
  name: string;
  address: string;
  phone: string;
  email: string;
  /** "HH:mm" */
  open_time: string;
  /** "HH:mm" */
  close_time: string;
  status: ActiveStatus;
  created_at: string;
  updated_at: string;
}

export type BranchPayload = Omit<Branch, 'id' | 'created_at' | 'updated_at'>;

export interface BranchListParams extends ListParams {
  status?: string;
}
