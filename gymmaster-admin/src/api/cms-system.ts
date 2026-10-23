import { http as api } from '@/utils/api';

export interface PublicTrainerItem {
  id: number;
  user_id: number;
  full_name: string;
  slug: string;
  headline: string;
  bio: string;
  specialties: string;
  certifications: string;
  experience_years: number;
  avatar_url: string;
  featured: boolean;
  is_published: boolean;
  rating_avg: number;
  rating_count: number;
}

export interface TestimonialItem {
  id: number;
  member_id?: number;
  author_name: string;
  author_role: string;
  avatar_url: string;
  rating: number;
  content: string;
  before_after_note: string;
  status: 'PENDING' | 'APPROVED' | 'REJECTED';
  approved_at?: string;
  created_at: string;
}

export interface PostItem {
  id: number;
  slug: string;
  title: string;
  excerpt: string;
  cover_url: string;
  category: string;
  content: string;
  status: 'DRAFT' | 'PUBLISHED' | 'ARCHIVED';
  published_at?: string;
  created_at: string;
}

export interface ContactMessageItem {
  id: number;
  branch_id?: number;
  full_name: string;
  phone: string;
  email: string;
  subject: string;
  message: string;
  status: 'NEW' | 'CONTACTED' | 'CLOSED';
  handled_note: string;
  created_at: string;
}

export interface ConversationItem {
  id: number;
  pt_contract_id: number;
  member_id: number;
  member_name: string;
  member_code: string;
  trainer_id: number;
  trainer_name: string;
  contract_status: string;
  remaining_sessions: number;
  last_message_at?: string;
  last_message_preview: string;
  unread_count: number;
}

export interface MessageItem {
  id: number;
  conversation_id: number;
  sender_user_id: number;
  sender_name: string;
  sender_role: string;
  content: string;
  attachment_url?: string;
  read_at?: string;
  created_at: string;
}

export interface CalendarTokenRes {
  token: string;
  ics_url: string;
}

export interface SearchHit {
  id: number;
  category: 'MEMBER' | 'INVOICE' | 'PRODUCT' | 'PLAN';
  title: string;
  subtitle: string;
  link: string;
}

export interface BranchTransferItem {
  id: number;
  member_id: number;
  member_name: string;
  member_code: string;
  from_branch_id: number;
  from_branch_name: string;
  to_branch_id: number;
  to_branch_name: string;
  subscription_id?: number;
  transfer_fee: number;
  reason: string;
  status: string;
  approved_at?: string;
  created_at: string;
}

export interface LegacyImportRow {
  full_name: string;
  phone: string;
  email?: string;
  gender?: string;
  plan_code?: string;
  sub_start_date?: string;
  sub_end_date?: string;
  remaining_sessions?: number;
  pt_package_code?: string;
  pt_remaining_sessions?: number;
  /** Email HLV phụ trách hợp đồng PT chuyển sang – bắt buộc khi có `pt_package_code`. */
  pt_trainer_email?: string;
  debt_amount?: number;
}

export interface LegacyImportJobItem {
  id: number;
  branch_id: number;
  branch_name: string;
  file_name: string;
  mode: 'DRY_RUN' | 'COMMIT';
  status: string;
  total_rows: number;
  ok_rows: number;
  error_rows: number;
  errors: { row: number; phone: string; message: string }[];
  created_at: string;
}

export interface SetupWizardStatus {
  completed: boolean;
  skipped: boolean;
  steps: {
    key: string;
    title: string;
    completed: boolean;
    count: number;
    route: string;
  }[];
}

export const cmsSystemApi = {
  // CMS & Marketing
  listTrainerProfiles: () => api.get<PublicTrainerItem[]>('/admin/trainer-profiles'),
  saveTrainerProfile: (body: Partial<PublicTrainerItem> & { user_id: number; slug: string }) =>
    api.post<PublicTrainerItem>('/admin/trainer-profiles', body),
  deleteTrainerProfile: (id: number) => api.delete<void>(`/admin/trainer-profiles/${id}`),

  listTestimonials: (status?: string) =>
    api.get<TestimonialItem[]>('/admin/testimonials', { params: { status } }),
  approveTestimonial: (id: number) =>
    api.post<TestimonialItem>(`/admin/testimonials/${id}/approve`),
  rejectTestimonial: (id: number) =>
    api.post<TestimonialItem>(`/admin/testimonials/${id}/reject`),
  deleteTestimonial: (id: number) => api.delete<void>(`/admin/testimonials/${id}`),

  listPosts: (params?: { status?: string; category?: string }) =>
    api.get<PostItem[]>('/admin/posts', { params }),
  createPost: (body: Partial<PostItem> & { slug: string; title: string; content: string }) =>
    api.post<PostItem>('/admin/posts', body),
  updatePost: (id: number, body: Partial<PostItem>) =>
    api.put<PostItem>(`/admin/posts/${id}`, body),
  deletePost: (id: number) => api.delete<void>(`/admin/posts/${id}`),

  listContactMessages: (status?: string) =>
    api.get<ContactMessageItem[]>('/admin/contact-messages', { params: { status } }),
  updateContactMessage: (id: number, body: { status: string; handled_note?: string }) =>
    api.put<ContactMessageItem>(`/admin/contact-messages/${id}`, body),

  // PT Messaging & Calendar ICS
  listConversations: () => api.get<ConversationItem[]>('/admin/conversations'),
  listMessages: (conversationId: number) =>
    api.get<MessageItem[]>(`/admin/conversations/${conversationId}/messages`),
  sendMessage: (conversationId: number, content: string) =>
    api.post<MessageItem>(`/admin/conversations/${conversationId}/messages`, { content }),
  getCalendarToken: () => api.get<CalendarTokenRes>('/admin/calendar-token'),
  rotateCalendarToken: () => api.post<CalendarTokenRes>('/admin/calendar-token/rotate'),

  // System Tools & Sprint 30
  globalSearch: (q: string) =>
    api.get<{ items: SearchHit[] }>('/admin/search', { params: { q } }),
  bulkMemberAction: (body: {
    member_ids: number[];
    action: string;
    title?: string;
    message?: string;
    to_branch_id?: number;
  }) => api.post<{ affected: number; message: string }>('/admin/members/bulk-action', body),

  listBranchTransfers: () => api.get<BranchTransferItem[]>('/admin/branch-transfers'),
  createBranchTransfer: (body: {
    member_id: number;
    to_branch_id: number;
    transfer_fee?: number;
    reason?: string;
  }) => api.post<BranchTransferItem>('/admin/branch-transfers', body),

  refundOnlineTxn: (id: number, body: { amount?: number; reason: string }) =>
    api.post<{ refund_id: number; gateway_ref: string }>(
      `/admin/online-payments/${id}/refund`,
      body,
    ),

  listLegacyImports: () => api.get<LegacyImportJobItem[]>('/admin/legacy-imports'),
  runLegacyImport: (body: {
    branch_id: number;
    file_name: string;
    mode: 'DRY_RUN' | 'COMMIT';
    rows: LegacyImportRow[];
  }) => api.post<LegacyImportJobItem>('/admin/legacy-imports', body),

  getSetupWizardStatus: () => api.get<SetupWizardStatus>('/admin/setup-wizard/status'),
  completeSetupWizard: (body: {
    center_name?: string;
    hotline?: string;
    opening_hours?: string;
    cancel_class_before_hours?: number;
    skip?: boolean;
  }) => api.post<SetupWizardStatus>('/admin/setup-wizard/complete', body),
};
