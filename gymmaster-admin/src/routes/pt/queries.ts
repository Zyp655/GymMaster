import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import invoiceApi from '@/api/invoice';
import memberApi from '@/api/member';
import ptApi from '@/api/pt';
import type {
  CancelContractInput,
  ChangeTrainerInput,
  CommissionPayInput,
  PtSellRequest,
  PtSessionCancelInput,
  PtSessionCompleteInput,
} from '@/api/pt';
import ptPackageApi from '@/api/ptPackage';
import settingApi from '@/api/setting';
import staffApi from '@/api/staff';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { DEFAULT_CANCEL_BEFORE_MINUTES } from './constants';

const OPTIONS_STALE = 5 * 60_000;

/** Thao tác PT ảnh hưởng hợp đồng ↔ buổi ↔ hoa hồng ↔ hóa đơn → làm mới cả nhóm. */
export function invalidatePt(queryClient: QueryClient) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: [ptApi.key] }),
    queryClient.invalidateQueries({ queryKey: [ptApi.commissionKey] }),
    queryClient.invalidateQueries({ queryKey: [invoiceApi.key] }),
    queryClient.invalidateQueries({ queryKey: [memberApi.key] }),
  ]);
}

function usePtMutation<V, R>(fn: (v: V) => Promise<R>) {
  const queryClient = useQueryClient();
  return useMutation({ mutationFn: fn, onSuccess: () => invalidatePt(queryClient) });
}

/* ------------------------------------------------------------------ Dropdown */

export function usePtPackageOptions(enabled = true) {
  return useQuery({
    queryKey: [ptPackageApi.key, 'options'],
    queryFn: () => ptPackageApi.options(),
    staleTime: OPTIONS_STALE,
    enabled,
  });
}

export function usePtTrainerOptions(branchId?: number | null, enabled = true) {
  return useQuery({
    queryKey: ['pt-options', 'trainers', branchId ?? null],
    queryFn: () => staffApi.trainerOptions(branchId ?? undefined),
    staleTime: OPTIONS_STALE,
    enabled,
  });
}

/** Hạn hủy buổi PT (phút) – đọc cấu hình nếu có quyền, không thì dùng mặc định 720. */
export function useCancelBeforeMinutes() {
  const { can } = usePermission();
  const allowed = can(PERM.SETTING_MANAGE);
  const q = useQuery({
    queryKey: [settingApi.key, 'list'],
    queryFn: () => settingApi.list(),
    staleTime: OPTIONS_STALE,
    enabled: allowed,
    retry: false,
  });
  const v = q.data?.find((s) => s.key === 'pt.cancel_before_minutes')?.value;
  return typeof v === 'number' ? v : DEFAULT_CANCEL_BEFORE_MINUTES;
}

/* ------------------------------------------------------------------ Dữ liệu */

export function usePtContract(id?: number | null) {
  return useQuery({
    queryKey: [ptApi.key, 'contract', id],
    queryFn: () => ptApi.getContract(id as number),
    enabled: !!id,
  });
}

export function useMemberPtContracts(memberId?: number | null) {
  return useQuery({
    queryKey: [ptApi.key, 'member-contracts', memberId],
    queryFn: () => ptApi.memberContracts(memberId as number),
    enabled: !!memberId,
  });
}

export function usePtSession(id?: number | null) {
  return useQuery({
    queryKey: [ptApi.key, 'session', id],
    queryFn: () => ptApi.getSession(id as number),
    enabled: !!id,
  });
}

/* ------------------------------------------------------------------ Mutation */

export const useSellPt = () => usePtMutation((data: PtSellRequest) => ptApi.sell(data));

export const useChangeTrainer = () =>
  usePtMutation(({ id, data }: { id: number; data: ChangeTrainerInput }) =>
    ptApi.changeTrainer(id, data),
  );

export const useCancelContract = () =>
  usePtMutation(({ id, data }: { id: number; data: CancelContractInput }) =>
    ptApi.cancelContract(id, data),
  );

export const useBookSession = () =>
  usePtMutation(({ contractId, startAt }: { contractId: number; startAt: string }) =>
    ptApi.createSession(contractId, startAt),
  );

export const useRescheduleSession = () =>
  usePtMutation(({ id, startAt }: { id: number; startAt: string }) =>
    ptApi.reschedule(id, startAt),
  );

export const useCancelSession = () =>
  usePtMutation(({ id, data }: { id: number; data: PtSessionCancelInput }) =>
    ptApi.cancelSession(id, data),
  );

export const useCompleteSession = () =>
  usePtMutation(({ id, data }: { id: number; data: PtSessionCompleteInput }) =>
    ptApi.completeSession(id, data),
  );

export const useNoShowSession = () =>
  usePtMutation(({ id, note }: { id: number; note?: string }) => ptApi.noShow(id, note));

export const usePayCommission = () =>
  usePtMutation((data: CommissionPayInput) => ptApi.payCommission(data));
