# Eneba clone: Intern Assignment 2027

A digital-key marketplace in Eneba's style. React + TypeScript frontend, Go REST API, PostgreSQL.

**Live demo:** https://eneba-clone-nine.vercel.app/store
**API:** https://eneba-clone-d0ia.onrender.com/api/products

> The API runs on Render's free tier, which sleeps after ~15 minutes without traffic. The first request after a pause can take up to a minute while it wakes up; after that it's fast.

> Status: **day 6.** Data-driven homepage (banner carousel + product rows, each from its own provider), store with filters/facets, product page with seller offers, search with suggestions, accounts, and a full purchase with real key reservation. Demo account: `demo@eneba.test` / `demo1234`. Test card: `4242 4242 4242 4242` (or `4000 0000 0000 0002` for a decline).

## Repo layout

```
backend/
  cmd/api/        HTTP server: config, DB pool, router, middleware
  cmd/seedgen/    deterministic seed generator → seed/seed.sql
  internal/httpx/ JSON + shared error format + panic recoverer + JSON body decoding
  internal/auth/  store.go (users + sessions SQL) · handler.go (register/login/logout/me, cookie) · middleware.go (RequireUser)
  internal/home/   handler.go (layout + block endpoints) · providers.go (one Provider per block type)
  internal/orders/ store.go (quote, order + key reservation, payment, history) · payments.go (PaymentProvider + simulated)
                  handler.go (HTTP) · types.go (JSON)
  internal/catalog/ params.go (validation) · store.go (list/facet/suggest SQL) · product.go (product + offers SQL)
                  synonyms.go (search short names) · handler.go (HTTP) · types.go (JSON)
  migrations/     golang-migrate SQL files (NNNNNN_name.up.sql / .down.sql)
  seed/seed.sql   generated seed data (committed, loaded with psql)
frontend/src/
  api/            fetch client (ApiError) + typed endpoint functions
  app/            routes
  components/     layout (Header, Layout, ScrollToTop, ErrorBoundary), ui (Price, Cover), usePageTitle
  features/store/ StorePage, FilterSidebar, SortSelect, ProductCard, Pagination, useStoreParams (URL state)
  features/product/ ProductPage, OffersTable
  features/search/  SearchBox (suggestions dropdown), useDebouncedValue
  features/auth/    LoginPage, RegisterPage, RequireAuth (protected routes), useCurrentUser, useLogout
  features/home/    HomePage (loads the layout, then each block on its own), BannerCarousel, ProductRow
  features/account/ AccountPage (profile + order history)
  features/cart/    cartStore (zustand, saved in localStorage), CartPage
  features/checkout/ CheckoutPage (countdown, test cards), useCountdown
  features/orders/  OrderPage (keys + copy button), StatusBadge
  styles/         Tailwind + Eneba design tokens
docs/AI_LOG.md    full AI conversation log (assignment deliverable)
docker-compose.yml
Makefile
```

## Running locally

Requirements: Docker Desktop, Go 1.24.7+, Node 20+.

**Everything in Docker (easiest on Windows; no Go or Node install needed)**, from the repo root:

```powershell
docker compose up -d db
docker compose run --rm migrate up
docker compose run --rm seed
docker compose up -d --build api web    # API on :8080, frontend on http://localhost:5173
docker compose logs -f api web          # watch the logs (Ctrl+C stops watching, not the apps)
```

Frontend edits hot-reload as usual. After changing Go code, run `docker compose up -d --build api` again. This route also avoids Windows Smart App Control, which blocks Go's unsigned build tools (`asm.exe`) on some Windows 11 PCs.

**Windows (PowerShell) with Go and Node installed locally**, from the repo root, no `make` needed:

```powershell
docker compose up -d db                 # Postgres on localhost:5432
docker compose run --rm migrate up      # create tables (and apply new migrations later)
docker compose run --rm seed            # load the 200 demo products
cd backend;  go run ./cmd/api           # API on http://localhost:8080 (leave running)
# in a second terminal:
cd frontend; npm install; npm run dev   # http://localhost:5173
```

