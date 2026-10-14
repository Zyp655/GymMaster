import { lazy, Suspense, useState } from 'react';
import { Button, Tooltip } from 'antd';
import { ShoppingCartOutlined } from '@ant-design/icons';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';

const SellSubscriptionModal = lazy(() => import('./SellSubscriptionModal'));

/** Nút "Bán gói" nhanh ở header (chỉ hiện khi có quyền `subscription.sell`). */
export default function QuickSellButton({ compact }: { compact?: boolean }) {
  const { can } = usePermission();
  const [open, setOpen] = useState(false);
  const [loaded, setLoaded] = useState(false);
  if (!can(PERM.SUBSCRIPTION_SELL)) return null;

  return (
    <>
      <Tooltip title={compact ? 'Bán gói tập' : undefined}>
        <Button
          color="orange"
          variant="solid"
          icon={<ShoppingCartOutlined />}
          onClick={() => {
            setLoaded(true);
            setOpen(true);
          }}
          aria-label="Bán gói tập"
        >
          {compact ? null : 'Bán gói'}
        </Button>
      </Tooltip>
      {loaded && (
        <Suspense fallback={null}>
          <SellSubscriptionModal open={open} onClose={() => setOpen(false)} />
        </Suspense>
      )}
    </>
  );
}
