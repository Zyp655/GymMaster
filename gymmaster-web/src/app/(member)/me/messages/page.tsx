import type { Metadata } from 'next';
import { MessagesView } from '@/features/portal/components/MessagesView';

export const metadata: Metadata = {
  title: 'Tin nhắn HLV & Đồng bộ Lịch | GymMaster',
};

export default function MessagesPage() {
  return <MessagesView />;
}