**macOS / Linux / Git Bash** (with `make`):

```bash
make db-reset      # fresh Postgres + migrations + seed (wipes local data)
make api           # Go API on :8080
cd frontend && npm install && npm run dev   # http://localhost:5173
```

Other targets: `make migrate-up`, `make migrate-down`, `make seed`, `make seed-gen`, `make psql`.

The API reads `DATABASE_URL` and `PORT` from the environment and defaults to the local Docker database, so `go run ./cmd/api` works with no setup.

## API

All endpoints are under `/api` and return JSON. Errors always look like:

```json
{ "error": { "code": "invalid_parameter", "message": "sort must be one of relevance, price_asc, price_desc, newest, popular" } }
```

Codes: `invalid_parameter` (400), `not_found` (404), `method_not_allowed` (405), `internal` (500, details only in the server log).

| Endpoint | Description |
|---|---|
| `GET /api/healthz` | `{"status":"ok"}` when the database is reachable |
| `GET /api/products` | Store listing (below) |
| `GET /api/filters` | All platforms, regions, the cheapest/most expensive listed price, and the sort options |
| `GET /api/products/{slug}` | One product: details, cheapest in-stock price (`null` when sold out), `in_stock`, and `variants` (same game on other platforms/regions). 404 if the slug doesn't exist |
| `GET /api/products/{slug}/offers` | `{ items }`: every active seller offer with seller rating/sales, `stock` (usable keys), `in_stock` and `is_best`. Sorted in-stock first, then price, then seller rating |
| `GET /api/search/suggest?q=&limit=6` | `{ items }`: up to `limit` (1–10) best matches for the header dropdown, same search as the store. Fewer than 2 characters returns an empty list |
| `POST /api/auth/register` | `{email, password, display_name?}` → 201 + user, and signs you in. 400 for an invalid email, a password outside 8–72 characters or a name over 40; 409 `email_taken` |
| `POST /api/auth/login` | `{email, password}` → 200 + user and the session cookie. 401 `invalid_credentials` for both an unknown email and a wrong password |
| `POST /api/auth/logout` | Deletes the session and the cookie → 204 |
| `GET /api/auth/me` | `{"user": {...}}` when signed in, `{"user": null}` when not (always 200: being signed out isn't an error) |
| `GET /api/home/layout` | `{blocks: [{id, type, title}]}`: the homepage's active blocks in order (from `home_blocks`) |
| `GET /api/home/blocks/{id}` | `{id, type, title, data}`: one block's content from its provider. Product blocks return `{items, see_all_url}`, the banner block `{items: [banners]}` |
| `POST /api/cart/quote` | `{items: [{offer_id, qty}]}` → each line priced from the database with a `status` (`ok`, `not_enough_stock`, `sold_out`, `unavailable`), plus `total_cents` and `all_available`. Public |
| `POST /api/orders` | 🔒 Header `Idempotency-Key` + `{items}` → 201 pending order with keys reserved for 15 minutes (`expires_at`). Same key again → 200 with the same order. 409 `out_of_stock` if any line can't be filled (nothing is saved). Starting a new checkout releases your previous unpaid order |
| `POST /api/orders/{id}/pay` | 🔒 `{card_number}` → 200 paid order with keys. 402 `payment_declined` (keys released, order `failed`), 409 `reservation_expired`, 400 `invalid_card`. Paying a paid order again is harmless |
| `GET /api/orders` | 🔒 Your 50 latest orders |
| `GET /api/orders/{id}` | 🔒 One of your orders; `keys` filled once paid. Other people's orders → 404 |

🔒 = needs a signed-in session (401 otherwise).

`GET /api/products` parameters (all optional):

| Param | Example | Notes |
|---|---|---|
| `q` | `eldn ring`, `gta` | Typo-tolerant search: substring match **or** trigram word similarity (`pg_trgm`), after expanding common short names (`gta`, `cod`, `bg3`, `rdr2`, `fifa`… see `backend/internal/catalog/synonyms.go`). Max 100 chars |
| `platform` | `steam` or `steam,xbox` or repeated | Any of the listed platforms |
| `region` | `global` | Same rules as `platform` |
| `type` | `gift_card` | `game`, `dlc`, `gift_card`, `subscription`; same rules as `platform` |
| `min_price`, `max_price` | `10`, `29.99` | Euros; compared with the cheapest in-stock offer |
| `sort` | `relevance` (default), `popular`, `price_asc`, `price_desc`, `newest` | Relevance without `q` = popular |
| `page`, `per_page` | `2`, `24` | `per_page` 1–60, default 24 |

Response: `{ items, total, page, per_page, total_pages, facets: { platforms: [{slug,name,count}], regions: [...], types: [...] } }`.

Only products that can be bought right now are listed (an active offer with at least one usable key); `price_cents` is the cheapest such offer. Facet counts apply every filter **except their own**, so ticking "Steam" still shows how many Xbox results you would get by adding Xbox.

Demo account: `demo@eneba.test` / `demo1234`.

## Seed data

`make seed-gen` runs a Go program (standard library only) that writes `backend/seed/seed.sql`. It uses a fixed random seed, so the output is identical every run.

| | |
|---|---|
| Products | 200 (171 games, 6 DLC, 16 gift cards, 7 subscriptions) |
| Platforms | Steam, EA App, Ubisoft Connect, Battle.net, Xbox, PlayStation, Nintendo |
| Regions | Global, Europe, North America, United Kingdom, Latin America |
| Sellers | 8, rated 4.36–4.97 |
| Offers | 776 (1–8 per product), priced 40–95% of retail for games, around face value for gift cards |
| Keys | ~4,500 codes; ~8% of offers sold out, ~2% inactive |
| Homepage | 6 blocks (banner, bestsellers, deals, new releases, gift cards, subscriptions), 3 banners |

Game titles are real so search feels realistic. Prices, sellers, keys and descriptions are invented. No cover images are seeded (`image_url` is null); the frontend will render placeholder covers.

## Data model decisions

- **A product is one sellable variant** (title + platform + region), as on Eneba: "Elden Ring (PC) Steam Key GLOBAL". `title` keeps the base name for grouping.
- **Money is integer cents.**
- **Stock = number of available keys**, never a stored counter. An expired reservation (`status = 'reserved' AND reserved_until < now()`) counts as available.
- **`order_items` copies the price and seller** at purchase time.
- **Search:** `pg_trgm` GIN index on `lower(name)` for typos and partial names, plus a generated `tsvector` column.
- **Sessions** store only a SHA-256 hash of the cookie token.

## Deployment

| Part | Host | Notes |
|---|---|---|
| Database | Neon (Postgres, Frankfurt) | Migrations and seed applied with the same Docker images as locally (`migrate/migrate`, `psql`) |
| API | Render web service (Docker, Frankfurt, free) | Root directory `backend`, built from `backend/Dockerfile`; `DATABASE_URL` set as a Render environment variable; health check `/api/healthz` |
| Frontend | Vercel (Vite preset) | Root directory `frontend`; `frontend/vercel.json` rewrites `/api/*` to the Render API, so the browser only ever talks to one origin |

Every push to `main` redeploys both Render and Vercel.

## Accounts and security

- **Passwords** are hashed with bcrypt (cost 10). They're never stored or logged in plain text.
- **Sessions:** on sign-in the server creates a random 256-bit token and sends it in a cookie. The `sessions` table stores only its SHA-256 hash, so a database leak doesn't hand out working cookies. Sign-out deletes the row, so the cookie stops working immediately (unlike a JWT). Sessions last 30 days.
- **The cookie** is `HttpOnly` (page JavaScript can't read it, so an XSS bug can't steal it), `SameSite=Lax` (not sent on POSTs from other sites) and `Secure` in production.
- **Cross-site requests:** POST endpoints only accept `Content-Type: application/json`, which a plain HTML form on another site can't send.
- **No account enumeration:** an unknown email and a wrong password give the same 401 message, and take the same time, because unknown emails are checked against a dummy bcrypt hash.
- **Redirects after sign-in** (`/login?next=…`) only accept paths on this site, so the login page can't send people to `//evil.com`.
- **Protected pages:** `RequireAuth` on the frontend only hides pages. The real protection is `RequireUser` on the backend, which answers 401 without a valid session.
- **Not done (would be next):** rate limiting login attempts, email verification, password reset.

## Checkout and key reservation

The part of the app that has to be right: **one key must never be sold to two buyers.**

1. **The cart lives in the browser** (`localStorage`), so guests can fill it. It only stores offer ids and quantities the server cares about; `POST /cart/quote` reprices it from the database every time, so a tampered cart can't change what anyone pays.
2. **`POST /orders` runs one transaction:** read current prices, insert the order and its lines (price and seller copied, so history stays correct), then reserve keys with:
   ```sql
   UPDATE product_keys SET status = 'reserved', reserved_until = $3, order_item_id = $2
   WHERE id IN (SELECT id FROM product_keys
                WHERE offer_id = $1 AND <usable>
                ORDER BY id LIMIT $4
                FOR UPDATE SKIP LOCKED)
   ```
   `FOR UPDATE` locks the chosen keys until the transaction ends; `SKIP LOCKED` makes simultaneous checkouts take *different* free keys instead of waiting. If any line gets fewer keys than asked, the whole transaction rolls back.
3. **Idempotency-Key:** the browser sends one random key per checkout attempt. A double click or a retry returns the same order instead of reserving twice (enforced by a `UNIQUE (user_id, idempotency_key)` constraint, so even two requests at the same instant make one order).
4. **Reservations expire after 15 minutes without a background job:** a key counts as free once `reserved_until` has passed, and unpaid orders are marked `expired` when they're read. A user also has at most one unpaid order; starting a new checkout releases the old one's keys.
5. **Pay** locks the order row (so two Pay clicks can't both charge), checks the order still holds all its keys, then charges through a `PaymentProvider` interface. The simulated provider approves `4242 4242 4242 4242` and declines `4000 0000 0000 0002`. On success the keys become `sold`; on a decline they're released immediately.

**Tested** with 40 buyers ordering the same offer at the same instant, on offers with 3 keys left, repeated on 5 different offers. Every time exactly 3 got an order, 37 got `out_of_stock`, and the 3 keys handed out were all different. 40 identical requests from one user with the same Idempotency-Key created exactly one order. Afterwards the whole database had no key linked to two order lines, and no sold key without a paid order.

**Not done (would be next):** real payments (Stripe test mode would be a new `PaymentProvider`), emailing keys, refunds.

## Homepage blocks

The homepage layout lives in the database, not in the frontend code:

- `home_blocks` lists the blocks: type, title, position, an `is_active` switch and a JSON `config` (for example `{"limit": 12, "min_discount_pct": 40}`).
- `GET /api/home/layout` returns the active blocks in order. The frontend then fetches **each block separately** from `GET /api/home/blocks/{id}`.
- Every block type has its own **provider** behind one Go interface (`home.Provider`): `banner` (its own `banners` table, with optional start and end dates), `best_sellers`, `deals`, `new_releases` and `by_type`. Each runs its own query and keeps one card per game, so the same title doesn't fill a row three times.
- Because blocks load independently, a slow or failing block shows its own error card with "Try again", and the rest of the page works. (Tested by breaking one block's config in the database.)
- Reordering, retitling or hiding a block is a database change, with no deploy. (Tested: hiding "New releases" and moving "Subscriptions" to the top changed the page on reload.)

**Trade-off vs. the brief:** the brief says each block should get its content "from its own backend service or database". Here the providers are separate modules in one Go service rather than separate deployed services. On free hosting, several services would mean several cold starts and deploys without showing much more. Each provider can still be moved into its own service by replacing it with an HTTP call; the frontend wouldn't change.

