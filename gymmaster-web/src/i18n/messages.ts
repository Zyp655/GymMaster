import auth from '../../messages/vi/auth.json';
import common from '../../messages/vi/common.json';
import landing from '../../messages/vi/landing.json';
import portal from '../../messages/vi/portal.json';

/**
 * Chuỗi giao diện tách theo domain: messages/vi/<domain>.json → namespace <domain>.
 */
export const messages = {
  common,
  auth,
  portal,
  landing,
};

export type Messages = typeof messages;
