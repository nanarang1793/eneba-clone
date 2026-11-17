CREATE TYPE order_status AS ENUM ('pending', 'paid', 'failed', 'expired');

CREATE TABLE orders (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          uuid NOT NULL REFERENCES users(id),
    status           order_status NOT NULL DEFAULT 'pending',
    total_cents      integer NOT NULL CHECK (total_cents >= 0),
    currency         char(3) NOT NULL DEFAULT 'EUR',
    idempotency_key  text NOT NULL,
    payment_ref      text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    paid_at          timestamptz,
    UNIQUE (user_id, idempotency_key)
);

CREATE INDEX orders_user_created_idx ON orders (user_id, created_at DESC);

-- Price and seller are copied at purchase time so history stays correct
-- if the offer later changes or disappears.
CREATE TABLE order_items (
    id                bigserial PRIMARY KEY,
    order_id          uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    offer_id          bigint NOT NULL REFERENCES offers(id),
    product_id        bigint NOT NULL REFERENCES products(id),
    seller_id         integer NOT NULL REFERENCES sellers(id),
    unit_price_cents  integer NOT NULL CHECK (unit_price_cents > 0),
    quantity          integer NOT NULL CHECK (quantity > 0)
);

CREATE INDEX order_items_order_idx ON order_items (order_id);

ALTER TABLE product_keys
    ADD CONSTRAINT product_keys_order_item_fk
    FOREIGN KEY (order_item_id) REFERENCES order_items(id);
