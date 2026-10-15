DROP TABLE IF EXISTS legacy_import_jobs;
DROP TABLE IF EXISTS member_branch_transfers;
ALTER TABLE pt_contracts DROP COLUMN IF EXISTS source;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS source;
ALTER TABLE refunds DROP COLUMN IF EXISTS gateway_ref;
ALTER TABLE refunds DROP COLUMN IF EXISTS status;
ALTER TABLE refunds DROP COLUMN IF EXISTS online_txn_id;
