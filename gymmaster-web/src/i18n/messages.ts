import audit from '../../messages/vi/audit.json';
import auth from '../../messages/vi/auth.json';
import branch from '../../messages/vi/branch.json';
import checkin from '../../messages/vi/checkin.json';
import classes from '../../messages/vi/classes.json';
import common from '../../messages/vi/common.json';
import dashboard from '../../messages/vi/dashboard.json';
import invoice from '../../messages/vi/invoice.json';
import layout from '../../messages/vi/layout.json';
import member from '../../messages/vi/member.json';
import notification from '../../messages/vi/notification.json';
import plan from '../../messages/vi/plan.json';
import portal from '../../messages/vi/portal.json';
import pos from '../../messages/vi/pos.json';
import promotion from '../../messages/vi/promotion.json';
import pt from '../../messages/vi/pt.json';
import report from '../../messages/vi/report.json';
import role from '../../messages/vi/role.json';
import room from '../../messages/vi/room.json';
import setting from '../../messages/vi/setting.json';
import shift from '../../messages/vi/shift.json';
import staff from '../../messages/vi/staff.json';
import subscription from '../../messages/vi/subscription.json';

/**
 * Chuỗi giao diện tách theo domain: messages/vi/<domain>.json → namespace <domain>.
 * Mỗi module chỉ sửa file JSON của mình; file này đã đăng ký sẵn mọi domain.
 */
export const messages = {
  common,
  auth,
  layout,
  dashboard,
  branch,
  role,
  staff,
  room,
  shift,
  member,
  plan,
  promotion,
  setting,
  subscription,
  invoice,
  checkin,
  classes,
  pt,
  pos,
  report,
  notification,
  portal,
  audit,
};

export type Messages = typeof messages;
