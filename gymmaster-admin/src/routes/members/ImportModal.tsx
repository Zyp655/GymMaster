import { useEffect, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Col,
  Form,
  Modal,
  Result,
  Row,
  Space,
  Table,
  Typography,
  Upload,
} from 'antd';
import type { UploadFile } from 'antd';
import { DownloadOutlined, InboxOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import memberApi from '@/api/member';
import type { MemberImportResult } from '@/api/member';
import BranchSelect from '@/components/BranchSelect';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';

export interface ImportModalProps {
  open: boolean;
  onClose: () => void;
}

const XLSX_MIME = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';

/** Tải Blob về máy với tên file cho trước. */
function saveBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

/** Nhập hội viên từ Excel: tải mẫu → chọn chi nhánh + file → kết quả created/skipped + lỗi theo dòng. */
export default function ImportModal({ open, onClose }: ImportModalProps) {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { branchId: workingBranch } = useWorkingBranch();
  const [branchId, setBranchId] = useState<number | null>(null);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [result, setResult] = useState<MemberImportResult | null>(null);
  const [downloading, setDownloading] = useState(false);

  useEffect(() => {
    if (!open) return;
    setBranchId(workingBranch ?? null);
    setFileList([]);
    setResult(null);
  }, [open, workingBranch]);

  const run = useMutation({
    mutationFn: ({ file, branch }: { file: File; branch: number }) =>
      memberApi.import(file, branch),
    onSuccess: (res) => {
      setResult(res);
      if (res.created > 0) {
        message.success(`Đã nhập ${res.created} hội viên`);
        void queryClient.invalidateQueries({ queryKey: [memberApi.key] });
      }
    },
    onError: (e) => showError(e, 'Không thể nhập file'),
  });

  const downloadTemplate = async () => {
    setDownloading(true);
    try {
      const blob = await memberApi.importTemplate();
      saveBlob(blob, 'mau-nhap-hoi-vien.xlsx');
    } catch (e) {
      showError(e, 'Không tải được file mẫu');
    } finally {
      setDownloading(false);
    }
  };

  const file = fileList[0]?.originFileObj;
  const submit = () => {
    if (!branchId) return message.warning('Vui lòng chọn chi nhánh');
    if (!file) return message.warning('Vui lòng chọn file Excel (.xlsx)');
    run.mutate({ file, branch: branchId });
  };

  return (
    <Modal
      open={open}
      title="Nhập hội viên từ Excel"
      onCancel={onClose}
      width={720}
      footer={
        result ? (
          <Space>
            <Button
              onClick={() => {
                setResult(null);
                setFileList([]);
              }}
            >
              Nhập file khác
            </Button>
            <Button type="primary" onClick={onClose}>
              Đóng
            </Button>
          </Space>
        ) : (
          <Space>
            <Button onClick={onClose}>Hủy</Button>
            <Button type="primary" loading={run.isPending} onClick={submit}>
              Nhập dữ liệu
            </Button>
          </Space>
        )
      }
    >
      {result ? (
        <>
          <Result
            status={result.created > 0 ? (result.skipped > 0 ? 'warning' : 'success') : 'error'}
            title={`Đã thêm ${result.created} hội viên`}
            subTitle={
              result.skipped > 0
                ? `Bỏ qua ${result.skipped} dòng (xem chi tiết bên dưới)`
                : undefined
            }
            style={{ padding: '12px 0' }}
          />
          {result.errors.length > 0 && (
            <Table
              size="small"
              rowKey={(r) => `${r.row}-${r.message}`}
              dataSource={result.errors}
              pagination={result.errors.length > 10 ? { pageSize: 10, size: 'small' } : false}
              columns={[
                { title: 'Dòng', dataIndex: 'row', width: 80 },
                { title: 'Lý do', dataIndex: 'message' },
              ]}
            />
          )}
        </>
      ) : (
        <Form layout="vertical">
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            message="Dùng đúng file mẫu: Họ tên*, SĐT*, Email, Giới tính, Ngày sinh (dd/mm/yyyy), CCCD, Địa chỉ, Nguồn. Tối đa 2000 dòng; dòng trùng SĐT sẽ bị bỏ qua."
            action={
              <Button
                size="small"
                icon={<DownloadOutlined />}
                loading={downloading}
                onClick={downloadTemplate}
              >
                Tải file mẫu
              </Button>
            }
          />
          <Row gutter={16}>
            <Col xs={24} sm={12}>
              <Form.Item label="Chi nhánh chính cho hội viên được nhập" required>
                <BranchSelect
                  activeOnly
                  value={branchId}
                  onChange={(v) => setBranchId((v as number | null) ?? null)}
                />
              </Form.Item>
            </Col>
          </Row>
          <Upload.Dragger
            accept={`.xlsx,${XLSX_MIME}`}
            maxCount={1}
            fileList={fileList}
            beforeUpload={(f) => {
              if (!/\.xlsx$/i.test(f.name)) {
                message.error('Chỉ nhận file Excel (.xlsx)');
                return Upload.LIST_IGNORE;
              }
              return false;
            }}
            onChange={({ fileList: next }) => setFileList(next.slice(-1))}
          >
            <p className="ant-upload-drag-icon">
              <InboxOutlined />
            </p>
            <p className="ant-upload-text">Kéo thả hoặc bấm để chọn file .xlsx</p>
            <Typography.Text type="secondary">Mỗi lần một file</Typography.Text>
          </Upload.Dragger>
        </Form>
      )}
    </Modal>
  );
}
