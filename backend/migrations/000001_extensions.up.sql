-- citext: case-insensitive emails.
-- pg_trgm: trigram similarity for typo-tolerant search.
-- pgcrypto: gen_random_uuid() and bcrypt for the seeded demo user.
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
