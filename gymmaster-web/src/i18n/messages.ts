import auth from '../../messages/vi/auth.json';
import common from '../../messages/vi/common.json';
import portal from '../../messages/vi/portal.json';

/**
 * Chuỗi giao diện tách theo domain: messages/vi/<domain>.json → namespace <domain>.
 */
export const messages = {
  common,
  auth,
  portal,
};

export type Messages = typeof messages;
