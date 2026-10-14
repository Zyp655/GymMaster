import { Suspense, lazy } from 'react';
import { Segmented, Skeleton } from 'antd';
import type { Member } from '@/api/member';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { useUrlParams } from '@/hooks/useUrlParams';

const MemberBookingsTab = lazy(() => import('@/routes/classes/MemberBookingsTab'));
const MemberPtTab = lazy(() => import('./MemberPtTab'));

type Part = 'class' | 'pt';

/**
 * Tab "Lớp & PT" của hồ sơ hội viên: Segmented "Lớp nhóm" (MemberBookingsTab của module lớp)
 * | "PT" (hợp đồng + buổi sắp tới). Phần đang xem lưu ở URL `?part=pt`.
 */
export default function MemberClassPtTab({ member }: { member: Member }) {
  const { can } = usePermission();
  const { get, set } = useUrlParams();
  const canClass = can([PERM.CLASS_VIEW, PERM.BOOKING_MANAGE]);
  const canPt = can(PERM.PT_VIEW);
  const wanted: Part = get('part') === 'pt' ? 'pt' : 'class';
  const part: Part = !canClass ? 'pt' : !canPt ? 'class' : wanted;

  return (
    <div>
      {canClass && canPt && (
        <Segmented<Part>
          style={{ marginBottom: 12 }}
          value={part}
          onChange={(v) => set({ part: v === 'pt' ? 'pt' : undefined })}
          options={[
            { value: 'class', label: 'Lớp nhóm' },
            { value: 'pt', label: 'PT' },
          ]}
        />
      )}
      <Suspense fallback={<Skeleton active />}>
        {part === 'pt' ? (
          <MemberPtTab member={member} />
        ) : (
          <MemberBookingsTab memberId={member.id} />
        )}
      </Suspense>
    </div>
  );
}
