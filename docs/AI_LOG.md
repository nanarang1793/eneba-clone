# AI Assistant Log — Eneba Intern Assignment 2027

Full log of prompts sent to the AI assistant (Claude) and its responses while building the project.
New sessions are appended at the bottom.

---

## Session 1 — 2026-11-16

### Prompt 1

> Keep a full log of our conversation (my prompts and your responses) in `docs/AI_LOG.md`, appended after every session.

### Response 1

Created `docs/AI_LOG.md` with this structure: one section per session, and each prompt quoted verbatim with the assistant's reply after it. The project repository wasn't in the assistant's workspace, so it asked where the repo lives (a folder on the user's computer, or a GitHub repo) so the log can be committed alongside the code.

---

### Prompt 2

> [Attached: Eneba_Internship_Assignment.pdf]
> I'm doing this internship assignment (pasted below). I want to build it with React + TypeScript and a Go backend with PostgreSQL. Before writing any code, propose: (1) the scope you'd recommend and what to cut, (2) the data model, (3) the REST API endpoints, (4) the folder structure, (5) a build order. Explain the trade-offs. Don't write code yet

### Response 2

Here's the plan. It's built around the brief's point that a few features done in depth beat many done minimally.

#### 1. Scope

**Go deep on these (this is where the interesting engineering is):**
- **Store + Search.** One query engine powers both. It handles filters (platform, region, price range), sorting, pagination and filter counts. Search tolerates typos and partial names using Postgres `pg_trgm`. Filters live in the URL, so a result page can be shared and the back button works.
- **Product page with seller offers.** Offers are sorted by price, with seller rating and stock for each, and the best price highlighted.
- **Checkout that hands out real keys.** Keys are actual database rows. Checkout reserves them inside a transaction (`SELECT … FOR UPDATE SKIP LOCKED`), so two buyers can never get the same key and an offer can't oversell. This is the part most worth being able to explain.
- **Data-driven homepage.** The block layout comes from the database. Each block has its own backend provider and endpoint, and the frontend just renders whatever block types it receives.

**Build properly but keep small:**
- **Auth:** email and password, bcrypt hashing, a session cookie. Checkout and orders are protected routes.
- **Eneba styling:** dark purple theme, yellow call-to-action buttons, Eneba-style product cards, a header with search. Desktop first, then one responsive pass.

**Cut (and say so in the README):**
- **Seller accounts, dashboards, creating listings:** sellers and offers are seeded. The brief describes the marketplace from the buyer's side.
- **Admin panel:** content comes from seed data and migrations.
- **Real payments:** a simulated provider sits behind an interface, and test card numbers succeed or fail. Stripe test mode is a stretch goal.
- **Emailing keys:** keys show on the order page instead.
- **Account extras:** password reset, OAuth and email verification.
- **Reviews, wishlist, multi-currency (EUR only), translations, refunds.**
- **Real microservices:** see trade-off 1.

#### 2. Data model (PostgreSQL)

```
users         id uuid PK, email citext UNIQUE, password_hash, display_name, created_at
sessions      token_hash PK, user_id FK, expires_at, created_at

platforms     id, slug, name                  -- Steam, Xbox, PSN, Nintendo, EA App…
regions       id, slug, name                  -- Global, Europe, North America…
products      id, slug UNIQUE, name, description, image_url,
              type (game|gift_card|dlc|subscription),
              platform_id FK, region_id FK, msrp_cents, release_date, publisher,
              search_vector tsvector GENERATED, created_at
sellers       id, slug, name, rating numeric(3,2), sales_count, created_at
offers        id, product_id FK, seller_id FK, price_cents, is_active, created_at
              UNIQUE (product_id, seller_id)
product_keys  id, offer_id FK, code, status (available|reserved|sold),
              reserved_until, order_item_id FK NULL

orders        id uuid, user_id FK, status (pending|paid|failed|expired),
              total_cents, currency, idempotency_key UNIQUE, payment_ref,
              created_at, paid_at
order_items   id, order_id FK, offer_id FK, product_id, seller_id,
              unit_price_cents, quantity       -- snapshot at purchase time

home_blocks   id, type (banner|best_sellers|deals|new_releases|…), title,
              position, config jsonb, is_active
banners       id, title, subtitle, image_url, cta_label, cta_url,
              position, starts_at, ends_at
```

