import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  App,
  Button,
  Card,
  Checkbox,
  Col,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Modal,
  Progress,
  Row,
  Segmented,
  Select,
  Space,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  AppleOutlined,
  DeleteOutlined,
  EditOutlined,
  FrownOutlined,
  HeartOutlined,
  MehOutlined,
  MinusCircleOutlined,
  NotificationOutlined,
  PlusOutlined,
  PushpinOutlined,
  ReloadOutlined,
  SmileOutlined,
  StarOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import {
  engagementApi,
  MEAL_TYPE_OPTIONS,
} from '@/api/loyalty-engagement';
import type {
  Announcement,
  AnnouncementInput,
  Food,
  FoodInput,
  MealPlan,
  MealPlanInput,
  MealPlanItem,
  Survey,
  SurveyInput,
} from '@/api/loyalty-engagement';
import ActionBar from '@/components/ActionBar';
import BranchSelect from '@/components/BranchSelect';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MemberSelect from '@/components/MemberSelect';
import PageTitle from '@/components/PageTitle';
import StatCard from '@/components/StatCard';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';
import { DATE_FORMAT, formatDate, formatDateTime, formatNumber, toApiDate } from '@/utils/format';

/* --------------------------------------------------------------------------
 * TAB 1: Thực đơn dinh dưỡng & Món ăn (/admin/meal-plans, /admin/foods)
 * -------------------------------------------------------------------------- */
function NutritionAndFoodsTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const [subView, setSubView] = useState<'plans' | 'foods'>('plans');

  const [planModal, setPlanModal] = useState<{ open: boolean; record: MealPlan | null }>({
    open: false,
    record: null,
  });
  const [foodModal, setFoodModal] = useState<{ open: boolean; record: Food | null }>({
    open: false,
    record: null,
  });

  const [planForm] = Form.useForm<{
    member_id: number;
    name: string;
    goal?: string;
    target_calories: number;
    start_date?: Dayjs | null;
    end_date?: Dayjs | null;
    items?: MealPlanItem[];
  }>();
  const [foodForm] = Form.useForm<FoodInput>();

  const foodsListQuery = useQuery({
    queryKey: [engagementApi.foodKey, 'all'],
    queryFn: () => engagementApi.listFoods({ limit: 200 }),
  });

  const plansTable = useTableQuery<MealPlan, { page?: number; limit?: number }>({
    queryKey: [engagementApi.mealPlanKey, 'list'],
    queryFn: (p) => engagementApi.listMealPlans(p),
    defaults: { order_by: 'id', order_dir: 'DESC' },
  });

  const foodsTable = useTableQuery<Food, { page?: number; limit?: number }>({
    queryKey: [engagementApi.foodKey, 'list'],
    queryFn: (p) => engagementApi.listFoods(p),
    defaults: { order_by: 'id', order_dir: 'ASC' },
  });

  useEffect(() => {
    if (!planModal.open) return;
    planForm.resetFields();
    if (planModal.record) {
      planForm.setFieldsValue({
        member_id: planModal.record.member_id,
        name: planModal.record.name,
        goal: planModal.record.goal,
        target_calories: planModal.record.target_calories || 2000,
        start_date: planModal.record.start_date ? dayjs(planModal.record.start_date) : null,
        end_date: planModal.record.end_date ? dayjs(planModal.record.end_date) : null,
        items: planModal.record.items ?? [],
      });
    } else {
      planForm.setFieldsValue({
        target_calories: 2000,
        start_date: dayjs(),
        end_date: dayjs().add(30, 'day'),
        items: [{ day_index: 1, meal: 'BREAKFAST', food_id: foodsListQuery.data?.items?.[0]?.id ?? 1, qty: 1 }],
      });
    }
  }, [planModal, planForm, foodsListQuery.data]);

  useEffect(() => {
    if (!foodModal.open) return;
    foodForm.resetFields();
    if (foodModal.record) {
      foodForm.setFieldsValue(foodModal.record);
    } else {
      foodForm.setFieldsValue({
        unit: '100g',
        calories: 150,
        protein: 20,
        carbs: 10,
        fat: 5,
      });
    }
  }, [foodModal, foodForm]);

  const savePlanMut = useMutation({
    mutationFn: (v: {
      member_id: number;
      name: string;
      goal?: string;
      target_calories: number;
      start_date?: Dayjs | null;
      end_date?: Dayjs | null;
      items?: MealPlanItem[];
    }) => {
      const payload: MealPlanInput = {
        member_id: v.member_id,
        name: v.name.trim(),
        goal: v.goal?.trim() || '',
        target_calories: Number(v.target_calories) || 2000,
        start_date: toApiDate(v.start_date) ?? null,
        end_date: toApiDate(v.end_date) ?? null,
        status: 'ACTIVE',
        items: (v.items ?? []).map((it) => ({
          day_index: Number(it.day_index) || 1,
          meal: it.meal || 'BREAKFAST',
          food_id: Number(it.food_id),
          qty: Number(it.qty) || 1,
          note: it.note?.trim() || '',
        })),
      };
      return planModal.record
        ? engagementApi.updateMealPlan(planModal.record.id, payload)
        : engagementApi.createMealPlan(payload);
    },
    onSuccess: () => {
      message.success(planModal.record ? 'Đã cập nhật thực đơn' : 'Đã giao thực đơn dinh dưỡng mới');
      void qc.invalidateQueries({ queryKey: [engagementApi.mealPlanKey] });
      setPlanModal({ open: false, record: null });
    },
    onError: (err) => showError(err, 'Không thể lưu thực đơn dinh dưỡng'),
  });

  const deletePlanMut = useMutation({
    mutationFn: (id: number) => engagementApi.deleteMealPlan(id),
    onSuccess: () => {
      message.success('Đã xóa thực đơn dinh dưỡng');
      void qc.invalidateQueries({ queryKey: [engagementApi.mealPlanKey] });
    },
    onError: (err) => showError(err, 'Không thể xóa thực đơn'),
  });

  const saveFoodMut = useMutation({
    mutationFn: (v: FoodInput) => {
      const payload: FoodInput = {
        name: v.name.trim(),
        unit: v.unit.trim(),
        calories: Number(v.calories) || 0,
        protein: Number(v.protein) || 0,
        carbs: Number(v.carbs) || 0,
        fat: Number(v.fat) || 0,
      };
      return foodModal.record
        ? engagementApi.updateFood(foodModal.record.id, payload)
        : engagementApi.createFood(payload);
    },
    onSuccess: () => {
      message.success(foodModal.record ? 'Đã cập nhật món ăn' : 'Đã thêm món ăn vào thư viện');
      void qc.invalidateQueries({ queryKey: [engagementApi.foodKey] });
      setFoodModal({ open: false, record: null });
    },
    onError: (err) => showError(err, 'Không thể lưu món ăn'),
  });

  const deleteFoodMut = useMutation({
    mutationFn: (id: number) => engagementApi.deleteFood(id),
    onSuccess: () => {
      message.success('Đã xóa món ăn');
      void qc.invalidateQueries({ queryKey: [engagementApi.foodKey] });
    },
    onError: (err) => showError(err, 'Không thể xóa món ăn'),
  });

  const planColumns = useMemo<ColumnsType<MealPlan>>(
    () => [
      {
        title: 'Tên thực đơn',
        dataIndex: 'name',
        render: (name: string, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>{name}</Typography.Text>
            {r.goal && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                Mục tiêu: {r.goal}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Hội viên áp dụng',
        key: 'member',
        render: (_, r) => (
          <Space>
            <Typography.Text strong>
              {r.member_name || `Hội viên #${r.member_id}`}
            </Typography.Text>
            {r.member_code && <Tag className="code-text">{r.member_code}</Tag>}
          </Space>
        ),
      },
      {
        title: 'Calo mục tiêu / ngày',
        dataIndex: 'target_calories',
        align: 'right',
        render: (v: number) => <Tag color="volcano">{formatNumber(v)} kcal/ngày</Tag>,
      },
      {
        title: 'Thời gian áp dụng',
        key: 'dates',
        render: (_, r) =>
          r.start_date ? `${formatDate(r.start_date)} → ${formatDate(r.end_date)}` : '—',
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        render: (s: string) => <StatusTag status={s} />,
      },
      {
        title: '',
        key: 'actions',
        width: 100,
        render: (_, r) =>
          canManage ? (
            <Space size={4}>
              <Button
                type="text"
                size="small"
                icon={<EditOutlined />}
                onClick={() => setPlanModal({ open: true, record: r })}
              />
              <ConfirmButton
                perm={PERM.ENGAGEMENT_MANAGE}
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
                confirmTitle={`Xóa thực đơn "${r.name}"?`}
                onConfirm={() => deletePlanMut.mutateAsync(r.id)}
                aria-label="Xóa"
              />
            </Space>
          ) : null,
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canManage],
  );

  const foodColumns = useMemo<ColumnsType<Food>>(
    () => [
      {
        title: 'Tên thực phẩm / Món ăn',
        dataIndex: 'name',
        render: (v: string) => <Typography.Text strong>{v}</Typography.Text>,
      },
      {
        title: 'Khẩu phần chuẩn',
        dataIndex: 'unit',
        width: 120,
      },
      {
        title: 'Năng lượng (Calories)',
        dataIndex: 'calories',
        align: 'right',
        render: (v: number) => <Tag color="orange">{v} kcal</Tag>,
      },
      {
        title: 'Đạm (Protein)',
        dataIndex: 'protein',
        align: 'right',
        render: (v: number) => `${v} g`,
      },
      {
        title: 'Tinh bột (Carbs)',
        dataIndex: 'carbs',
        align: 'right',
        render: (v: number) => `${v} g`,
      },
      {
        title: 'Chất béo (Fat)',
        dataIndex: 'fat',
        align: 'right',
        render: (v: number) => `${v} g`,
      },
      {
        title: '',
        key: 'actions',
        width: 100,
        render: (_, r) =>
          canManage ? (
            <Space size={4}>
              <Button
                type="text"
                size="small"
                icon={<EditOutlined />}
                onClick={() => setFoodModal({ open: true, record: r })}
              />
              <ConfirmButton
                perm={PERM.ENGAGEMENT_MANAGE}
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
                confirmTitle={`Xóa món ăn "${r.name}"?`}
                onConfirm={() => deleteFoodMut.mutateAsync(r.id)}
                aria-label="Xóa"
              />
            </Space>
          ) : null,
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canManage],
  );

  return (
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col>
          <Segmented
            value={subView}
            onChange={(v) => setSubView(v as 'plans' | 'foods')}
            options={[
              { label: 'Thực đơn dinh dưỡng hội viên (/admin/meal-plans)', value: 'plans' },
              { label: 'Thư viện món ăn & Macro (/admin/foods)', value: 'foods' },
            ]}
          />
        </Col>
        <Col>
          {subView === 'plans' ? (
            <ActionBar
              onAdd={() => setPlanModal({ open: true, record: null })}
              addPerm={PERM.ENGAGEMENT_MANAGE}
              addText="Soạn thực đơn mới"
              onRefresh={() => plansTable.refetch()}
              refreshing={plansTable.isFetching}
            />
          ) : (
            <ActionBar
              onAdd={() => setFoodModal({ open: true, record: null })}
              addPerm={PERM.ENGAGEMENT_MANAGE}
              addText="Thêm món ăn mới"
              onRefresh={() => foodsTable.refetch()}
              refreshing={foodsTable.isFetching}
            />
          )}
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder={
          subView === 'plans' ? 'Tìm theo tên thực đơn, hội viên...' : 'Tìm theo tên món ăn...'
        }
        colTextSearch={6}
      />

      {subView === 'plans' ? (
        <DataTable<MealPlan>
          table={plansTable}
          columns={planColumns}
          emptyText="Chưa có thực đơn dinh dưỡng nào được giao"
        />
      ) : (
        <DataTable<Food>
          table={foodsTable}
          columns={foodColumns}
          emptyText="Chưa có món ăn nào trong thư viện"
        />
      )}

      {/* Modal tạo thực đơn */}
      <Modal
        open={planModal.open}
        title={planModal.record ? `Sửa thực đơn: ${planModal.record.name}` : 'Soạn thực đơn dinh dưỡng cho Hội viên'}
        okText="Lưu thực đơn"
        cancelText="Hủy"
        onOk={() => planForm.submit()}
        onCancel={() => setPlanModal({ open: false, record: null })}
        confirmLoading={savePlanMut.isPending}
        width={680}
        forceRender
      >
        <Form
          form={planForm}
          layout="vertical"
          onFinish={(v) => savePlanMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Row gutter={12}>
            <Col xs={24} sm={12}>
              <Form.Item
                name="member_id"
                label="Hội viên"
                rules={[{ required: true, message: 'Vui lòng chọn hội viên' }]}
              >
                <MemberSelect />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="name"
                label="Tên thực đơn"
                rules={[{ required: true, whitespace: true, message: 'Nhập tên thực đơn' }]}
              >
                <Input placeholder="VD: Thực đơn Eat Clean Giảm mỡ 1800 kcal" />
              </Form.Item>
            </Col>
            <Col xs={12} sm={8}>
              <Form.Item name="target_calories" label="Mục tiêu Calo / ngày" rules={[{ required: true }]}>
                <InputNumber min={800} max={5000} step={100} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={12} sm={8}>
              <Form.Item name="start_date" label="Ngày bắt đầu">
                <DatePicker style={{ width: '100%' }} format={DATE_FORMAT} />
              </Form.Item>
            </Col>
            <Col xs={12} sm={8}>
              <Form.Item name="end_date" label="Ngày kết thúc">
                <DatePicker style={{ width: '100%' }} format={DATE_FORMAT} />
              </Form.Item>
            </Col>
            <Col xs={24}>
              <Form.Item name="goal" label="Ghi chú mục tiêu dinh dưỡng">
                <Input placeholder="VD: Giảm 3kg mỡ trong 4 tuần, ưu tiên đạm..." />
              </Form.Item>
            </Col>
          </Row>

          <Typography.Text strong style={{ display: 'block', marginBottom: 8 }}>
            Chi tiết khẩu phần bữa ăn gợi ý:
          </Typography.Text>
          <Form.List name="items">
            {(fields, { add, remove }) => (
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                {fields.map(({ key, name, ...restField }) => (
                  <Row gutter={8} align="middle" key={key}>
                    <Col xs={6}>
                      <Form.Item
                        {...restField}
                        name={[name, 'meal']}
                        style={{ marginBottom: 0 }}
                        rules={[{ required: true }]}
                      >
                        <Select options={MEAL_TYPE_OPTIONS} placeholder="Bữa" />
                      </Form.Item>
                    </Col>
                    <Col xs={11}>
                      <Form.Item
                        {...restField}
                        name={[name, 'food_id']}
                        style={{ marginBottom: 0 }}
                        rules={[{ required: true, message: 'Chọn món' }]}
                      >
                        <Select
                          showSearch
                          optionFilterProp="label"
                          placeholder="Chọn món ăn"
                          options={(foodsListQuery.data?.items ?? []).map((f) => ({
                            value: f.id,
                            label: `${f.name} (${f.calories} kcal/${f.unit})`,
                          }))}
                        />
                      </Form.Item>
                    </Col>
                    <Col xs={5}>
                      <Form.Item
                        {...restField}
                        name={[name, 'qty']}
                        style={{ marginBottom: 0 }}
                        rules={[{ required: true }]}
                      >
                        <InputNumber min={0.5} step={0.5} style={{ width: '100%' }} placeholder="SL" />
                      </Form.Item>
                    </Col>
                    <Col xs={2}>
                      <Button
                        type="text"
                        danger
                        icon={<MinusCircleOutlined />}
                        onClick={() => remove(name)}
                      />
                    </Col>
                  </Row>
                ))}
                <Button
                  type="dashed"
                  block
                  icon={<PlusOutlined />}
                  onClick={() =>
                    add({
                      day_index: 1,
                      meal: 'LUNCH',
                      food_id: foodsListQuery.data?.items?.[0]?.id ?? 1,
                      qty: 1,
                    })
                  }
                >
                  Thêm món vào thực đơn
                </Button>
              </Space>
            )}
          </Form.List>
        </Form>
      </Modal>

      {/* Modal thêm/sửa món ăn */}
      <Modal
        open={foodModal.open}
        title={foodModal.record ? `Sửa món ăn: ${foodModal.record.name}` : 'Thêm món ăn vào thư viện dinh dưỡng'}
        okText="Lưu món ăn"
        cancelText="Hủy"
        onOk={() => foodForm.submit()}
        onCancel={() => setFoodModal({ open: false, record: null })}
        confirmLoading={saveFoodMut.isPending}
        width={500}
        forceRender
      >
        <Form
          form={foodForm}
          layout="vertical"
          onFinish={(v) => saveFoodMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Row gutter={12}>
            <Col xs={16}>
              <Form.Item
                name="name"
                label="Tên món ăn"
                rules={[{ required: true, whitespace: true, message: 'Nhập tên món ăn' }]}
              >
                <Input placeholder="VD: Ức gà áp chảo (100g)" />
              </Form.Item>
            </Col>
            <Col xs={8}>
              <Form.Item
                name="unit"
                label="Đơn vị tính"
                rules={[{ required: true, whitespace: true, message: 'Nhập ĐVT' }]}
              >
                <Input placeholder="100g / 1 bát" />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="calories" label="Năng lượng (kcal)" rules={[{ required: true }]}>
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="protein" label="Chất đạm - Protein (g)" rules={[{ required: true }]}>
                <InputNumber min={0} step={0.5} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="carbs" label="Tinh bột - Carbs (g)" rules={[{ required: true }]}>
                <InputNumber min={0} step={0.5} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="fat" label="Chất béo - Fat (g)" rules={[{ required: true }]}>
                <InputNumber min={0} step={0.5} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </>
  );
}

/* --------------------------------------------------------------------------
 * TAB 2: Bảng tin chi nhánh (/admin/announcements)
 * -------------------------------------------------------------------------- */
function AnnouncementsTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();

  const [modal, setModal] = useState<{ open: boolean; record: Announcement | null }>({
    open: false,
    record: null,
  });
  const [form] = Form.useForm<{
    branch_id?: number | null;
    title: string;
    body: string;
    pinned?: boolean;
    expire_at?: Dayjs | null;
  }>();

  const table = useTableQuery<Announcement, { branch_id?: number; page?: number; limit?: number }>({
    queryKey: [engagementApi.announcementKey, 'list'],
    queryFn: (p) => engagementApi.listAnnouncements(p),
    filterKeys: ['branch_id'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  useEffect(() => {
    if (!modal.open) return;
    form.resetFields();
    if (modal.record) {
      form.setFieldsValue({
        branch_id: modal.record.branch_id ?? undefined,
        title: modal.record.title,
        body: modal.record.body,
        pinned: modal.record.pinned,
        expire_at: modal.record.expire_at ? dayjs(modal.record.expire_at) : null,
      });
    } else {
      form.setFieldsValue({
        branch_id: branchId ?? undefined,
        pinned: false,
      });
    }
  }, [modal, form, branchId]);

  const saveMut = useMutation({
    mutationFn: (v: {
      branch_id?: number | null;
      title: string;
      body: string;
      pinned?: boolean;
      expire_at?: Dayjs | null;
    }) => {
      const payload: AnnouncementInput = {
        branch_id: v.branch_id ?? null,
        title: v.title.trim(),
        body: v.body.trim(),
        pinned: !!v.pinned,
        expire_at: v.expire_at ? v.expire_at.toISOString() : null,
      };
      return modal.record
        ? engagementApi.updateAnnouncement(modal.record.id, payload)
        : engagementApi.createAnnouncement(payload);
    },
    onSuccess: () => {
      message.success(modal.record ? 'Đã cập nhật bản tin' : 'Đã đăng bảng tin chi nhánh mới');
      void qc.invalidateQueries({ queryKey: [engagementApi.announcementKey] });
      setModal({ open: false, record: null });
    },
    onError: (err) => showError(err, 'Không thể lưu bản tin'),
  });

  const deleteMut = useMutation({
    mutationFn: (id: number) => engagementApi.deleteAnnouncement(id),
    onSuccess: () => {
      message.success('Đã xóa bản tin');
      void qc.invalidateQueries({ queryKey: [engagementApi.announcementKey] });
    },
    onError: (err) => showError(err, 'Không thể xóa bản tin'),
  });

  const columns = useMemo<ColumnsType<Announcement>>(
    () => [
      {
        title: 'Tiêu đề bản tin',
        dataIndex: 'title',
        render: (title: string, r) => (
          <Space direction="vertical" size={2}>
            <Space>
              {r.pinned && (
                <Tag color="red" icon={<PushpinOutlined />}>
                  Ghim nổi bật
                </Tag>
              )}
              <Typography.Text strong>{title}</Typography.Text>
            </Space>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {r.body}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Phạm vi hiển thị',
        dataIndex: 'branch_name',
        width: 180,
        render: (v: string | undefined, r) =>
          r.branch_id ? v || `CN #${r.branch_id}` : <Tag color="blue">Toàn hệ thống</Tag>,
      },
      {
        title: 'Ngày đăng',
        dataIndex: 'publish_at',
        width: 150,
        render: (v: string) => formatDateTime(v),
      },
      {
        title: 'Hết hạn',
        dataIndex: 'expire_at',
        width: 130,
        render: (v: string | null | undefined) => (v ? formatDate(v) : 'Không thời hạn'),
      },
      {
        title: '',
        key: 'actions',
        width: 100,
        render: (_, r) =>
          canManage ? (
            <Space size={4}>
              <Button
                type="text"
                size="small"
                icon={<EditOutlined />}
                onClick={() => setModal({ open: true, record: r })}
              />
              <ConfirmButton
                perm={PERM.ENGAGEMENT_MANAGE}
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
                confirmTitle={`Xóa bản tin "${r.title}"?`}
                onConfirm={() => deleteMut.mutateAsync(r.id)}
                aria-label="Xóa"
              />
            </Space>
          ) : null,
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canManage],
  );

  return (
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col />
        <Col>
          <ActionBar
            onAdd={() => setModal({ open: true, record: null })}
            addPerm={PERM.ENGAGEMENT_MANAGE}
            addText="Đăng bản tin mới"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tiêu đề, nội dung thông báo..."
        colTextSearch={6}
        filters={
          branchId
            ? []
            : [
                {
                  name: 'branch_id',
                  placeholder: 'Chi nhánh',
                  options: branches.map((b) => ({ label: b.name, value: b.id })),
                  col: 5,
                },
              ]
        }
      />

      <DataTable<Announcement>
        table={table}
        columns={columns}
        emptyText="Chưa có thông báo nào trên bảng tin"
      />

      <Modal
        open={modal.open}
        title={modal.record ? 'Cập nhật bản tin' : 'Đăng thông báo lên Bảng tin Hội viên'}
        okText="Đăng bản tin"
        cancelText="Hủy"
        onOk={() => form.submit()}
        onCancel={() => setModal({ open: false, record: null })}
        confirmLoading={saveMut.isPending}
        width={560}
        forceRender
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(v) => saveMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Form.Item
            name="title"
            label="Tiêu đề thông báo"
            rules={[{ required: true, whitespace: true, message: 'Nhập tiêu đề' }]}
          >
            <Input placeholder="VD: Thông báo lịch hoạt động dịp Lễ Quốc Khánh 2/9" />
          </Form.Item>
          <Row gutter={12}>
            <Col xs={12}>
              <Form.Item name="branch_id" label="Chi nhánh (Để trống = Toàn hệ thống)">
                <BranchSelect activeOnly allowClear />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="expire_at" label="Ngày gỡ tự động (Tùy chọn)">
                <DatePicker style={{ width: '100%' }} format={DATE_FORMAT} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            name="body"
            label="Nội dung chi tiết"
            rules={[{ required: true, whitespace: true, message: 'Nhập nội dung thông báo' }]}
          >
            <Input.TextArea rows={4} placeholder="Nhập nội dung thông báo gửi tới hội viên..." />
          </Form.Item>
          <Form.Item name="pinned" valuePropName="checked">
            <Checkbox>Ghim bản tin này lên đầu trang Cổng hội viên (/me)</Checkbox>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

/* --------------------------------------------------------------------------
 * TAB 3: Khảo sát & Dashboard NPS (/admin/surveys, /admin/reports/nps)
 * -------------------------------------------------------------------------- */
function SurveysAndNpsTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm<SurveyInput>();

  const npsQuery = useQuery({
    queryKey: [engagementApi.npsKey, branchId],
    queryFn: () => engagementApi.npsReport({ branch_id: branchId ?? undefined }),
  });

  const surveysTable = useTableQuery<Survey, { page?: number; limit?: number }>({
    queryKey: [engagementApi.surveyKey, 'list'],
    queryFn: (p) => engagementApi.listSurveys(p),
    defaults: { order_by: 'id', order_dir: 'DESC' },
  });

  const createSurveyMut = useMutation({
    mutationFn: (v: SurveyInput) =>
      engagementApi.createSurvey({
        title: v.title.trim(),
        description: v.description?.trim() || '',
        type: v.type ?? 'NPS',
        status: 'ACTIVE',
      }),
    onSuccess: () => {
      message.success('Đã phát hành khảo sát NPS mới!');
      void qc.invalidateQueries({ queryKey: [engagementApi.surveyKey] });
      setModalOpen(false);
    },
    onError: (err) => showError(err, 'Không thể tạo khảo sát'),
  });

  const closeSurveyMut = useMutation({
    mutationFn: (id: number) => engagementApi.updateSurvey(id, { status: 'CLOSED' }),
    onSuccess: () => {
      message.success('Đã đóng đợt khảo sát');
      void qc.invalidateQueries({ queryKey: [engagementApi.surveyKey] });
    },
    onError: (err) => showError(err, 'Không thể đóng khảo sát'),
  });

  const nps = npsQuery.data;

  const surveyColumns = useMemo<ColumnsType<Survey>>(
    () => [
      {
        title: 'Chiến dịch khảo sát',
        dataIndex: 'title',
        render: (t: string, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>{t}</Typography.Text>
            {r.description && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {r.description}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Loại khảo sát',
        dataIndex: 'type',
        width: 140,
        render: (t: string) => <Tag color="geekblue">{t === 'NPS' ? 'Đo lường NPS (0–10)' : t}</Tag>,
      },
      {
        title: 'Ngày bắt đầu',
        dataIndex: 'start_at',
        width: 150,
        render: (v: string) => formatDateTime(v),
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        width: 130,
        render: (s: string) => <StatusTag status={s} />,
      },
      {
        title: '',
        key: 'actions',
        width: 120,
        render: (_, r) =>
          canManage && r.status === 'ACTIVE' ? (
            <ConfirmButton
              perm={PERM.ENGAGEMENT_MANAGE}
              size="small"
              confirmTitle={`Kết thúc đợt khảo sát "${r.title}"?`}
              onConfirm={() => closeSurveyMut.mutateAsync(r.id)}
            >
              Đóng khảo sát
            </ConfirmButton>
          ) : null,
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canManage],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row gutter={[12, 12]}>
        <Col xs={12} md={6}>
          <StatCard
            label="Chỉ số Net Promoter Score (NPS)"
            value={nps ? (nps.nps_score > 0 ? `+${nps.nps_score}` : nps.nps_score) : 0}
            caption={`Tổng số phản hồi: ${nps?.total_responses ?? 0} lượt`}
            icon={<StarOutlined />}
            loading={npsQuery.isLoading}
          />
        </Col>
        <Col xs={12} md={6}>
          <StatCard
            label="Promoters – Hài lòng cao (9–10đ)"
            value={`${nps?.promoters_pct ?? 0}%`}
            caption={`${nps?.promoters_count ?? 0} hội viên`}
            icon={<SmileOutlined />}
            loading={npsQuery.isLoading}
          />
        </Col>
        <Col xs={12} md={6}>
          <StatCard
            label="Passives – Trung lập (7–8đ)"
            value={`${nps?.passives_pct ?? 0}%`}
            caption={`${nps?.passives_count ?? 0} hội viên`}
            icon={<MehOutlined />}
            loading={npsQuery.isLoading}
          />
        </Col>
        <Col xs={12} md={6}>
          <StatCard
            label="Detractors – Chưa hài lòng (0–6đ)"
            value={`${nps?.detractors_pct ?? 0}%`}
            caption={`${nps?.detractors_count ?? 0} hội viên cần CSKH`}
            icon={<FrownOutlined />}
            loading={npsQuery.isLoading}
          />
        </Col>
      </Row>

      <Card size="small" title="Phân bổ tỷ lệ Promoters / Passives / Detractors">
        <Row gutter={24} align="middle">
          <Col xs={24} md={8}>
            <Typography.Text strong style={{ color: '#16a34a' }}>
              Promoters (9–10 điểm): {nps?.promoters_pct ?? 0}%
            </Typography.Text>
            <Progress percent={nps?.promoters_pct ?? 0} strokeColor="#16a34a" />
          </Col>
          <Col xs={24} md={8}>
            <Typography.Text strong style={{ color: '#eab308' }}>
              Passives (7–8 điểm): {nps?.passives_pct ?? 0}%
            </Typography.Text>
            <Progress percent={nps?.passives_pct ?? 0} strokeColor="#eab308" />
          </Col>
          <Col xs={24} md={8}>
            <Typography.Text strong style={{ color: '#dc2626' }}>
              Detractors (0–6 điểm): {nps?.detractors_pct ?? 0}%
            </Typography.Text>
            <Progress percent={nps?.detractors_pct ?? 0} strokeColor="#dc2626" />
          </Col>
        </Row>
      </Card>

      <Card
        size="small"
        title="Danh sách chiến dịch khảo sát hội viên (/admin/surveys)"
        extra={
          <Space>
            {canManage && (
              <Button
                color="orange"
                variant="solid"
                icon={<PlusOutlined />}
                onClick={() => {
                  form.resetFields();
                  form.setFieldsValue({ type: 'NPS' });
                  setModalOpen(true);
                }}
              >
                Tạo khảo sát NPS mới
              </Button>
            )}
            <Button
              icon={<ReloadOutlined spin={surveysTable.isFetching || npsQuery.isFetching} />}
              onClick={() => {
                void surveysTable.refetch();
                void npsQuery.refetch();
              }}
            >
              Làm mới
            </Button>
          </Space>
        }
      >
        <DataTable<Survey>
          table={surveysTable}
          columns={surveyColumns}
          emptyText="Chưa có đợt khảo sát nào"
        />
      </Card>

      <Modal
        open={modalOpen}
        title="Tạo đợt khảo sát mức độ hài lòng hội viên (NPS)"
        okText="Phát hành khảo sát"
        cancelText="Hủy"
        onOk={() => form.submit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={createSurveyMut.isPending}
        width={500}
        forceRender
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(v) => createSurveyMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Form.Item
            name="title"
            label="Tiêu đề đợt khảo sát"
            rules={[{ required: true, whitespace: true, message: 'Nhập tiêu đề khảo sát' }]}
          >
            <Input placeholder="VD: Khảo sát chất lượng dịch vụ GymMaster Quý 4/2026" />
          </Form.Item>
          <Form.Item name="type" label="Loại khảo sát">
            <Select
              options={[
                { value: 'NPS', label: 'Khảo sát chuẩn NPS (Thang điểm 0 – 10)' },
                { value: 'CUSTOM', label: 'Khảo sát lấy ý kiến tùy chỉnh' },
              ]}
            />
          </Form.Item>
          <Form.Item name="description" label="Câu hỏi / Lời dẫn hiển thị cho hội viên">
            <Input.TextArea
              rows={3}
              placeholder="Bạn có sẵn lòng giới thiệu phòng tập GymMaster cho bạn bè, đồng nghiệp không?"
            />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}

export default function EngagementPage() {
  const { can } = usePermission();
  const canManage = can(PERM.ENGAGEMENT_MANAGE);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <PageTitle
        title="Dinh dưỡng, Bảng tin & Khảo sát NPS"
        subtitle="Giao thực đơn dinh dưỡng cá nhân hóa, quản lý bảng tin chi nhánh và đo lường chỉ số hài lòng hội viên NPS"
      />

      <Tabs
        defaultActiveKey="nutrition"
        items={[
          {
            key: 'nutrition',
            label: (
              <Space size={6}>
                <AppleOutlined />
                <span>Thực đơn dinh dưỡng & Món ăn</span>
              </Space>
            ),
            children: <NutritionAndFoodsTab canManage={canManage} />,
          },
          {
            key: 'announcements',
            label: (
              <Space size={6}>
                <NotificationOutlined />
                <span>Bảng tin chi nhánh</span>
              </Space>
            ),
            children: <AnnouncementsTab canManage={canManage} />,
          },
          {
            key: 'nps',
            label: (
              <Space size={6}>
                <HeartOutlined />
                <span>Khảo sát & Dashboard NPS</span>
              </Space>
            ),
            children: <SurveysAndNpsTab canManage={canManage} />,
          },
        ]}
      />
    </Space>
  );
}
