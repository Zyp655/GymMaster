import type { Messages } from '@/i18n/messages';

declare module 'next-intl' {
  interface AppConfig {
    Locale: 'vi';
    Messages: Messages;
  }
}