Key decisions:
- **A product is one sellable variant** (game + platform + region), the way Eneba lists them ("Elden Ring (PC) Steam Key GLOBAL"). Filters and joins stay simple. Downside: the same game on three platforms is three rows, and grouping editions under one parent game is cut.
- **Money is stored as integer cents,** never floats.
- **Stock is the count of available keys,** not a separate number column, so it can't drift out of sync.
- **`order_items` copies the price and seller,** so order history stays correct if an offer changes later.
- **Cheapest price per product** (for sorting and price filters) is computed with a view or lateral join at first. If that gets slow, store `min_price_cents` on `products` and keep it updated with a trigger. A few hundred products don't need it.
- **Indexes:** GIN trigram on `lower(name)`, GIN on `search_vector`, B-tree on `offers(product_id, price_cents)` and `product_keys(offer_id, status)`.
- **`msrp_cents`** lets the Deals block show "-35%" the way Eneba does.

#### 3. REST API

Everything is under `/api` and returns JSON. Errors always have the same shape: `{ "error": { "code", "message" } }`.

**Auth**
- `POST /auth/register`
- `POST /auth/login` sets an httpOnly session cookie
- `POST /auth/logout`
- `GET /auth/me`

**Catalog & search**
- `GET /products?q=&platform=&region=&min_price=&max_price=&sort=relevance|price_asc|price_desc|newest|popular&page=&per_page=` returns `{ items, total, facets }`
- `GET /filters` returns platforms, regions and price bounds for building the filter UI
- `GET /products/{slug}`
- `GET /products/{slug}/offers?sort=price`
- `GET /search/suggest?q=&limit=6` returns light results for the header dropdown

**Homepage**
- `GET /home/layout` returns an ordered list `[{ id, type, title }]`
- `GET /home/blocks/{id}` returns that block's data, produced by its own provider

**Cart & checkout** (the cart itself lives in the browser)
- `POST /cart/quote` with `{ items: [{ offer_id, qty }] }` returns current prices, availability and total
- `POST /orders` (signed in, `Idempotency-Key` header) creates a pending order and reserves keys for 15 minutes
- `POST /orders/{id}/pay` (signed in) runs the simulated payment. On success the keys become sold and the order paid.
- `GET /orders` (signed in) returns order history
- `GET /orders/{id}` (owner only) includes the keys once the order is paid

**Ops**
- `GET /healthz`

#### 4. Folder structure

```
eneba-clone/
├── backend/
│   ├── cmd/api/main.go            # wiring: config, DB pool, router, start server
│   ├── internal/
│   │   ├── config/
│   │   ├── httpx/                 # router, middleware (auth, CORS, logging, recover), error helpers
│   │   ├── auth/                  # handlers, service, session store
│   │   ├── catalog/               # product list/detail, filters, offers
│   │   ├── search/                # suggest + ranking query
│   │   ├── home/
│   │   │   ├── layout.go
│   │   │   └── blocks/            # banner.go, best_sellers.go, deals.go … (one provider each)
│   │   ├── orders/                # cart quote, order creation, key reservation
│   │   ├── payments/              # Provider interface + simulated implementation
│   │   └── db/                    # sqlc-generated code + queries/*.sql
│   ├── migrations/                # goose SQL migrations
│   ├── seed/                      # seed generator (products, sellers, offers, keys)
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── app/                   # router, providers, layout shell
│   │   ├── api/                   # fetch client + typed endpoint functions
│   │   ├── features/
│   │   │   ├── home/              # BlockRenderer + one component per block type
│   │   │   ├── store/             # filters, sort, product grid
│   │   │   ├── search/            # header search + suggestions
│   │   │   ├── product/           # details + offers table
│   │   │   ├── cart/
│   │   │   ├── checkout/
│   │   │   ├── orders/
│   │   │   └── auth/
│   │   ├── components/            # ui/ (Button, Price, Badge, Card…), layout/ (Header, Footer)
│   │   ├── styles/                # Eneba design tokens
│   │   └── types/
│   └── vite.config.ts
├── docs/
│   ├── AI_LOG.md
│   └── ARCHITECTURE.md
├── docker-compose.yml             # local Postgres
└── README.md                      # setup, scope decisions, what was cut and why
```

