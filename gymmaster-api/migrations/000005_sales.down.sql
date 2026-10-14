DELETE FROM permissions WHERE module IN ('subscription', 'invoice');
DROP TABLE IF EXISTS subscription_events;
DROP TABLE IF EXISTS subscription_freezes;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS refunds;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS invoice_items;
DROP TABLE IF EXISTS invoices;
DROP SEQUENCE IF EXISTS invoice_no_seq;
