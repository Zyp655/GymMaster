DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('locker.view', 'locker.manage'));
DELETE FROM permissions WHERE code IN ('locker.view', 'locker.manage');

ALTER TABLE invoice_items DROP CONSTRAINT IF EXISTS invoice_items_item_type_check;
ALTER TABLE invoice_items
    ADD CONSTRAINT invoice_items_item_type_check
    CHECK (item_type IN ('MEMBERSHIP', 'PT', 'PRODUCT', 'FEE', 'OTHER'));

DROP TABLE IF EXISTS locker_rentals;
DROP TABLE IF EXISTS lockers;