Stack picks:
- **Backend:** `chi` router, `pgx` v5, `sqlc` for type-safe queries, `goose` for migrations.
- **Frontend:** Vite, React Router, TanStack Query for server data, Zustand for the cart, Tailwind configured with Eneba's colors and spacing as design tokens.

#### 5. Build order (7 days)

| Day | Work | Why this order |
|---|---|---|
| 1 | Repo skeleton, docker-compose Postgres, migrations, seed (~200 products, ~10 sellers, offers, keys), `/healthz`, empty Vite app. **Deploy the empty app end to end on day 1.** | Every later feature needs data. Finding deploy problems on day 7 is the classic way to fail. |
| 2 | Catalog API (filters, sort, pagination, facets) + Store page, header/footer, design tokens | The core screen; reused by search and homepage |
| 3 | Search (trigram + full-text ranking, suggest endpoint, debounced header dropdown) + Product page with offers | Builds directly on the catalog queries |
| 4 | Auth (register, login, sessions, `/me`) + protected routes | Checkout depends on it |
| 5 | Cart, quote, order creation with key reservation, simulated payment, order history with keys | Hardest part; done once its dependencies exist |
| 6 | Homepage blocks (layout + providers), Eneba styling polish, loading/empty/error states, responsive pass | Blocks reuse product cards and need order data for Best Sellers |
| 7 | Tests on risky paths (key reservation under concurrency, search ranking, auth), README with scope justification, final deploy, log | Buffer for whatever slipped |

#### Trade-offs

