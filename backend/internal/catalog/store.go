package catalog

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store runs the catalog SQL. All queries are plain SQL so each one can be
// copied into psql and explained line by line.
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store { return &Store{db: db} }

// UsableKey is the SQL condition for a key that can still be sold: it's
// available, or it was reserved but the reservation has expired (see the data
// model notes in the README). Stock everywhere is a count of these.
const UsableKey = `(k.status = 'available' OR (k.status = 'reserved' AND k.reserved_until < now()))`

// listingCTE is the "what can you buy right now" view of the catalog:
// one row per product that has at least one active offer with a usable key,
// with the cheapest such price. Every list/facet/suggest query starts from it.
const listingCTE = `
WITH listing AS (
    SELECT p.id, p.slug, p.name, p.title, p.type::text AS type, p.image_url,
           p.msrp_cents, p.release_date, p.sales_count,
           pl.slug AS platform_slug, pl.name AS platform_name,
           r.slug  AS region_slug,   r.name  AS region_name,
           min(o.price_cents) AS price_cents,
           count(o.id)        AS offer_count
    FROM products p
    JOIN platforms pl ON pl.id = p.platform_id
    JOIN regions   r  ON r.id  = p.region_id
    JOIN offers    o  ON o.product_id = p.id AND o.is_active
    WHERE EXISTS (
        SELECT 1 FROM product_keys k
        WHERE k.offer_id = o.id AND ` + UsableKey + `
    )
    GROUP BY p.id, pl.id, r.id
)`

// ---------- small query builder ----------

// where collects SQL conditions and their arguments. arg() appends a value
// and returns its placeholder ($1, $2, …), so values are never pasted into
// the SQL string (no SQL injection).
type where struct {
	conds []string
	args  []any
}

func (w *where) arg(v any) string {
	w.args = append(w.args, v)
	return "$" + strconv.Itoa(len(w.args))
}

func (w *where) add(cond string) { w.conds = append(w.conds, cond) }

// sql joins the conditions with AND ("TRUE" when there are none).
func (w *where) sql() string {
	if len(w.conds) == 0 {
		return "TRUE"
	}
	return strings.Join(w.conds, " AND ")
}

// buildWhere turns the filters into SQL. `skip` leaves one filter out; that's
// how facet counts work: the platform counts apply every filter *except*
// platform, so ticking "Steam" doesn't make the other platforms show 0.
// It also returns the placeholder of the search text (for relevance sorting).
func buildWhere(p ListParams, skip string) (w *where, qArg string) {
	w = &where{}
	if p.Q != "" {
		q := expandQuery(p.Q) // "gta" -> "grand theft auto"
		// Substring match catches short queries ("fifa") that trigrams miss;
		// word_similarity (<%) catches typos ("eldn ring", "witchr").
		// \, % and _ are escaped so they're matched literally by LIKE.
		like := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q) + "%"
		qArg = w.arg(q)
		w.add(fmt.Sprintf("(lower(l.name) LIKE %s OR %s <%% lower(l.name))", w.arg(like), qArg))
	}
	if len(p.Platforms) > 0 && skip != "platform" {
		w.add("l.platform_slug = ANY(" + w.arg(p.Platforms) + ")")
	}
	if len(p.Regions) > 0 && skip != "region" {
		w.add("l.region_slug = ANY(" + w.arg(p.Regions) + ")")
	}
	if p.MinCents != nil {
		w.add("l.price_cents >= " + w.arg(*p.MinCents))
	}
	if p.MaxCents != nil {
		w.add("l.price_cents <= " + w.arg(*p.MaxCents))
	}
	return w, qArg
}

// orderBy maps a sort option to SQL. `l.id` is the last tie-breaker so the
// order is stable and pagination never shows a product twice.
func orderBy(sort, qArg string) string {
	switch sort {
	case SortPriceAsc:
		return "l.price_cents ASC, l.id"
	case SortPriceDesc:
		return "l.price_cents DESC, l.id"
	case SortNewest:
		return "l.release_date DESC NULLS LAST, l.id"
	case SortRelevance:
		if qArg != "" {
			return "word_similarity(" + qArg + ", lower(l.name)) DESC, l.sales_count DESC, l.id"
		}
	}
	return "l.sales_count DESC, l.id" // popular, and relevance without a search
}

// ---------- queries ----------

