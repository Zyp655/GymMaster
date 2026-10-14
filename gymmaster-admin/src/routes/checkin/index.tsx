import { useCallback, useEffect, useRef, useState } from 'react';
import type { KeyboardEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Alert, Button, Card, Col, Form, Input, Row, Space, Tabs, Typography } from 'antd';
import type { InputRef } from 'antd';
import {
  CameraOutlined,
  HistoryOutlined,
  LoginOutlined,
  QrcodeOutlined,
  StopOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import checkinApi from '@/api/checkin';
import type { ScanResult } from '@/api/checkin';
import memberApi from '@/api/member';
import BranchSelect from '@/components/BranchSelect';
import MemberSelect from '@/components/MemberSelect';
import PageTitle from '@/components/PageTitle';
import { PERM } from '@/constants/permissions';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';
import CameraScanner from './CameraScanner';
import { cameraScanSupported } from './barcode';
import ResultCard from './ResultCard';
import TodayPanel from './TodayPanel';
import { playError, playSuccess } from './sound';
import './checkin.css';

const BRANCH_KEY = 'gm_checkin_branch';
/** Tự xóa thẻ kết quả sau (ms). */
const RESULT_TTL = 8000;
/** Bỏ qua cùng một mã trong khoảng (ms) – camera đọc liên tục. */
const SAME_TOKEN_WINDOW = 3000;

function readStoredBranch(): number | null {
  try {
    const v = Number(localStorage.getItem(BRANCH_KEY));
    return Number.isInteger(v) && v > 0 ? v : null;
  } catch {
    return null;
  }
}

/** Màn hình check-in tại quầy `/app/checkin`. */
export default function CheckinPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { can } = usePermission();
  const { branchId: working } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const activeBranches = branches.filter((b) => b.status !== 'INACTIVE');

  const [branchId, setBranchId] = useState<number | null>(() => working ?? readStoredBranch());
  const [tab, setTab] = useState<'qr' | 'manual'>('qr');
  const [code, setCode] = useState('');
  const [camera, setCamera] = useState(false);
  const [memberId, setMemberId] = useState<number>();
  const [note, setNote] = useState('');
  const [result, setResult] = useState<ScanResult | null>(null);

  const inputRef = useRef<InputRef>(null);
  const lastToken = useRef<{ token: string; at: number }>({ token: '', at: 0 });
  const clearTimer = useRef<ReturnType<typeof setTimeout>>(undefined);

  // Chi nhánh làm việc ở header đổi → theo; chưa chọn → chi nhánh đầu tiên.
  useEffect(() => {
    if (working) setBranchId(working);
  }, [working]);
  useEffect(() => {
    if (!branchId && activeBranches.length > 0) setBranchId(activeBranches[0].id);
    else if (branchId && branches.length > 0 && !branches.some((b) => b.id === branchId)) {
      setBranchId(activeBranches[0]?.id ?? null);
    }
  }, [branchId, branches, activeBranches]);
  useEffect(() => {
    try {
      if (branchId) localStorage.setItem(BRANCH_KEY, String(branchId));
    } catch {
      /* bỏ qua */
    }
  }, [branchId]);

  const focusInput = useCallback(() => {
    if (tab === 'qr') inputRef.current?.focus({ preventScroll: true });
  }, [tab]);

  // Ô nhập luôn focus (máy quét cầm tay gõ như bàn phím) – trừ khi người dùng đang ở ô khác/hộp thoại.
  useEffect(() => {
    if (tab !== 'qr') return;
    focusInput();
    const t = setInterval(() => {
      const el = document.activeElement;
      const busy =
        el &&
        el !== document.body &&
        (el.matches('input, textarea, select, [contenteditable="true"]') ||
          !!el.closest('.ant-select-dropdown, .ant-modal, .ant-dropdown'));
      if (!busy) focusInput();
    }, 800);
    return () => clearInterval(t);
  }, [tab, focusInput]);

  useEffect(() => () => clearTimeout(clearTimer.current), []);

  const showResult = (res: ScanResult) => {
    setResult(res);
    if (res.result === 'ALLOWED') playSuccess();
    else playError();
    clearTimeout(clearTimer.current);
    clearTimer.current = setTimeout(() => setResult(null), RESULT_TTL);
    void queryClient.invalidateQueries({ queryKey: [checkinApi.key] });
    if (res.member) {
      void queryClient.invalidateQueries({ queryKey: [memberApi.key, 'detail', res.member.id] });
    }
  };

  const scan = useMutation({
    mutationFn: (token: string) => checkinApi.scan(token, branchId!),
    onSuccess: showResult,
    onError: (e) => {
      playError();
      showError(e, 'Không thể check-in');
    },
  });

  const manual = useMutation({
    mutationFn: () => checkinApi.manual(memberId!, branchId!, note.trim()),
    onSuccess: (res) => {
      showResult(res);
      if (res.result === 'ALLOWED') {
        setMemberId(undefined);
        setNote('');
      }
    },
    onError: (e) => {
      playError();
      showError(e, 'Không thể check-in');
    },
  });

  const submitToken = useCallback(
    (raw: string) => {
      const token = raw.trim();
      if (!token || !branchId) return;
      const now = Date.now();
      if (lastToken.current.token === token && now - lastToken.current.at < SAME_TOKEN_WINDOW) {
        return;
      }
      lastToken.current = { token, at: now };
      scan.mutate(token);
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [branchId],
  );

  const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      submitToken(code);
      setCode('');
    }
  };

  const canScan = can(PERM.CHECKIN_SCAN);
  const showCameraButton = cameraScanSupported();

  return (
    <div>
      <Card variant="borderless" className="page-card">
        <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
          <Col>
            <PageTitle
              title="Check-in tại quầy"
              subtitle="Quét mã QR hội viên hoặc check-in thủ công"
            />
          </Col>
          <Col>
            {can(PERM.CHECKIN_VIEW) && (
              <Button icon={<HistoryOutlined />} onClick={() => navigate('/app/checkins')}>
                Lịch sử ra vào
              </Button>
            )}
          </Col>
        </Row>

        <Row gutter={[24, 16]}>
          <Col xs={24} lg={10}>
            <Form layout="vertical">
              <Form.Item label="Chi nhánh check-in" style={{ marginBottom: 12 }}>
                <BranchSelect
                  activeOnly
                  size="large"
                  value={branchId}
                  onChange={(v) => setBranchId((v as number | null) ?? null)}
                />
              </Form.Item>
            </Form>
            {!branchId && <Alert type="warning" showIcon message="Chọn chi nhánh trước khi quét" />}

            <Tabs
              activeKey={tab}
              onChange={(k) => setTab(k as 'qr' | 'manual')}
              items={[
                {
                  key: 'qr',
                  label: (
                    <span>
                      <QrcodeOutlined /> Quét mã QR
                    </span>
                  ),
                  children: (
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <Input
                        ref={inputRef}
                        size="large"
                        className="checkin-scan-input"
                        prefix={<QrcodeOutlined />}
                        placeholder="Quét hoặc dán mã QR rồi nhấn Enter"
                        value={code}
                        disabled={!canScan || !branchId}
                        autoComplete="off"
                        spellCheck={false}
                        onChange={(e) => setCode(e.target.value)}
                        onKeyDown={onKeyDown}
                        suffix={
                          <Button
                            type="primary"
                            size="small"
                            loading={scan.isPending}
                            disabled={!code.trim() || !branchId}
                            onClick={() => {
                              submitToken(code);
                              setCode('');
                            }}
                          >
                            Gửi
                          </Button>
                        }
                      />
                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                        Máy quét cầm tay sẽ tự gõ mã và Enter. Ô nhập luôn được giữ focus.
                      </Typography.Text>
                      {showCameraButton && (
                        <Button
                          block
                          icon={camera ? <StopOutlined /> : <CameraOutlined />}
                          onClick={() => setCamera((c) => !c)}
                          disabled={!branchId}
                        >
                          {camera ? 'Tắt camera' : 'Quét bằng camera'}
                        </Button>
                      )}
                      {camera && branchId && <CameraScanner onDetect={submitToken} />}
                    </Space>
                  ),
                },
                {
                  key: 'manual',
                  label: (
                    <span>
                      <LoginOutlined /> Thủ công
                    </span>
                  ),
                  children: (
                    <Form layout="vertical" onFinish={() => manual.mutate()}>
                      <Form.Item label="Hội viên" required>
                        <MemberSelect
                          size="large"
                          value={memberId}
                          onChange={(id) => setMemberId(id)}
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
                      <Button
                        type="primary"
                        htmlType="submit"
                        size="large"
                        block
                        icon={<LoginOutlined />}
                        loading={manual.isPending}
                        disabled={!memberId || !branchId || !canScan}
                      >
                        Check-in
                      </Button>
                    </Form>
                  ),
                },
              ]}
            />
          </Col>

          <Col xs={24} lg={14}>
            {result ? (
              <div onClick={() => setResult(null)} title="Bấm để ẩn">
                <ResultCard result={result} />
              </div>
            ) : (
              <div className="checkin-idle">
                <QrcodeOutlined style={{ fontSize: 56, marginBottom: 12 }} />
                <Typography.Title level={4} type="secondary" style={{ margin: 0 }}>
                  Sẵn sàng quét
                </Typography.Title>
                <Typography.Text type="secondary">
                  Kết quả check-in sẽ hiển thị tại đây
                </Typography.Text>
              </div>
            )}
          </Col>
        </Row>
      </Card>

      <TodayPanel branchId={branchId} />
    </div>
  );
}
