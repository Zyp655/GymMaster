import { getRequestConfig } from 'next-intl/server';
import { messages } from './messages';

export const LOCALE = 'vi';
export const TIME_ZONE = 'Asia/Ho_Chi_Minh';

// Ứng dụng chỉ dùng tiếng Việt: không có định tuyến theo locale (/vi/...).
export default getRequestConfig(async () => ({
  locale: LOCALE,
  timeZone: TIME_ZONE,
  messages,
}));
