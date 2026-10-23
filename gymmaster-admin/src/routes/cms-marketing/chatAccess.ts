/**
 * Quy tắc gửi tin nhắn hội thoại PT – khớp `CMSService.SendMessage` (gymmaster-api):
 * chỉ người tham gia (HLV của hợp đồng hoặc hội viên) được gửi, và hợp đồng phải ACTIVE.
 * OWNER/MANAGER chỉ xem giám sát (chỉ đọc).
 */
export interface ChatSendState {
  canSend: boolean;
  /** Lý do không gửi được (hiển thị cho người dùng); `null` khi gửi được. */
  reason: string | null;
}

export function chatSendState(
  conv: { trainer_id: number; contract_status: string } | null | undefined,
  userId: number | null | undefined,
): ChatSendState {
  if (!conv) return { canSend: false, reason: 'Chọn một hội thoại để xem tin nhắn.' };
  if (!userId || conv.trainer_id !== userId) {
    return {
      canSend: false,
      reason:
        'Bạn đang xem ở chế độ giám sát (chỉ đọc). Chỉ HLV phụ trách và hội viên của hợp đồng PT mới được gửi tin nhắn.',
    };
  }
  if (conv.contract_status !== 'ACTIVE') {
    return {
      canSend: false,
      reason: 'Hợp đồng PT không còn hiệu lực nên không thể gửi tin nhắn.',
    };
  }
  return { canSend: true, reason: null };
}
