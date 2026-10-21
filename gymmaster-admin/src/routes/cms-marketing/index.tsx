import { useEffect, useRef, useState } from 'react';
import {
  App,
  Badge,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  List,
  Modal,
  Popconfirm,
  Rate,
  Row,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import {
  CalendarOutlined,
  CheckOutlined,
  CloseOutlined,
  DeleteOutlined,
  EditOutlined,
  MessageOutlined,
  PlusOutlined,
  SendOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  cmsSystemApi,
  type ContactMessageItem,
  type PostItem,
  type PublicTrainerItem,
  type TestimonialItem,
} from '@/api/cms-system';
import { appConfig } from '@/config';
import { PERM, ROLE } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { showError } from '@/utils/errors';
import { formatDateTime } from '@/utils/format';
import { chatSendState } from './chatAccess';

const POST_CATEGORY_LABELS: Record<string, string> = {
  TAP_LUYEN: 'Tập luyện',
  DINH_DUONG: 'Dinh dưỡng',
  SU_KIEN: 'Sự kiện',
};

/** Origin của API (vd http://localhost:8000) – link .ics do API phục vụ, không phải origin admin. */
function apiOrigin(): string {
  try {
    return new URL(appConfig.apiUrl).origin;
  } catch {
    return window.location.origin;
  }
}

export default function CmsMarketingPage() {
  const qc = useQueryClient();
  const { message } = App.useApp();
  const { can, roles, user } = usePermission();
  // Khớp quyền API: bài viết/cảm nhận cần cms.manage; hồ sơ HLV thêm staff.view;
  // liên hệ thêm lead.view.
  const canCms = can(PERM.CMS_MANAGE);
  const canTrainers = can([PERM.CMS_MANAGE, PERM.STAFF_VIEW]);
  const canContacts = can([PERM.CMS_MANAGE, PERM.LEAD_VIEW]);
  // API chỉ trả hội thoại cho OWNER/MANAGER (giám sát) và TRAINER (của chính mình).
  const canMessages = [ROLE.OWNER, ROLE.MANAGER, ROLE.TRAINER].some((r) => roles.includes(r));
  const [postModal, setPostModal] = useState(false);
  const [editingPost, setEditingPost] = useState<PostItem | null>(null);
  const [postForm] = Form.useForm();

  const [trainerModal, setTrainerModal] = useState(false);
  const [trainerForm] = Form.useForm();

  const [selectedConvId, setSelectedConvId] = useState<number | null>(null);
  const [chatText, setChatText] = useState('');

  const postsQuery = useQuery({
    queryKey: ['admin-cms-posts'],
    queryFn: () => cmsSystemApi.listPosts(),
    enabled: canCms,
  });
  const trainersQuery = useQuery({
    queryKey: ['admin-cms-trainers'],
    queryFn: () => cmsSystemApi.listTrainerProfiles(),
    enabled: canTrainers,
  });
  const testimonialsQuery = useQuery({
    queryKey: ['admin-cms-testimonials'],
    queryFn: () => cmsSystemApi.listTestimonials(),
    enabled: canCms,
  });
  const contactsQuery = useQuery({
    queryKey: ['admin-cms-contacts'],
    queryFn: () => cmsSystemApi.listContactMessages(),
    enabled: canContacts,
  });
  const convsQuery = useQuery({
    queryKey: ['admin-pt-conversations'],
    queryFn: () => cmsSystemApi.listConversations(),
    enabled: canMessages,
  });
  const activeConvId = selectedConvId ?? convsQuery.data?.[0]?.id ?? null;
  const activeConv = convsQuery.data?.find((c) => c.id === activeConvId) ?? null;
  // OWNER/MANAGER chỉ xem giám sát – API chỉ cho HLV phụ trách (hợp đồng ACTIVE) gửi tin.
  const sendState = chatSendState(activeConv, user?.id);

  const messagesQuery = useQuery({
    queryKey: ['admin-pt-messages', activeConvId],
    queryFn: () => cmsSystemApi.listMessages(activeConvId!),
    enabled: !!activeConvId,
  });

  // Tự cuộn xuống tin nhắn mới nhất khi mở hội thoại / có tin mới.
  const chatBoxRef = useRef<HTMLDivElement>(null);
  const messageCount = messagesQuery.data?.length ?? 0;
  useEffect(() => {
    const el = chatBoxRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [activeConvId, messageCount]);

  const calTokenQuery = useQuery({
    queryKey: ['admin-calendar-token'],
    queryFn: () => cmsSystemApi.getCalendarToken(),
    enabled: canMessages,
  });

  const savePostMut = useMutation({
    mutationFn: (values: Record<string, unknown>) =>
      editingPost
        ? cmsSystemApi.updatePost(editingPost.id, values)
        : cmsSystemApi.createPost(values as { slug: string; title: string; content: string }),
    onSuccess: () => {
      message.success('Đã lưu bài viết');
      setPostModal(false);
      setEditingPost(null);
      postForm.resetFields();
      qc.invalidateQueries({ queryKey: ['admin-cms-posts'] });
    },
  });

  const deletePostMut = useMutation({
    mutationFn: (id: number) => cmsSystemApi.deletePost(id),
    onSuccess: () => {
      message.success('Đã xóa bài viết');
      qc.invalidateQueries({ queryKey: ['admin-cms-posts'] });
    },
  });

  const saveTrainerMut = useMutation({
    mutationFn: (values: Record<string, unknown>) =>
      cmsSystemApi.saveTrainerProfile(values as { user_id: number; slug: string }),
    onSuccess: () => {
      message.success('Đã lưu hồ sơ HLV công khai');
      setTrainerModal(false);
      trainerForm.resetFields();
      qc.invalidateQueries({ queryKey: ['admin-cms-trainers'] });
    },
  });

  const approveTestiMut = useMutation({
    mutationFn: (id: number) => cmsSystemApi.approveTestimonial(id),
    onSuccess: () => {
      message.success('Đã duyệt hiển thị cảm nhận');
      qc.invalidateQueries({ queryKey: ['admin-cms-testimonials'] });
    },
  });

  const rejectTestiMut = useMutation({
    mutationFn: (id: number) => cmsSystemApi.rejectTestimonial(id),
    onSuccess: () => {
      message.success('Đã từ chối cảm nhận');
      qc.invalidateQueries({ queryKey: ['admin-cms-testimonials'] });
    },
  });

  const updateContactMut = useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) =>
      cmsSystemApi.updateContactMessage(id, { status }),
    onSuccess: () => {
      message.success('Đã cập nhật trạng thái liên hệ');
      qc.invalidateQueries({ queryKey: ['admin-cms-contacts'] });
    },
  });

  const sendMsgMut = useMutation({
    mutationFn: () => cmsSystemApi.sendMessage(activeConvId!, chatText),
    onSuccess: () => {
      setChatText('');
      qc.invalidateQueries({ queryKey: ['admin-pt-messages', activeConvId] });
      qc.invalidateQueries({ queryKey: ['admin-pt-conversations'] });
    },
    onError: (e) => showError(e, 'Không thể gửi tin nhắn'),
  });

  return (
    <Card
      title="Quản trị Nội dung Landing, Cảm nhận Hội viên & Tin nhắn PT"
      extra={
        calTokenQuery.data && (
          <Button
            icon={<CalendarOutlined />}
            onClick={() => {
              navigator.clipboard?.writeText(apiOrigin() + calTokenQuery.data!.ics_url);
              message.success('Đã sao chép đường dẫn đồng bộ lịch dạy .ics');
            }}
          >
            Sao chép link Lịch .ics
          </Button>
        )
      }
    >
      <Tabs
        items={[
          canCms && {
            key: 'posts',
            label: 'Bài viết Tin tức & Blog',
            children: (
              <>
                <div style={{ marginBottom: 16, textAlign: 'right' }}>
                  <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={() => {
                      setEditingPost(null);
                      postForm.resetFields();
                      setPostModal(true);
                    }}
                  >
                    Viết bài mới
                  </Button>
                </div>
                <Table<PostItem>
                  rowKey="id"
                  loading={postsQuery.isLoading}
                  dataSource={postsQuery.data ?? []}
                  columns={[
                    { title: 'Tiêu đề', dataIndex: 'title' },
                    { title: 'Slug', dataIndex: 'slug' },
                    {
                      title: 'Chuyên mục',
                      dataIndex: 'category',
                      render: (v: string) => <Tag color="blue">{POST_CATEGORY_LABELS[v] ?? v}</Tag>,
                    },
                    {
                      title: 'Trạng thái',
                      dataIndex: 'status',
                      render: (s: string) => (
                        <Tag color={s === 'PUBLISHED' ? 'green' : 'default'}>
                          {s === 'PUBLISHED' ? 'Đã xuất bản' : 'Bản nháp'}
                        </Tag>
                      ),
                    },
                    {
                      title: 'Ngày tạo',
                      dataIndex: 'created_at',
                      render: (v: string) => formatDateTime(v),
                    },
                    {
                      title: 'Thao tác',
                      render: (_, r) => (
                        <Space>
                          <Button
                            size="small"
                            icon={<EditOutlined />}
                            onClick={() => {
                              setEditingPost(r);
                              postForm.setFieldsValue(r);
                              setPostModal(true);
                            }}
                          />
                          <Popconfirm
                            title="Xóa bài viết này?"
                            okText="Xóa"
                            cancelText="Hủy"
                            okButtonProps={{ danger: true }}
                            onConfirm={() => deletePostMut.mutate(r.id)}
                          >
                            <Button size="small" danger icon={<DeleteOutlined />} />
                          </Popconfirm>
                        </Space>
                      ),
                    },
                  ]}
                />
              </>
            ),
          },
          canTrainers && {
            key: 'trainers',
            label: 'Hồ sơ HLV Công khai',
            children: (
              <>
                {canCms && (
                  <div style={{ marginBottom: 16, textAlign: 'right' }}>
                    <Button
                      type="primary"
                      icon={<PlusOutlined />}
                      onClick={() => {
                        trainerForm.resetFields();
                        setTrainerModal(true);
                      }}
                    >
                      Cập nhật Hồ sơ HLV
                    </Button>
                  </div>
                )}
                <Table<PublicTrainerItem>
                  rowKey="id"
                  scroll={{ x: 900 }}
                  loading={trainersQuery.isLoading}
                  dataSource={trainersQuery.data ?? []}
                  columns={[
                    { title: 'Họ tên HLV', dataIndex: 'full_name' },
                    { title: 'Slug', dataIndex: 'slug' },
                    { title: 'Chức danh nổi bật', dataIndex: 'headline' },
                    { title: 'Chuyên môn', dataIndex: 'specialties' },
                    {
                      title: 'Kinh nghiệm',
                      dataIndex: 'experience_years',
                      render: (v: number) => `${v} năm`,
                    },
                    {
                      title: 'Đánh giá',
                      render: (_, r) => (
                        <span>
                          {r.rating_count > 0
                            ? `⭐ ${(r.rating_avg ?? 0).toFixed(1)} (${r.rating_count} lượt)`
                            : 'Chưa có đánh giá'}
                        </span>
                      ),
                    },
                    {
                      title: 'Hiển thị',
                      dataIndex: 'is_published',
                      render: (v: boolean) => (
                        <Tag color={v ? 'green' : 'default'}>{v ? 'Công khai' : 'Ẩn'}</Tag>
                      ),
                    },
                  ]}
                />
              </>
            ),
          },
          canCms && {
            key: 'testimonials',
            label: 'Cảm nhận Hội viên',
            children: (
              <Table<TestimonialItem>
                rowKey="id"
                loading={testimonialsQuery.isLoading}
                dataSource={testimonialsQuery.data ?? []}
                columns={[
                  { title: 'Hội viên', dataIndex: 'author_name' },
                  { title: 'Vai trò / Gói', dataIndex: 'author_role' },
                  {
                    title: 'Số sao',
                    dataIndex: 'rating',
                    render: (v: number) => <Rate disabled value={v} />,
                  },
                  { title: 'Nội dung cảm nhận', dataIndex: 'content' },
                  { title: 'Kết quả đạt được', dataIndex: 'before_after_note' },
                  {
                    title: 'Trạng thái',
                    dataIndex: 'status',
                    render: (s: string) => (
                      <Tag color={s === 'APPROVED' ? 'green' : s === 'REJECTED' ? 'red' : 'gold'}>
                        {s === 'APPROVED' ? 'Đã duyệt' : s === 'REJECTED' ? 'Từ chối' : 'Chờ duyệt'}
                      </Tag>
                    ),
                  },
                  {
                    title: 'Thao tác',
                    render: (_, r) => (
                      <Space>
                        {r.status !== 'APPROVED' && (
                          <Button
                            size="small"
                            type="primary"
                            icon={<CheckOutlined />}
                            onClick={() => approveTestiMut.mutate(r.id)}
                          >
                            Duyệt
                          </Button>
                        )}
                        {r.status !== 'REJECTED' && (
                          <Button
                            size="small"
                            danger
                            icon={<CloseOutlined />}
                            onClick={() => rejectTestiMut.mutate(r.id)}
                          >
                            Từ chối
                          </Button>
                        )}
                      </Space>
                    ),
                  },
                ]}
              />
            ),
          },
          canContacts && {
            key: 'contacts',
            label: 'Liên hệ từ Website',
            children: (
              <Table<ContactMessageItem>
                rowKey="id"
                loading={contactsQuery.isLoading}
                dataSource={contactsQuery.data ?? []}
                columns={[
                  { title: 'Họ tên khách', dataIndex: 'full_name' },
                  { title: 'Số điện thoại', dataIndex: 'phone' },
                  { title: 'Email', dataIndex: 'email' },
                  { title: 'Chủ đề', dataIndex: 'subject' },
                  { title: 'Lời nhắn', dataIndex: 'message' },
                  {
                    title: 'Trạng thái',
                    dataIndex: 'status',
                    render: (s: string, r) => (
                      <Select
                        size="small"
                        value={s}
                        style={{ width: 130 }}
                        onChange={(val) => updateContactMut.mutate({ id: r.id, status: val })}
                        options={[
                          { value: 'NEW', label: 'Mới tiếp nhận' },
                          { value: 'CONTACTED', label: 'Đã liên hệ' },
                          { value: 'CLOSED', label: 'Đã đóng' },
                        ]}
                      />
                    ),
                  },
                ]}
              />
            ),
          },
          canMessages && {
            key: 'messages',
            label: 'Tin nhắn HLV ↔ Hội viên',
            children: (
              <Row gutter={16}>
                <Col xs={24} md={8}>
                  <Card size="small" title="Danh sách Hợp đồng PT">
                    <List
                      dataSource={convsQuery.data ?? []}
                      renderItem={(item) => (
                        <List.Item
                          style={{
                            cursor: 'pointer',
                            background: item.id === activeConvId ? 'var(--gm-unread)' : undefined,
                            padding: '8px 12px',
                          }}
                          onClick={() => setSelectedConvId(item.id)}
                        >
                          <List.Item.Meta
                            avatar={
                              <Badge count={item.unread_count}>
                                <MessageOutlined style={{ fontSize: 20 }} />
                              </Badge>
                            }
                            title={`${item.member_name} (${item.member_code})`}
                            description={`HLV: ${item.trainer_name} · Còn ${item.remaining_sessions} buổi`}
                          />
                        </List.Item>
                      )}
                    />
                  </Card>
                </Col>
                <Col xs={24} md={16}>
                  <Card size="small" title="Nội dung trao đổi trực tuyến (SSE Realtime)">
                    <div
                      ref={chatBoxRef}
                      style={{
                        height: 320,
                        overflowY: 'auto',
                        padding: 12,
                        background: 'var(--gm-hover)',
                        borderRadius: 8,
                        marginBottom: 12,
                      }}
                    >
                      {(messagesQuery.data ?? []).map((m) => (
                        <div key={m.id} style={{ marginBottom: 10 }}>
                          <Typography.Text strong>{m.sender_name}</Typography.Text>{' '}
                          <Tag>{m.sender_role}</Tag>{' '}
                          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                            {formatDateTime(m.created_at)}
                          </Typography.Text>
                          <div
                            style={{
                              background: 'var(--gm-surface)',
                              padding: '8px 12px',
                              borderRadius: 6,
                              border: '1px solid var(--gm-border)',
                              marginTop: 4,
                            }}
                          >
                            {m.content}
                          </div>
                        </div>
                      ))}
                    </div>
                    <Space.Compact style={{ width: '100%' }}>
                      <Input
                        placeholder="Nhập lời nhắn hướng dẫn tập luyện / dinh dưỡng cho học viên..."
                        value={chatText}
                        onChange={(e) => setChatText(e.target.value)}
                        onPressEnter={() =>
                          sendState.canSend &&
                          chatText.trim() &&
                          !sendMsgMut.isPending &&
                          sendMsgMut.mutate()
                        }
                        disabled={!sendState.canSend}
                      />
                      <Button
                        type="primary"
                        icon={<SendOutlined />}
                        loading={sendMsgMut.isPending}
                        disabled={!sendState.canSend || !chatText.trim()}
                        onClick={() => sendMsgMut.mutate()}
                      >
                        Gửi
                      </Button>
                    </Space.Compact>
                    {sendState.reason && activeConvId && (
                      <Typography.Text
                        type="secondary"
                        style={{ display: 'block', marginTop: 8, fontSize: 12 }}
                      >
                        {sendState.reason}
                      </Typography.Text>
                    )}
                  </Card>
                </Col>
              </Row>
            ),
          },
        ].filter((t): t is Exclude<typeof t, false> => !!t)}
      />

      <Modal
        title={editingPost ? 'Cập nhật bài viết Blog' : 'Tạo bài viết Blog mới'}
        open={postModal}
        onCancel={() => setPostModal(false)}
        onOk={() => postForm.submit()}
        confirmLoading={savePostMut.isPending}
        okText="Lưu bài viết"
        cancelText="Hủy"
        width={680}
      >
        <Form form={postForm} layout="vertical" onFinish={(v) => savePostMut.mutate(v)}>
          <Form.Item
            name="title"
            label="Tiêu đề bài viết"
            rules={[
              { required: true, whitespace: true, message: 'Vui lòng nhập tiêu đề bài viết' },
            ]}
          >
            <Input />
          </Form.Item>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item
                name="slug"
                label="Đường dẫn tĩnh (Slug)"
                rules={[
                  { required: true, whitespace: true, message: 'Vui lòng nhập đường dẫn tĩnh' },
                ]}
              >
                <Input placeholder="vd: lo-trinh-tang-co-cho-nguoi-moi" />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item name="category" label="Chuyên mục" initialValue="TAP_LUYEN">
                <Select
                  options={[
                    { value: 'TAP_LUYEN', label: 'Tập luyện' },
                    { value: 'DINH_DUONG', label: 'Dinh dưỡng' },
                    { value: 'SU_KIEN', label: 'Sự kiện' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item name="status" label="Trạng thái" initialValue="PUBLISHED">
                <Select
                  options={[
                    { value: 'PUBLISHED', label: 'Xuất bản' },
                    { value: 'DRAFT', label: 'Bản nháp' },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="excerpt" label="Tóm tắt ngắn">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item
            name="content"
            label="Nội dung chi tiết"
            rules={[
              { required: true, whitespace: true, message: 'Vui lòng nhập nội dung bài viết' },
            ]}
          >
            <Input.TextArea rows={6} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="Cập nhật Hồ sơ Huấn luyện viên Công khai"
        open={trainerModal}
        onCancel={() => setTrainerModal(false)}
        onOk={() => trainerForm.submit()}
        confirmLoading={saveTrainerMut.isPending}
        okText="Lưu hồ sơ"
        cancelText="Hủy"
      >
        <Form form={trainerForm} layout="vertical" onFinish={(v) => saveTrainerMut.mutate(v)}>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item
                name="user_id"
                label="ID Tài khoản HLV"
                rules={[{ required: true, message: 'Vui lòng nhập ID tài khoản HLV' }]}
              >
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="slug"
                label="Slug hồ sơ"
                rules={[{ required: true, whitespace: true, message: 'Vui lòng nhập slug hồ sơ' }]}
              >
                <Input placeholder="hlv-nguyen-minh-quan" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="headline" label="Tiêu đề giới thiệu">
            <Input placeholder="Chuyên gia Hình thể & Phục hồi Chấn thương" />
          </Form.Item>
          <Form.Item name="specialties" label="Chuyên môn">
            <Input placeholder="Hypertrophy, Giảm mỡ, Mobility" />
          </Form.Item>
          <Form.Item name="certifications" label="Chứng chỉ quốc tế">
            <Input placeholder="NASM-CPT, ISSA Nutrition" />
          </Form.Item>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item name="experience_years" label="Số năm KN" initialValue={5}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="featured"
                label="Nổi bật trang chủ"
                valuePropName="checked"
                initialValue={true}
              >
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="is_published"
                label="Hiển thị công khai"
                valuePropName="checked"
                initialValue={true}
              >
                <Switch />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="bio" label="Tiểu sử chi tiết">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
