# Eneba clone: Intern Assignment 2027

A digital-key marketplace in Eneba's style. React + TypeScript frontend, Go REST API, PostgreSQL.

**Live demo:** https://eneba-clone-nine.vercel.app/store
**API:** https://eneba-clone-d0ia.onrender.com/api/products

> The API runs on Render's free tier, which sleeps after ~15 minutes without traffic. The first request after a pause can take up to a minute while it wakes up; after that it's fast.

> Status: **day 2.** Catalog API (`/api/products` with filters, sorting, pagination and facet counts; `/api/filters`) and the Store page (search, filter sidebar, sort, product grid, pagination, filters kept in the URL).

## Repo layout

```
backend/
  cmd/api/        HTTP server: config, DB pool, router, middleware
  cmd/seedgen/    deterministic seed generator → seed/seed.sql
  internal/httpx/ JSON + shared error format + panic recoverer
  internal/catalog/ params.go (validation) · store.go (SQL) · handler.go (HTTP) · types.go (JSON)
  migrations/     golang-migrate SQL files (NNNNNN_name.up.sql / .down.sql)
  seed/seed.sql   generated seed data (committed, loaded with psql)
frontend/src/
  api/            fetch client (ApiError) + typed endpoint functions
  app/            routes
  components/     layout (Header, Layout) and ui (Price)
  features/store/ StorePage, FilterSidebar, SortSelect, ProductCard, Pagination, useStoreParams (URL state)
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
docker compose run --rm migrate up      # create tables
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

`GET /api/products` parameters (all optional):

| Param | Example | Notes |
|---|---|---|
| `q` | `eldn ring` | Typo-tolerant search: substring match **or** trigram word similarity (`pg_trgm`). Max 100 chars |
| `platform` | `steam` or `steam,xbox` or repeated | Any of the listed platforms |
| `region` | `global` | Same rules as `platform` |
| `min_price`, `max_price` | `10`, `29.99` | Euros; compared with the cheapest in-stock offer |
| `sort` | `relevance` (default), `popular`, `price_asc`, `price_desc`, `newest` | Relevance without `q` = popular |
| `page`, `per_page` | `2`, `24` | `per_page` 1–60, default 24 |

Response: `{ items, total, page, per_page, total_pages, facets: { platforms: [{slug,name,count}], regions: [...] } }`.

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

