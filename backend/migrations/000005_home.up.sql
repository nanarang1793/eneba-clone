-- The homepage layout lives in the DB. The frontend asks for the ordered
-- list of blocks, then fetches each block's data from its own provider.
CREATE TABLE home_blocks (
    id         serial PRIMARY KEY,
    type       text NOT NULL,             -- 'banner' | 'best_sellers' | 'deals' | 'new_releases' | ...
    title      text NOT NULL,
    position   integer NOT NULL,
    config     jsonb NOT NULL DEFAULT '{}',
    is_active  boolean NOT NULL DEFAULT true
);

CREATE INDEX home_blocks_position_idx ON home_blocks (position) WHERE is_active;

CREATE TABLE banners (
    id         serial PRIMARY KEY,
    title      text NOT NULL,
    subtitle   text,
    image_url  text,
    cta_label  text NOT NULL,
    cta_url    text NOT NULL,
    position   integer NOT NULL,
    starts_at  timestamptz,
    ends_at    timestamptz
);
