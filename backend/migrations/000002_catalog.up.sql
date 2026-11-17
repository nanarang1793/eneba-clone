CREATE TABLE platforms (
    id    smallserial PRIMARY KEY,
    slug  text NOT NULL UNIQUE,
    name  text NOT NULL
);

CREATE TABLE regions (
    id    smallserial PRIMARY KEY,
    slug  text NOT NULL UNIQUE,
    name  text NOT NULL
);

CREATE TYPE product_type AS ENUM ('game', 'dlc', 'gift_card', 'subscription');

-- A product is one sellable variant: title + platform + region,
-- e.g. "Elden Ring (PC) Steam Key GLOBAL".
CREATE TABLE products (
    id            bigserial PRIMARY KEY,
    slug          text NOT NULL UNIQUE,
    name          text NOT NULL,
    title         text NOT NULL,            -- base title without platform/region, for grouping & display
    description   text NOT NULL DEFAULT '',
    image_url     text,
    type          product_type NOT NULL DEFAULT 'game',
    platform_id   smallint NOT NULL REFERENCES platforms(id),
    region_id     smallint NOT NULL REFERENCES regions(id),
    msrp_cents    integer CHECK (msrp_cents > 0),          -- "retail" price, used for discount %
    release_date  date,
    publisher     text,
    sales_count   integer NOT NULL DEFAULT 0,              -- incremented on purchase; seeded for popularity
    search_vector tsvector GENERATED ALWAYS AS (
        to_tsvector('simple', coalesce(name, '') || ' ' || coalesce(publisher, ''))
    ) STORED,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX products_platform_idx    ON products (platform_id);
CREATE INDEX products_region_idx      ON products (region_id);
CREATE INDEX products_release_idx     ON products (release_date DESC);
CREATE INDEX products_sales_idx       ON products (sales_count DESC);
CREATE INDEX products_search_idx      ON products USING gin (search_vector);
CREATE INDEX products_name_trgm_idx   ON products USING gin (lower(name) gin_trgm_ops);

CREATE TABLE sellers (
    id           serial PRIMARY KEY,
    slug         text NOT NULL UNIQUE,
    name         text NOT NULL,
    rating       numeric(3,2) NOT NULL CHECK (rating BETWEEN 0 AND 5),
    sales_count  integer NOT NULL DEFAULT 0,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE offers (
    id           bigserial PRIMARY KEY,
    product_id   bigint NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    seller_id    integer NOT NULL REFERENCES sellers(id),
    price_cents  integer NOT NULL CHECK (price_cents > 0),
    is_active    boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (product_id, seller_id)
);

CREATE INDEX offers_product_price_idx ON offers (product_id, price_cents) WHERE is_active;

CREATE TYPE key_status AS ENUM ('available', 'reserved', 'sold');

-- Stock is the number of usable keys, never a stored counter.
-- A 'reserved' key whose reserved_until has passed counts as available.
CREATE TABLE product_keys (
    id              bigserial PRIMARY KEY,
    offer_id        bigint NOT NULL REFERENCES offers(id) ON DELETE CASCADE,
    code            text NOT NULL UNIQUE,
    status          key_status NOT NULL DEFAULT 'available',
    reserved_until  timestamptz,
    order_item_id   bigint,                  -- FK added in the orders migration
    created_at      timestamptz NOT NULL DEFAULT now(),
    CHECK (status <> 'reserved' OR reserved_until IS NOT NULL)
);

CREATE INDEX product_keys_offer_status_idx ON product_keys (offer_id, status);
