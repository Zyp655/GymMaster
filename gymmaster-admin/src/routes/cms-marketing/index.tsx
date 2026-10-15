import { useState } from 'react';
import {
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
  message,
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
import { formatDateTime } from '@/utils/format';

export default function CmsMarketingPage() {
  const qc = useQueryClient();
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
  });
  const trainersQuery = useQuery({
    queryKey: ['admin-cms-trainers'],
    queryFn: () => cmsSystemApi.listTrainerProfiles(),
  });
  const testimonialsQuery = useQuery({
    queryKey: ['admin-cms-testimonials'],
    queryFn: () => cmsSystemApi.listTestimonials(),
  });
  const contactsQuery = useQuery({
    queryKey: ['admin-cms-contacts'],
    queryFn: () => cmsSystemApi.listContactMessages(),
  });
  const convsQuery = useQuery({
    queryKey: ['admin-pt-conversations'],
    queryFn: () => cmsSystemApi.listConversations(),
  });
  const activeConvId = selectedConvId ?? convsQuery.data?.[0]?.id ?? null;

  const messagesQuery = useQuery({
    queryKey: ['admin-pt-messages', activeConvId],
    queryFn: () => cmsSystemApi.listMessages(activeConvId!),
    enabled: !!activeConvId,
  });

  const calTokenQuery = useQuery({
    queryKey: ['admin-calendar-token'],
    queryFn: () => cmsSystemApi.getCalendarToken(),
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
  });

  return (
    <Card
      title="Quản trị Nội dung Landing, Cảm nhận Hội viên & Tin nhắn PT (Sprint 29)"
      extra={
        calTokenQuery.data && (
          <Button
            icon={<CalendarOutlined />}
            onClick={() => {
              navigator.clipboard?.writeText(window.location.origin + calTokenQuery.data!.ics_url);
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
          {
            key: 'posts',
            label: '1. Bài viết Tin tức & Blog',
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
                      render: (v: string) => <Tag color="blue">{v}</Tag>,
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
          {
            key: 'trainers',
            label: '2. Hồ sơ HLV Công khai',
            children: (
              <>
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
                <Table<PublicTrainerItem>
                  rowKey="id"
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
                          ⭐ {r.rating_avg?.toFixed(1) || '5.0'} ({r.rating_count} lượt)
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
          {
            key: 'testimonials',
            label: '3. Cảm nhận Hội viên',
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
                      <Tag
                        color={
                          s === 'APPROVED' ? 'green' : s === 'REJECTED' ? 'red' : 'gold'
                        }
                      >
                        {s === 'APPROVED'
                          ? 'Đã duyệt'
                          : s === 'REJECTED'
                            ? 'Từ chối'
                            : 'Chờ duyệt'}
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
          {
            key: 'contacts',
            label: '4. Liên hệ từ Website',
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
          {
            key: 'messages',
            label: '5. Tin nhắn HLV ↔ Hội viên',
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
                            background: item.id === activeConvId ? '#fff7ed' : undefined,
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
                      style={{
                        height: 320,
                        overflowY: 'auto',
                        padding: 12,
                        background: '#fafafa',
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
                              background: '#fff',
                              padding: '8px 12px',
                              borderRadius: 6,
                              border: '1px solid #f0f0f0',
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
                        onPressEnter={() => chatText.trim() && sendMsgMut.mutate()}
                        disabled={!activeConvId}
                      />
                      <Button
                        type="primary"
                        icon={<SendOutlined />}
                        loading={sendMsgMut.isPending}
                        disabled={!activeConvId || !chatText.trim()}
                        onClick={() => sendMsgMut.mutate()}
                      >
                        Gửi
                      </Button>
                    </Space.Compact>
                  </Card>
                </Col>
              </Row>
            ),
          },
        ]}
      />

      <Modal
        title={editingPost ? 'Cập nhật bài viết Blog' : 'Tạo bài viết Blog mới'}
        open={postModal}
        onCancel={() => setPostModal(false)}
        onOk={() => postForm.submit()}
        confirmLoading={savePostMut.isPending}
        width={680}
      >
        <Form form={postForm} layout="vertical" onFinish={(v) => savePostMut.mutate(v)}>
          <Form.Item name="title" label="Tiêu đề bài viết" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name="slug" label="Đường dẫn tĩnh (Slug)" rules={[{ required: true }]}>
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
          <Form.Item name="content" label="Nội dung chi tiết" rules={[{ required: true }]}>
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
      >
        <Form form={trainerForm} layout="vertical" onFinish={(v) => saveTrainerMut.mutate(v)}>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name="user_id" label="ID Tài khoản HLV" rules={[{ required: true }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="slug" label="Slug hồ sơ" rules={[{ required: true }]}>
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
              <Form.Item name="featured" label="Nổi bật trang chủ" valuePropName="checked" initialValue={true}>
                <Switch />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="is_published" label="Hiển thị công khai" valuePropName="checked" initialValue={true}>
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
