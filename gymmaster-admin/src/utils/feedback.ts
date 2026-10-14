/**
 * Cầu nối tới message/modal/notification của antd `<App>` để code ngoài component
 * (interceptor axios, thunk...) vẫn dùng được theme + locale hiện tại.
 * Trong component hãy dùng `const { message, modal } = App.useApp()`.
 */
import { message as staticMessage, Modal, notification as staticNotification } from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';
import type { ModalStaticFunctions } from 'antd/es/modal/confirm';
import type { NotificationInstance } from 'antd/es/notification/interface';

type ModalApi = Omit<ModalStaticFunctions, 'warn'>;

export const feedback: {
  message: MessageInstance;
  modal: ModalApi;
  notification: NotificationInstance;
} = {
  message: staticMessage,
  modal: Modal,
  notification: staticNotification,
};

export function bindFeedback(api: Partial<typeof feedback>): void {
  Object.assign(feedback, api);
}
