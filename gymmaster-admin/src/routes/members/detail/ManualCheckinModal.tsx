import { useEffect, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Button, Form, Input, Modal, Space } from 'antd';
import checkinApi from '@/api/checkin';
import type { ScanResult } from '@/api/checkin';
import memberApi from '@/api/member';
import type { Member } from '@/api/member';
import BranchSelect from '@/components/BranchSelect';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';
import ResultCard from '../../checkin/ResultCard';
import { playError, playSuccess } from '../../checkin/sound';

/** Check-in thủ công từ hồ sơ hội viên (chọn chi nhánh + ghi chú) → hiển thị thẻ kết quả. */
export default function ManualCheckinModal({
  member,
  open,
  onClose,
}: {
  member: Member;
  open: boolean;
  onClose: () => void;
}) {
  const { branchId: working } = useWorkingBranch();
  const queryClient = useQueryClient();
  const [branchId, setBranchId] = useState<number | null>(null);
  const [note, setNote] = useState('');
  const [result, setResult] = useState<ScanResult | null>(null);

  useEffect(() => {
    if (!open) return;
    setBranchId(working ?? member.home_branch_id);
    setNote('');
    setResult(null);
  }, [open, working, member.home_branch_id]);

  const run = useMutation({
    mutationFn: () => checkinApi.manual(member.id, branchId!, note.trim()),
    onSuccess: (res) => {
      setResult(res);
      if (res.result === 'ALLOWED') playSuccess();
      else playError();
      void queryClient.invalidateQueries({ queryKey: [checkinApi.key] });
      void queryClient.invalidateQueries({ queryKey: [memberApi.key, 'detail', member.id] });
    },
    onError: (e) => showError(e, 'Không thể check-in'),
  });

  return (
    <Modal
      open={open}
      title={`Check-in thủ công · ${member.full_name}`}
      onCancel={onClose}
      width={560}
      footer={
        result ? (
          <Button type="primary" onClick={onClose}>
            Đóng
          </Button>
        ) : (
          <Space>
            <Button onClick={onClose}>Hủy</Button>
            <Button
              type="primary"
              disabled={!branchId}
              loading={run.isPending}
              onClick={() => run.mutate()}
            >
              Check-in
            </Button>
          </Space>
        )
      }
    >
      {result ? (
        <ResultCard result={result} compact />
      ) : (
        <Form layout="vertical">
          <Form.Item label="Chi nhánh" required>
            <BranchSelect
              activeOnly
              value={branchId}
              onChange={(v) => setBranchId((v as number | null) ?? null)}
            />
          </Form.Item>
          <Form.Item label="Ghi chú">
            <Input
              value={note}
              maxLength={255}
              placeholder="VD: Quên điện thoại"
              onChange={(e) => setNote(e.target.value)}
            />
          </Form.Item>
        </Form>
      )}
    </Modal>
  );
}
