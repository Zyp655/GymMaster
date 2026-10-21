import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { App, DatePicker, Form, InputNumber, Modal, Select, Typography } from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import invoiceApi from '@/api/invoice';
import type { Invoice } from '@/api/invoice';
import { financeExtApi } from '@/api/payroll-finance';
import { setFormError, showError } from '@/utils/errors';
import { DATE_FORMAT, formatMoney, toApiDate } from '@/utils/format';

type Values = { invoice_id: number; installments: number; first_due_date?: Dayjs };

export interface InstallmentPlanModalProps {
  open: boolean;
  onClose: () => void;
  /** Hóa đơn cố định (mở từ chi tiết hóa đơn) – không cần chọn. */
  invoice?: Pick<Invoice, 'id' | 'invoice_no' | 'due' | 'member'> | null;
}

const invoiceLabel = (i: Pick<Invoice, 'invoice_no' | 'due' | 'member'>) =>
  `${i.invoice_no} · ${i.member?.full_name ?? 'Khách lẻ'}${i.member?.member_code ? ` (${i.member.member_code})` : ''} · còn nợ ${formatMoney(i.due)}`;

/** Tạo kế hoạch trả góp: chọn hóa đơn còn nợ bằng tìm kiếm (số HĐ / hội viên), kỳ đầu không ở quá khứ. */
export default function InstallmentPlanModal({
  open,
  onClose,
  invoice,
}: InstallmentPlanModalProps) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const [form] = Form.useForm<Values>();
  const [keyword, setKeyword] = useState('');

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    setKeyword('');
    form.setFieldsValue({
      invoice_id: invoice?.id,
      installments: 3,
      first_due_date: dayjs().add(1, 'month'),
    });
  }, [open, invoice, form]);

  // Chỉ hóa đơn còn nợ (chưa thu / thu một phần) mới chia kỳ được.
  const search = useQuery({
    queryKey: [invoiceApi.key, 'installment-pick', keyword],
    queryFn: async () => {
      const [a, b] = await Promise.all([
        invoiceApi.list({ keyword: keyword || undefined, status: 'UNPAID', limit: 20 }),
        invoiceApi.list({ keyword: keyword || undefined, status: 'PARTIAL', limit: 20 }),
      ]);
      return [...a.items, ...b.items]
        .filter((i) => i.member && i.due > 0)
        .sort((x, y) => y.id - x.id);
    },
    enabled: open && !invoice,
    staleTime: 30_000,
  });

  const create = useMutation({
    mutationFn: (v: Values) =>
      financeExtApi.createInstallmentPlan(Number(v.invoice_id), {
        installments: Number(v.installments) || 3,
        first_due_date: toApiDate(v.first_due_date) ?? undefined,
      }),
    onSuccess: () => {
      message.success('Đã tạo kế hoạch trả góp cho hóa đơn');
      void qc.invalidateQueries({ queryKey: [financeExtApi.installmentKey] });
      void qc.invalidateQueries({ queryKey: [invoiceApi.key] });
      onClose();
    },
    onError: (err) => {
      if (
        !setFormError(form, err, {
          installments: /số kỳ/i,
          first_due_date: /ngày đến hạn/i,
          invoice_id: /hóa đơn/i,
        })
      )
        showError(err, 'Không thể tạo kế hoạch trả góp');
    },
  });

  return (
    <Modal
      open={open}
      title="Tạo kế hoạch trả góp cho hóa đơn"
      okText="Tạo lịch trả góp"
      cancelText="Hủy"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={create.isPending}
      width={520}
      forceRender
    >
      <Form<Values>
        form={form}
        layout="vertical"
        onFinish={(v) => create.mutate(v)}
        style={{ marginTop: 12 }}
      >
        {invoice ? (
          <Form.Item name="invoice_id" label="Hóa đơn" hidden>
            <InputNumber />
          </Form.Item>
        ) : (
          <Form.Item
            name="invoice_id"
            label="Hóa đơn còn nợ"
            rules={[{ required: true, message: 'Chọn hóa đơn cần chia kỳ' }]}
          >
            <Select
              showSearch
              filterOption={false}
              onSearch={setKeyword}
              loading={search.isFetching}
              placeholder="Tìm theo số hóa đơn, tên / mã / SĐT hội viên"
              notFoundContent={search.isFetching ? 'Đang tìm…' : 'Không có hóa đơn còn nợ phù hợp'}
              options={(search.data ?? []).map((i) => ({ value: i.id, label: invoiceLabel(i) }))}
            />
          </Form.Item>
        )}
        {invoice && (
          <Typography.Paragraph>
            Hóa đơn: <strong>{invoiceLabel(invoice)}</strong>
          </Typography.Paragraph>
        )}
        <Form.Item
          name="installments"
          label="Số kỳ trả góp (2 – 12 kỳ)"
          rules={[
            { required: true, message: 'Nhập số kỳ trả góp' },
            { type: 'number', min: 2, max: 12, message: 'Số kỳ trả góp phải từ 2 đến 12 kỳ' },
          ]}
        >
          <InputNumber min={2} max={12} precision={0} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="first_due_date" label="Ngày đến hạn kỳ đầu tiên">
          <DatePicker
            style={{ width: '100%' }}
            format={DATE_FORMAT}
            disabledDate={(d) => d.isBefore(dayjs(), 'day')}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
