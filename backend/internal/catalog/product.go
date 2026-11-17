package catalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrNotFound is returned when no product has the requested slug.
// The handler turns it into a 404.
var ErrNotFound = errors.New("not found")

// ProductBySlug loads one product for the product page. Unlike the store
// listing, it also returns products that are sold out right now
// (price_cents is then null and in_stock false), so old links keep working.
func (s *Store) ProductBySlug(ctx context.Context, slug string) (ProductDetail, error) {
	var d ProductDetail
	err := s.db.QueryRow(ctx, `
        SELECT p.id, p.slug, p.name, p.title, p.type::text, p.description, p.image_url,
               p.publisher, to_char(p.release_date, 'YYYY-MM-DD'),
               pl.slug, pl.name, r.slug, r.name, p.msrp_cents,
               stock.min_price, coalesce(stock.offers, 0)
        FROM products p
        JOIN platforms pl ON pl.id = p.platform_id
        JOIN regions   r  ON r.id  = p.region_id
        -- cheapest price and number of sellers among offers that have a usable key
        LEFT JOIN LATERAL (
            SELECT min(o.price_cents) AS min_price, count(*) AS offers
            FROM offers o
            WHERE o.product_id = p.id AND o.is_active
              AND EXISTS (SELECT 1 FROM product_keys k WHERE k.offer_id = o.id AND `+UsableKey+`)
        ) stock ON true
        WHERE p.slug = $1`, slug,
	).Scan(&d.ID, &d.Slug, &d.Name, &d.Title, &d.Type, &d.Description, &d.ImageURL,
		&d.Publisher, &d.ReleaseDate,
		&d.Platform.Slug, &d.Platform.Name, &d.Region.Slug, &d.Region.Name, &d.MSRPCents,
		&d.PriceCents, &d.OfferCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, fmt.Errorf("product %q: %w", slug, err)
	}
	d.InStock = d.PriceCents != nil
	if d.PriceCents != nil {
		d.DiscountPercent = discountPercent(*d.PriceCents, d.MSRPCents)
	}

	// Other platforms/regions of the same game ("Elden Ring" on PS5, Xbox…),
	// for the "Choose platform & region" links on the page.
	rows, err := s.db.Query(ctx, listingCTE+`
        SELECT l.slug, l.platform_name, l.region_name, l.price_cents
        FROM listing l
        WHERE l.title = $1 AND l.slug <> $2
        ORDER BY l.platform_name, l.region_name`, d.Title, d.Slug)
	if err != nil {
		return d, fmt.Errorf("variants of %q: %w", slug, err)
	}
	defer rows.Close()
	d.Variants = []Variant{}
	for rows.Next() {
		var v Variant
		if err := rows.Scan(&v.Slug, &v.Platform, &v.Region, &v.PriceCents); err != nil {
			return d, fmt.Errorf("scan variant: %w", err)
		}
		d.Variants = append(d.Variants, v)
	}
	return d, rows.Err()
}

// OffersBySlug lists every active seller offer for a product: in-stock offers
// first, cheapest first, then the better-rated seller on equal prices.
// Sold-out offers stay in the list (greyed out in the UI) so the buyer sees
// who normally sells it.
func (s *Store) OffersBySlug(ctx context.Context, slug string) ([]Offer, error) {
	var productID int64
	err := s.db.QueryRow(ctx, `SELECT id FROM products WHERE slug = $1`, slug).Scan(&productID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("product %q: %w", slug, err)
	}

	rows, err := s.db.Query(ctx, `
        SELECT o.id, o.price_cents,
               s.slug, s.name, s.rating::float8, s.sales_count,
               stock.n
        FROM offers o
        JOIN sellers s ON s.id = o.seller_id
        -- number of usable keys for this offer
        JOIN LATERAL (
            SELECT count(*) AS n FROM product_keys k WHERE k.offer_id = o.id AND `+UsableKey+`
        ) stock ON true
        WHERE o.product_id = $1 AND o.is_active
        ORDER BY (stock.n > 0) DESC, o.price_cents ASC, s.rating DESC, o.id`, productID)
	if err != nil {
		return nil, fmt.Errorf("offers for %q: %w", slug, err)
	}
	defer rows.Close()

	offers := []Offer{}
	for rows.Next() {
		var o Offer
		if err := rows.Scan(&o.ID, &o.PriceCents,
			&o.Seller.Slug, &o.Seller.Name, &o.Seller.Rating, &o.Seller.SalesCount, &o.Stock); err != nil {
			return nil, fmt.Errorf("scan offer: %w", err)
		}
		o.InStock = o.Stock > 0
		offers = append(offers, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Because of the ORDER BY, the first offer is the best one if it's in stock.
	if len(offers) > 0 && offers[0].InStock {
		offers[0].IsBest = true
	}
	return offers, nil
}
