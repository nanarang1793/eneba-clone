ALTER TABLE product_keys DROP CONSTRAINT IF EXISTS product_keys_order_item_fk;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TYPE  IF EXISTS order_status;