func (s *Store) ListProducts(ctx context.Context, p ListParams) (ListResult, error) {
	res := ListResult{Page: p.Page, PerPage: p.PerPage, Items: []Product{}}

	// 1. Total number of matches (for pagination).
	w, qArg := buildWhere(p, "")
	err := s.db.QueryRow(ctx, listingCTE+` SELECT count(*) FROM listing l WHERE `+w.sql(), w.args...).Scan(&res.Total)
	if err != nil {
		return res, fmt.Errorf("count products: %w", err)
	}
	res.TotalPages = (res.Total + p.PerPage - 1) / p.PerPage

	// 2. The current page of items.
	res.Items, err = s.queryProducts(ctx, w, orderBy(p.Sort, qArg), p.PerPage, (p.Page-1)*p.PerPage)
	if err != nil {
		return res, err
	}

	// 3. Facet counts for the sidebar.
	if res.Facets.Platforms, err = s.facet(ctx, p, "platform", "platforms", "platform_slug"); err != nil {
		return res, err
	}
	if res.Facets.Regions, err = s.facet(ctx, p, "region", "regions", "region_slug"); err != nil {
		return res, err
	}
	return res, nil
}

// queryProducts returns product cards from the listing, filtered by w,
// in the given order. Used by the store page and by search suggestions.
func (s *Store) queryProducts(ctx context.Context, w *where, order string, limit, offset int) ([]Product, error) {
	limitArg, offsetArg := w.arg(limit), w.arg(offset)
	rows, err := s.db.Query(ctx, listingCTE+`
        SELECT l.id, l.slug, l.name, l.title, l.type, l.image_url,
               l.platform_slug, l.platform_name, l.region_slug, l.region_name,
               l.price_cents, l.msrp_cents, l.offer_count
        FROM listing l
        WHERE `+w.sql()+`
        ORDER BY `+order+`
        LIMIT `+limitArg+` OFFSET `+offsetArg, w.args...)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()
	items := []Product{}
	for rows.Next() {
		var it Product
		if err := rows.Scan(&it.ID, &it.Slug, &it.Name, &it.Title, &it.Type, &it.ImageURL,
			&it.Platform.Slug, &it.Platform.Name, &it.Region.Slug, &it.Region.Name,
			&it.PriceCents, &it.MSRPCents, &it.OfferCount); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		it.DiscountPercent = discountPercent(it.PriceCents, it.MSRPCents)
		items = append(items, it)
	}
	return items, rows.Err()
}

// Suggest returns up to `limit` best matches for the header search dropdown:
// the same search as the store page, without filters, counts or facets.
func (s *Store) Suggest(ctx context.Context, q string, limit int) ([]Product, error) {
	w, qArg := buildWhere(ListParams{Q: q}, "")
	return s.queryProducts(ctx, w, orderBy(SortRelevance, qArg), limit, 0)
}

// facet counts matching products per platform (or region). It starts from the
// full platforms table and LEFT JOINs the filtered listing, so options with
// zero matches still appear (the UI shows them greyed out).
func (s *Store) facet(ctx context.Context, p ListParams, skip, table, column string) ([]FacetCount, error) {
	w, _ := buildWhere(p, skip)
	// table and column are fixed strings from our own code, never user input.
	rows, err := s.db.Query(ctx, listingCTE+`
        SELECT t.slug, t.name, count(l.id)
        FROM `+table+` t
        LEFT JOIN listing l ON l.`+column+` = t.slug AND `+w.sql()+`
        GROUP BY t.id
        ORDER BY t.id`, w.args...)
	if err != nil {
		return nil, fmt.Errorf("%s facet: %w", skip, err)
	}
	defer rows.Close()
	out := []FacetCount{}
	for rows.Next() {
		var f FacetCount
		if err := rows.Scan(&f.Slug, &f.Name, &f.Count); err != nil {
			return nil, fmt.Errorf("%s facet: %w", skip, err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// Filters returns everything the sidebar needs before any search:
// all platforms and regions, and the cheapest/most expensive listed price.
func (s *Store) Filters(ctx context.Context) (Filters, error) {
	var f Filters
	var err error
	if f.Platforms, err = s.namedList(ctx, "SELECT slug, name FROM platforms ORDER BY id"); err != nil {
		return f, err
	}
	if f.Regions, err = s.namedList(ctx, "SELECT slug, name FROM regions ORDER BY id"); err != nil {
		return f, err
	}
	err = s.db.QueryRow(ctx, listingCTE+`
        SELECT coalesce(min(price_cents), 0), coalesce(max(price_cents), 0) FROM listing`,
	).Scan(&f.Price.MinCents, &f.Price.MaxCents)
	if err != nil {
		return f, fmt.Errorf("price bounds: %w", err)
	}
	f.Sorts = []string{SortRelevance, SortPopular, SortPriceAsc, SortPriceDesc, SortNewest}
	return f, nil
}

func (s *Store) namedList(ctx context.Context, sql string) ([]Named, error) {
	rows, err := s.db.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Named{}
	for rows.Next() {
		var n Named
		if err := rows.Scan(&n.Slug, &n.Name); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// discountPercent is the "-35%" badge: how far below retail the price is.
func discountPercent(price int, msrp *int) int {
	if msrp == nil || *msrp <= price {
		return 0
	}
	return (*msrp - price) * 100 / *msrp
}
