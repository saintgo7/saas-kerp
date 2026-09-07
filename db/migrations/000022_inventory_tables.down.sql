-- Roll back exactly what 000022 created, and nothing else.
--
-- Scope, stated explicitly because getting this wrong has cost this repository
-- before (000013's rollback used to drop tables owned by 000003, so a single
-- `migrate down` erased RLS permanently):
--
--   THIS FILE DROPS  the ten inventory tables introduced by 000022. Their
--                    indexes, constraints, triggers, RLS policies and the
--                    GRANTs to kerp_app/kerp_admin all belong to those tables
--                    and disappear with them.
--   THIS FILE KEEPS  trigger_set_updated_at()  (000003)
--                    current_tenant_id(), is_admin_context()  (000010/17/21)
--                    kerp_app, kerp_admin  (000016)
--                    every table, policy and privilege from 000001-000021.
--
-- Dropped children first so no statement needs CASCADE: a CASCADE here could
-- silently take a dependent object created by a LATER migration with it.

DROP TABLE IF EXISTS sales_order_items;
DROP TABLE IF EXISTS sales_orders;

DROP TABLE IF EXISTS purchase_order_items;
DROP TABLE IF EXISTS purchase_orders;

DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS stocks;

DROP TABLE IF EXISTS order_sequences;

-- products references product_categories, so it goes first.
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS warehouses;
DROP TABLE IF EXISTS product_categories;
