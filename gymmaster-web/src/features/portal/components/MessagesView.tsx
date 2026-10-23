'use client';

import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import {
  BellAlertIcon,
  CalendarDaysIcon,
  ChatBubbleLeftRightIcon,
  ClipboardDocumentIcon,
  PaperAirplaneIcon,
} from '@heroicons/react/24/outline';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { env } from '@/env';
import { formatDateTime } from '@/lib/format';
import { hasVisibleText } from '@/lib/text';
import {
  portalKeys,
  useMemberCalendarTokenQuery,
  useMemberConversationsQuery,
  useMemberMessagesQuery,
  useMemberNotificationPreferencesQuery,
  useRotateMemberCalendarTokenMutation,
  useSaveMemberNotificationPreferencesMutation,
  useSendMemberMessageMutation,
} from '../queries';
import type { NotificationPrefItem } from '../types';

export function MessagesView() {
  const t = useTranslations('portal.messages');
  const tPref = useTranslations('portal.messages.prefTypes');
  const tContract = useTranslations('portal.status.ptContract');
  /** Nhãn tiếng Việt cho trạng thái hợp đồng PT; mã lạ (chưa có bản dịch) giữ nguyên. */
  const contractLabel = (status: string) =>
    tContract.has(status as never) ? tContract(status as never) : status;
  const prefLabel = (type: string) => (tPref.has(type as never) ? tPref(type as never) : type);
  const [selectedConvId, setSelectedConvId] = useState<number | null>(null);
  const [chatText, setChatText] = useState('');

  const convQuery = useMemberConversationsQuery();
  const conversations = convQuery.data ?? [];
  const activeConvId = selectedConvId ?? conversations[0]?.id ?? null;

  const messagesQuery = useMemberMessagesQuery(activeConvId);
  const sendMut = useSendMemberMessageMutation();
  const messages = messagesQuery.data ?? [];
  const qc = useQueryClient();
  const scrollRef = useRef<HTMLDivElement | null>(null);

  // Tự cuộn xuống tin mới nhất khi mở hội thoại / có tin mới.
  const lastMessageId = messages.length > 0 ? messages[messages.length - 1].id : 0;
  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [activeConvId, lastMessageId]);

  // GET tin nhắn đánh dấu "đã đọc" phía máy chủ → làm mới danh sách hội thoại để badge "x mới" cập nhật.
  const activeUnread = conversations.find(c => c.id === activeConvId)?.unread_count ?? 0;
  const messagesUpdatedAt = messagesQuery.dataUpdatedAt;
  useEffect(() => {
    if (messagesUpdatedAt > 0 && activeUnread > 0) {
      void qc.invalidateQueries({ queryKey: portalKeys.conversations() });
    }
  }, [qc, messagesUpdatedAt, activeUnread]);

  const calQuery = useMemberCalendarTokenQuery();
  const rotateCalMut = useRotateMemberCalendarTokenMutation();

  const prefsQuery = useMemberNotificationPreferencesQuery();
  const savePrefsMut = useSaveMemberNotificationPreferencesMutation();

  // Chỉ gồm ký tự vô hình (zero-width…) coi như rỗng – không gửi bong bóng chat trống.
  const canSend = hasVisibleText(chatText);

  const handleSend = () => {
    if (!activeConvId || !canSend) return;
    sendMut.mutate(
      { conversationId: activeConvId, content: chatText.trim() },
      {
        onSuccess: () => setChatText(''),
        onError: err => {
          toast.error(err instanceof Error ? err.message : t('sendFailed'));
        },
      },
    );
  };

  const togglePref = (target: NotificationPrefItem) => {
    if (target.mandatory) return;
    const current = prefsQuery.data ?? [];
    const next = current.map(item =>
      item.channel === target.channel && item.type === target.type
        ? { ...item, enabled: !item.enabled }
        : item,
    );
    savePrefsMut.mutate(next, {
      onSuccess: () => toast.success(t('prefsSaved')),
    });
  };

  // ics_url do API trả về là đường dẫn trên máy chủ API (VD: /v1/public/calendar.ics?token=...),
  // nên phải ghép với origin của NEXT_PUBLIC_API_URL chứ không phải origin của web.
  const rawIcsUrl = calQuery.data?.ics_url;
  const icsUrl = rawIcsUrl
    ? /^https?:\/\//i.test(rawIcsUrl)
      ? rawIcsUrl
      : new URL(rawIcsUrl, env.NEXT_PUBLIC_API_URL).toString()
    : undefined;

  const copyIcsUrl = () => {
    if (!icsUrl) return;
    void navigator.clipboard.writeText(icsUrl);
    toast.success(t('icsCopied'));
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-h2 font-bold text-primary">{t('title')}</h1>
        <p className="mt-1 text-body text-secondary">{t('subtitle')}</p>
      </div>

      {/* Khu vực Chat HLV ↔ Hội viên */}
      <Card className="p-5">
        <div className="mb-4 flex items-center gap-2">
          <ChatBubbleLeftRightIcon className="size-5 text-brand-600" />
          <h2 className="text-h4 font-semibold text-primary">{t('chatTitle')}</h2>
        </div>

        {conversations.length === 0 ? (
          <div className="rounded-xl border border-dashed border-line p-6 text-center text-sm text-secondary">
            {t('noConversation')}
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-3">
            <div className="space-y-2 md:border-r md:border-line md:pr-4">
              {conversations.map(conv => {
                const active = conv.id === activeConvId;
                return (
                  <button
                    key={conv.id}
                    type="button"
                    onClick={() => setSelectedConvId(conv.id)}
                    className={`w-full rounded-xl border p-3 text-left transition-colors ${
                      active
                        ? 'border-brand-500 bg-brand-50/70'
                        : 'border-line bg-surface hover:bg-subtle'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-semibold text-primary">
                        {t('trainerName', { name: conv.trainer_name })}
                      </span>
                      {conv.unread_count > 0 ? (
                        <Badge variant="brand">{t('unread', { count: conv.unread_count })}</Badge>
                      ) : null}
                    </div>
                    <div className="mt-1 text-xs text-secondary">
                      {t('remainingSessions', {
                        count: conv.remaining_sessions,
                        status: contractLabel(conv.contract_status),
                      })}
                    </div>
                    {conv.last_message_preview ? (
                      <div className="mt-1 truncate text-xs text-tertiary">
                        {conv.last_message_preview}
                      </div>
                    ) : null}
                  </button>
                );
              })}
            </div>

            <div className="flex flex-col justify-between md:col-span-2">
              <div
                ref={scrollRef}
                className="max-h-80 min-h-56 space-y-3 overflow-y-auto rounded-xl bg-subtle/50 p-4"
              >
                {messages.length === 0 ? (
                  <p className="text-center text-sm text-secondary">{t('empty')}</p>
                ) : (
                  messages.map(m => {
                    const isMine = m.sender_role === 'MEMBER';
                    return (
                      <div
                        key={m.id}
                        className={`flex flex-col ${isMine ? 'items-end' : 'items-start'}`}
                      >
                        <div className="mb-1 text-xs text-tertiary">
                          {m.sender_name} · {formatDateTime(m.created_at)}
                        </div>
                        <div
                          className={`max-w-[80%] rounded-2xl px-3.5 py-2 text-sm ${
                            isMine
                              ? 'bg-brand-600 text-white'
                              : 'border border-line bg-surface text-primary'
                          }`}
                        >
                          {hasVisibleText(m.content) ? (
                            m.content
                          ) : (
                            <span className="italic opacity-80">{t('emptyContent')}</span>
                          )}
                        </div>
                      </div>
                    );
                  })
                )}
              </div>

              <div className="mt-3 flex gap-2">
                <Input
                  value={chatText}
                  onChange={e => setChatText(e.target.value)}
                  onKeyDown={e => {
                    if (e.key === 'Enter') handleSend();
                  }}
                  placeholder={t('placeholder')}
                />
                <Button onClick={handleSend} disabled={sendMut.isPending || !canSend}>
                  <PaperAirplaneIcon className="size-4" />
                  {t('send')}
                </Button>
              </div>
            </div>
          </div>
        )}
      </Card>

      {/* Đồng bộ lịch tập .ics */}
      <Card className="p-5">
        <div className="mb-2 flex items-center gap-2">
          <CalendarDaysIcon className="size-5 text-brand-600" />
          <h2 className="text-h4 font-semibold text-primary">{t('icsTitle')}</h2>
        </div>
        <p className="mb-4 text-sm text-secondary">{t('icsDescription')}</p>
        <div className="flex flex-wrap items-center gap-2">
          <Input
            readOnly
            value={icsUrl ?? t('icsLoading')}
            className="max-w-xl font-mono text-xs"
          />
          <Button variant="outline" onClick={copyIcsUrl}>
            <ClipboardDocumentIcon className="size-4" />
            {t('icsCopy')}
          </Button>
          <Button
            variant="ghost"
            onClick={() =>
              rotateCalMut.mutate(undefined, {
                onSuccess: () => toast.success(t('icsRotated')),
              })
            }
            disabled={rotateCalMut.isPending}
          >
            {t('icsRotate')}
          </Button>
        </div>
      </Card>

      {/* Tùy chọn nhận thông báo */}
      <Card className="p-5">
        <div className="mb-2 flex items-center gap-2">
          <BellAlertIcon className="size-5 text-brand-600" />
          <h2 className="text-h4 font-semibold text-primary">{t('prefsTitle')}</h2>
        </div>
        <p className="mb-4 text-sm text-secondary">{t('prefsDescription')}</p>

        <div className="divide-y divide-line">
          {(prefsQuery.data ?? [])
            .filter(item => item.channel === 'IN_APP')
            .map(item => (
              <div
                key={`${item.channel}-${item.type}`}
                className="flex items-center justify-between gap-4 py-3"
              >
                <div>
                  <div className="text-sm font-medium text-primary">{prefLabel(item.type)}</div>
                  <div className="text-xs text-tertiary">{t('prefsChannel')}</div>
                </div>
                <Button
                  size="sm"
                  variant={item.enabled ? 'primary' : 'outline'}
                  disabled={item.mandatory || savePrefsMut.isPending}
                  onClick={() => togglePref(item)}
                >
                  {item.enabled ? t('prefOn') : t('prefOff')}
                </Button>
              </div>
            ))}
        </div>
      </Card>
    </div>
  );
}