1. **Homepage blocks: one app split into modules vs. real microservices.** The brief says each block gets content "from its own backend service or database." I'd use one Go app where each block is a separate provider behind a common interface, with its own table or query and its own endpoint. The frontend fetches each block separately, so one failing block shows an error card instead of breaking the page. Real separate services would match the wording more literally. But on free hosting that means several cold starts and deploys, plus network plumbing, without showing much more. The README should say that a provider can become its own service without the frontend changing.
2. **Search: Postgres `pg_trgm` vs. a dedicated engine (Meilisearch, Typesense, Elasticsearch).** Trigram similarity handles typos ("eldn ring") and partial names well at this data size, and it adds nothing to deploy. A dedicated engine has better typo handling and faster suggestions, but it's another service to host and keep in sync. Start with Postgres, and if search turns out weak, add a search engine as a stretch.
3. **Cart in the browser vs. on the server.** A browser (localStorage) cart lets guests add items without signing in and needs no API. The cost is that the cart doesn't follow you to another device. The server stays in charge of prices: `/cart/quote` and `/orders` re-price everything, so a tampered cart can't change what you pay.
4. **Session cookie vs. JWT.** An opaque token in an httpOnly cookie, stored hashed in `sessions`, can be revoked (logout really logs out), and page JavaScript can't read it. A JWT saves one database lookup per request but is hard to revoke. For this app the session approach is simpler and safer.
5. **sqlc vs. an ORM (GORM).** With sqlc you write plain SQL and get typed Go functions, so every query is visible and explainable, which matters because you have to explain your code. The search ranking and the `FOR UPDATE SKIP LOCKED` reservation are awkward in an ORM anyway. The cost is a little more boilerplate for simple create/read/update/delete operations.
6. **Simulated payment vs. Stripe test mode.** The simulated provider is fully under your control and easy to test. Stripe looks more real but adds webhooks, which are unreliable against a free server that falls asleep. Putting payment behind an interface keeps Stripe as a later drop-in.
7. **Deployment.** Planned setup: Postgres on Neon (Render's free Postgres expires), Go API on Render via Docker, frontend on Vercel with a rewrite that sends `/api/*` to Render. The rewrite makes the API same-origin, so the session cookie is first-party. Cross-site cookies get blocked by Safari and are fiddly in Chrome. One known cost: Render's free tier sleeps when idle, so the first request after a pause is slow. Mention that in the README.
8. **Expiring reservations: checked on read vs. a background job.** Treat a key as available when `status = 'reserved' AND reserved_until < now()`. That needs no cron job and can't silently stop working. A goroutine that cleans up periodically can be added for tidiness.

Images: don't copy Eneba's cover art. Use placeholder art or a free game database API (e.g. RAWG) with attribution, and make the Eneba look come from colors, typography and component shapes.

---

### Prompt 3

> Set up the monorepo (frontend/ and backend/), Docker Compose with Postgres, migrations, and seed data: ~200 fake products, 5-10 sellers, offers with different prices, platforms and regions.

### Response 3

*(Work done: created the repo layout, `docker-compose.yml`, `Makefile`, 5 golang-migrate migrations, a Go seed generator → `backend/seed/seed.sql`, a stdlib-only `/api/healthz` server and a Vite/React/TS placeholder. Tested migrations up → down → up and loaded the seed twice on a local Postgres 16. Bug found and fixed during testing: Go's `Fprintf` treated the `%L`/`%I` in a SQL `format()` call as Go verbs.)*

The monorepo is set up, and the database side is tested against a real Postgres 16.

**What's in it**
- **`docker-compose.yml`**: Postgres 16, plus `migrate` (official golang-migrate image) and `seed` (psql) services, so nothing extra needs installing. `make db-reset` does a full rebuild: fresh database, migrations, seed.
- **5 migrations** with up and down: extensions (`citext`, `pg_trgm`, `pgcrypto`), catalog, users and sessions, orders, homepage blocks. Tested up → down (0 tables left) → up again.
- **Seed:** 200 products (171 games, 6 DLC, 16 gift cards, 7 subscriptions) across 7 platforms and 5 regions. There are 8 sellers and 776 offers (1–8 per product). Game keys are priced at 40–95% of retail and gift cards around face value. About 4,500 key codes; about 8% of offers are sold out and 2% inactive. Also 6 homepage blocks, 3 banners and a demo user (`demo@eneba.test` / `demo1234`). The seed runs twice in a row cleanly, and the same random seed always gives the same file.
- **Search check:** a trigram query for `eldn ring` returns the Elden Ring products, and `witchr` finds The Witcher 3.

**Changes from the plan**
- **Migrations use golang-migrate instead of goose.** Its Docker image runs migrations from Compose with nothing installed locally, and the plain `.up.sql`/`.down.sql` files also work with psql.
- **The seed is a generated SQL file,** not a Go program that writes to the database. Loading it needs only `psql`, and the file shows exactly what data the app starts with.
- **`products.title` and `products.sales_count` were added.** `title` lets variants be grouped and shown by base name. `sales_count` gives Bestsellers and "popular" sorting data before any real orders exist.
- **The health-check server uses Go's standard router for now.** The `chi` router and `pgx` driver come in with the catalog API.

**Not verified here:** this workspace has no internet, so Go modules and npm packages can't be downloaded. The frontend is hand-written Vite + React + TS config and hasn't been installed or built. Run `npm install && npm run dev` first. `vercel.json` has a placeholder Render URL to fill in after the first deploy.

---

## Session 2 — 2026-11-16

### Prompt 1

> (attached: the Session 1 chat transcript and `Eneba Internship Assignment.pdf`) waht next?

### Response 1

The assistant read the brief and the transcript, then looked for the project on the user's computer. `eneba-clone.zip` had only existed in the previous chat's workspace and wasn't on the PC. It suggested these next steps, in order:

1. **Get the code onto the computer.** Download `eneba-clone.zip` from the previous chat and unzip it somewhere permanent, so the assistant can work in that folder and keep `docs/AI_LOG.md` up to date there.
2. **Put it on GitHub now, not on day 7.** Create an empty repo and push the folder, so reviewers can see the history day by day.
3. **Check it runs locally.** Install Docker Desktop, Go 1.22+ and Node 20+. Then run `docker compose up -d` and `make db-reset` (or the `docker compose run` equivalents on Windows), then `go run ./cmd/api` (check `/api/healthz`), then `npm install && npm run dev`. The frontend has never been built.
4. **Deploy the empty app** (Neon + Render + Vercel), which was the last part of day 1. Deploy problems are cheaper to find now.
5. **Then day 2:** the catalog API (filters, sort, pagination, facet counts with chi and pgx) and the Store page.

### Prompt 2

> i downloaded both of them

### Response 2

The assistant found `eneba-clone.zip` and `AI_LOG.md` in Downloads. The zip already holds the complete Session 1 log; the separate `AI_LOG.md` is an older one-exchange copy and can be deleted. It unzipped the project to `Downloads\eneba-clone` and appended this session to `docs/AI_LOG.md`.

It also tested whether day 2 could be built in its workspace. It can't: package downloads are blocked there (no Go module proxy and no npm registry), so `chi`, `pgx`, React and Vite can't be installed or compiled. Day 2 code written there would be untested. To unblock this, the user can either allow package installs in Claude's network settings (the change applies to a new chat), or run the toolchain on their own PC and report the output.

---

## Session 3 — 2026-11-16

### Prompt 1

> [Attached: Makefile] *(no message)*

### Prompt 2

> Continue my Eneba internship project in `Downloads\eneba-clone`. Read `docs/AI_LOG.md` first. It has the assignment, the agreed plan (scope, data model, API, folder structure, 7-day build order) and what's done so far (Day 1: monorepo, Docker Compose Postgres, migrations, seed).
> Do Day 2 now:
>
> 1. Backend: add chi + pgx, and build `GET /api/products` (filters for platform, region, min/max price; sort by relevance, price_asc, price_desc, newest, popular; pagination; facet counts), `GET /api/filters`, and the shared error format. Use the trigram search for `q`.
> 2. Frontend: install the deps (React Router, TanStack Query, Tailwind with Eneba design tokens) and build the Store page: Eneba-style header with search, filter sidebar, sort dropdown, product grid, pagination, with filters kept in the URL.
> 3. Actually run it: start Postgres, migrate, seed, run the API and the frontend build, and test the endpoints with real queries. Fix whatever breaks.
> 4. Write the code into my folder, tell me the commands to run it on my Windows PC, and append this session to `docs/AI_LOG.md`.
>
> Keep the code simple enough that I can explain every part of it.

### Response 2

*(Work done: read the log and the Day 1 code; built the backend and frontend in the assistant's workspace; ran everything against a real Postgres 16; wrote the files back to `Downloads\eneba-clone`.)*

**Backend** (`backend/`)
- `go.mod` now uses **chi v5.2.1** and **pgx v5.7.5** (`pgxpool`).
- `internal/httpx/httpx.go`: `JSON()`, the shared error format `{ "error": { "code", "message" } }` with codes `invalid_parameter`, `not_found`, `method_not_allowed`, `internal`; JSON 404/405 handlers; a recoverer that turns panics into a JSON 500. 500s log the real error and send the client a generic message.
- `internal/catalog/`:
  - `params.go` validates the query string (sort whitelist, `page` 1–1000, `per_page` 1–60, prices in euros converted to integer cents, `min_price ≤ max_price`, `q` ≤ 100 chars). `platform`/`region` accept `steam,xbox` or repeated params.
  - `store.go` holds all the SQL. A `listing` CTE gives one row per product that can be bought now (active offer with a usable key, the same "expired reservation counts as available" rule as the schema) with its cheapest price. A tiny `where` builder adds conditions and returns `$1, $2…` placeholders, so user input never goes into the SQL text. Search is `lower(name) LIKE '%q%' OR q <% lower(name)` (substring for short words like "fifa", trigram word similarity for typos); relevance sorts by `word_similarity`, then sales. Every sort ends with `id` so pages never overlap.
  - Facet counts start from the full platforms/regions table and `LEFT JOIN` the filtered listing, applying every filter *except their own* (so ticking Steam still shows how many Xbox results there would be, and zero-count options still appear).
  - `handler.go` only parses → calls the store → writes JSON. `types.go` holds the response structs.
- `cmd/api/main.go`: config from env (defaults to the local Docker DB), pgx pool + ping at start-up, chi router with RequestID, RealIP, Logger, Recoverer and a 10 s timeout. `/api/healthz` now also pings the database.

**Frontend** (`frontend/src/`)
- Added `react-router` 7, `@tanstack/react-query` 5, `tailwindcss` 4 + `@tailwindcss/vite`.
- `styles/index.css`: Eneba design tokens as Tailwind v4 `@theme` variables (deep purple `#1F0A4D`, lavender `#A182E8`, orange `#FD5A20` from Eneba's brand palette, plus a yellow CTA). They become classes like `bg-page`, `bg-cta`, `text-muted`.
- `api/client.ts`: one `apiGet()` that throws an `ApiError(status, code, message)` built from the shared error format.
- `features/store/useStoreParams.ts`: **the URL is the only filter state.** The URL uses the same parameter names as the API, so the API request is the URL's query string (known keys only). Changing a filter resets to page 1; empty values are removed from the URL.
- `StorePage.tsx`: TanStack Query keyed by that query string, with `keepPreviousData` so old results stay (dimmed, `aria-busy`) while new ones load. Loading skeletons, an empty state with "Clear filters", and an error box that shows the backend's message.
- `FilterSidebar.tsx` (price form with Apply, platform/region checkboxes with counts, Clear all), `SortSelect.tsx`, `ProductCard.tsx` (placeholder cover coloured from the title, region tag, struck-through retail price, discount badge, "From €x", offer count), `Pagination.tsx` (`1 … 4 5 6 … 9`), `Header.tsx` (search goes to `/store?q=…`). `/` redirects to `/store` until the homepage on day 6. On phones the filters hide behind a "Filters (n)" button.

**Testing done**
- Postgres 16 (local binaries; the workspace has no Docker daemon), migrations 1–5 and `seed.sql` applied with psql: 200 products, 776 offers, 198 listable.
- API tested with curl: typo search (`eldn ring` → 6 Elden Ring products, `witchr` → 5 Witcher 3), multi-platform + region + price filters, all 5 sorts, pagination past the end (empty items, correct total), and every validation error (400s, 404, 405 all in the shared format).
- Cross-checked one filtered, sorted, paginated listing (Steam + Xbox, Global, price ascending, 5 pages) against an independent SQL query: same 42 products in the same order, no duplicates.
- Browser test (Playwright + headless Chrome) of 13 steps: redirect, tick Steam (48) then Xbox (112, PlayStation still shows 45), sort, price 10–30 (64 results, all within range, sorted across page 1 → 2), back button, reload keeps every filter, Clear all, header search, empty state, error box, 404 page. No console errors. `tsc` + `vite build` pass.

**Bugs found and fixed while testing**
- Setting min and max price with two separate URL updates loses the first one (both start from the same old URL). Added `setPriceRange()` that sets both in one update.
- Favicon 404 in the console; added an inline SVG favicon.
- Mobile layout put all filters above the products; they now collapse behind a button.
- Two test-only issues (Playwright reading the grid before the new response arrived; the header's placeholder links sharing a CSS class with the loading state) led to adding `aria-busy` to the results grid, which is also better for screen readers.

**Changes from the plan**
- **No sqlc (for now).** The product list has optional filters and five sort orders, which sqlc handles badly (you end up with many near-identical queries or `CASE WHEN` tricks). Plain pgx with a 20-line `where` builder is easier to explain. sqlc can still be used for the fixed queries later (auth, orders).
- **Tailwind v4:** design tokens live in CSS (`@theme`), not `tailwind.config.js`.
- **Search runs on the `listing` CTE**, so the trigram index isn't used. At 200 products a whole `/api/products` request (4 queries) took 8 ms median and 23 ms at worst in testing; at scale, the `q` filter would move into the CTE's `WHERE` so the index kicks in.

**Workspace notes:** the Go module proxy, `golang.org` and `gopkg.in` are blocked in the assistant's workspace, but GitHub isn't. The dependencies were built from the official GitHub mirrors with Go's own `golang.org/x/mod/zip`; the resulting `go.sum` hashes match the ones published in pgx's own `go.sum` exactly, so `go` on the user's PC verifies them normally.

**Next (day 3):** product page with offers, `/search/suggest` and the header dropdown, and search synonyms ("gta" currently finds nothing because no product name contains it).

### Prompts 3–6 (running it on Windows)

> *(screenshots of PowerShell errors, in order)* `docker compose up -d db` → "no configuration file provided"; then "failed to connect to the docker API … docker_engine"; then `go run ./cmd/api` → "'go' is not recognized"; then, after installing Go 1.27.4, "error obtaining buildID for go tool asm: fork/exec C:\Program Files\Go\pkg\tool\windows_amd64\asm.exe: An Application Control policy has blocked this file"; then `npm` → "not recognized".

### Responses 3–6

- Wrong folder: PowerShell opened in `C:\WINDOWS\System32`; `cd $HOME\Downloads\eneba-clone` first.
- Docker Desktop wasn't running; start it and wait for "Engine running". After that, the database, all 5 migrations and the seed worked.
- Go wasn't installed; installed with winget, then opened a new terminal.
- **Windows Smart App Control** blocks Go's unsigned build tools, so `go run` can't work on this PC without turning it off (which on most Windows 11 versions can't be undone without resetting Windows). Instead, `docker-compose.yml` got an `api` service that builds `backend/Dockerfile` (the deploy image), so the API runs in Linux inside Docker. The build worked on the user's PC.
- Node wasn't installed either, and Vite's own unsigned helpers (esbuild, Rollup) would likely be blocked the same way, so a `web` service now runs the Vite dev server in a `node:22-alpine` container. The `frontend/` folder is mounted in, `node_modules` lives in a Docker volume (Linux packages must not mix with a Windows `node_modules`), and the API is reached as `http://api:8080`. `vite.config.ts` got a `watch.usePolling` option, switched on by `VITE_USE_POLLING`, because file-change events from Windows don't reach the container. Tested in the assistant's workspace: fresh `npm install`, the proxy reaching the API, the store page loading 198 results, and hot reload picking up a saved file.

### Prompts 7–20 (GitHub and deployment)

> *(step-by-step questions while putting the project on GitHub and deploying it: installing Git, a push to the placeholder `YOUR-NAME` URL, removing the nested duplicate `eneba-clone/eneba-clone` folder, setting up Neon, Render and Vercel, plus screenshots of each screen)*

### Responses 7–20

- **GitHub:** installed Git (`winget install Git.Git`, then a new terminal), deleted the stale nested copy and the old `frontend/src/App.tsx`, amended the first commit so the duplicate never reached GitHub, and pushed to `github.com/nanarang1793/eneba-clone`.
- **Neon:** project in AWS Europe Central 1 (Frankfurt). Used the **direct** connection string (no `-pooler`), because pgx caches prepared statements, which a transaction-mode pooler can break, and dropped `&channel_binding=require`, which golang-migrate's driver doesn't support (`sslmode=require` still encrypts). The first seed attempt failed with `relation "product_keys" does not exist` because migrations hadn't been run yet; after `migrate up` and the seed, `select count(*) from products` returned 200. The database password appeared in a screenshot, so it was reset in Neon before use.
- **Render:** web service from the GitHub repo, Docker runtime, root directory `backend`, free instance in Frankfurt, `DATABASE_URL` as a secret environment variable, health check `/api/healthz`. The first deploy took 1 min 11 s; logs showed `api listening on :10000` and health checks answering 200. Opening the bare domain shows "Not Found", which is expected because every route lives under `/api`. Live at https://eneba-clone-d0ia.onrender.com/api/products.
- **Vercel:** the import first picked `backend` as the root directory with the Go preset; changed to `frontend` (Vite preset). The assistant put the Render URL into `frontend/vercel.json` so `/api/*` is rewritten to the API. Deployed in 11 s; the preview showed the store with 198 results and live facet counts, confirming the full chain Vercel → Render → Neon. Live at https://eneba-clone-nine.vercel.app/store.
- That completes the Day 1 leftovers (code on GitHub, app deployed). From now on every push redeploys both hosts.

## Session 4 — 2026-11-16 (Day 3)

### Prompt 1

> start day 3

### Response 1

*(Work done: built and tested in the assistant's workspace against Postgres 16 with the project's migrations and seed, then wrote the files into `Downloads\eneba-clone`.)*

**Backend** (`backend/internal/catalog/`)
- `product.go` (new):
  - `ProductBySlug` loads the product with a `LEFT JOIN LATERAL` that finds the cheapest in-stock price and the number of sellers with stock. Sold-out products still load (`price_cents: null`, `in_stock: false`), so old links don't break.
  - It also lists the same title on other platforms/regions (`variants`).
  - `OffersBySlug` lists every active offer with the seller's rating and sales, and stock counted from usable keys. Order: in stock first, then price, then seller rating. Sold-out offers stay in the list.
  - The first row is flagged `is_best` only if it's in stock. Example: Assassin's Creed Shadows (PS5) has a cheaper BaltiKeys offer that's sold out, and the best price correctly goes to LootLane.
- `synonyms.go` (new): a small map from the short names people type to words that appear in product names (`gta` → "grand theft auto", `cod`, `bg3`, `rdr2`, `fifa` → "ea sports fc", `ac`, `xgp`, `ps`…). Trigram search fixes typos, but it can't know that "gta" means Grand Theft Auto.
- `store.go`:
  - The "usable key" rule (available, or reserved with an expired reservation) is now one named constant, used by every query.
  - The product-card query moved into `queryProducts()`, which both the store page and the new `Suggest()` use.
  - Search text is run through `expandQuery()` first.
- `handler.go`: `GET /api/products/{slug}`, `GET /api/products/{slug}/offers` (both 404 in the shared error format for unknown slugs), and `GET /api/search/suggest?q=&limit=` (limit 1–10; under 2 characters returns an empty list without querying).

**Frontend** (`frontend/src/`)
- `features/product/ProductPage.tsx`:
  - The page has a breadcrumb, the cover, name, tags ("Works in: Europe"), publisher and release date, and links to other platforms/regions with prices.
  - A price box shows the retail price struck through, the discount, "From €x" and a "Compare offers" button that jumps to the offers.
  - An "About" section shows the description.
  - Product and offers are two parallel queries, so each can load or fail on its own. It has loading skeletons, a "Product not found" page, an out-of-stock state, and sets the browser tab title.
- `features/product/OffersTable.tsx`: one row per seller with rating, compact sales count ("313K sales"), stock, price and an "Add to cart" button. The button is disabled until the cart exists on day 5. The best row gets a yellow border and a "Best price" badge; sold-out rows are dimmed.
- `features/search/SearchBox.tsx` + `useDebouncedValue.ts`: the header search now shows suggestions.
  - Requests wait 250 ms after typing stops, so "eldn" typed letter by letter sent exactly one request.
  - The old list stays visible while the next one loads.
  - ↑/↓ highlight a row, Enter opens it (or runs a full search if nothing is highlighted), and the last row is "See all results for…".
  - It follows the ARIA combobox pattern. The list stays open while you click it by preventing the default on `mousedown`, which fires before the input's `blur`.
- `components/ui/Cover.tsx`: the placeholder cover moved here so cards, the product page and the dropdown share it. Product cards are now links to `/product/:slug`.
- `main.tsx`: TanStack Query now retries only network/server errors, never 4xx, since a 404 won't change on a retry.

**Testing done**
- API with curl:
  - The Elden Ring (PC) offers match the database (BaltiKeys €31.69 best, then €37.69, €39.99, €47.99).
  - A sold-out offer sorts last, a fully sold-out product returns `in_stock: false` with no offers, and unknown slugs return 404.
  - `limit=50` returns 400.
  - Suggestions for `el`, `eldn`, `gta`, `gta 5`, `gta v`, `cod`, `bg3`, `rdr2`, `fifa`, `ac valhalla`, `ps plus`, `xgp`, `witcher` and `baldurs gate` all return the right games. `e` (one letter) returns nothing.
- Browser (Playwright + headless Chrome):
  - All 13 Day 2 store checks still pass.
  - 16 new checks pass: card → product page (title, tab title), offers sorted with one "Best price" badge, price box, variant link, breadcrumb back to the store, a dimmed sold-out row, the out-of-stock page, an unknown product, and one debounced request while typing. Also ↓↓+Enter opening the highlighted product, `gta` suggestions, Escape closing the list, a mouse click on a suggestion, "See all results", the "No matches" row, and a 1-character query hiding the list.
  - No console errors other than the intentional 404 test.
  - `tsc` and `vite build` pass; `go vet` is clean.

**Bugs found and fixed while testing**
- Pressing Escape to close the suggestions also **cleared the search text**, because browsers clear `type="search"` inputs on Escape. The first Escape now only closes the list; a second one clears the text as usual.
- An unknown product made 4 failed requests instead of 2, because TanStack Query retried the 404s. It now skips retries for 4xx.
- `new Date("2022-12-03")` is UTC midnight, so the release date showed one day early in US time zones. It's now formatted in UTC.
- Postgres can't use an output alias inside an `ORDER BY` expression (`ORDER BY (stock > 0)`), so offer stock is computed in a `LATERAL` join instead.

**Known trade-off:** typo search for "eldn" also suggests two Skyrim products ("**Elde**r Scrolls" is a close trigram match). Elden Ring ranks first, which is acceptable for typo-tolerant search.

**Next (day 4):** accounts. Register, log in and log out with bcrypt-hashed passwords and an httpOnly session cookie stored hashed in `sessions`, plus `GET /auth/me` and protected routes.
