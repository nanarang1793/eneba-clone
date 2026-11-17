package home

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"eneba/backend/internal/catalog"
)

// Provider produces the data for one kind of homepage block.
//
// The assignment asks for each block to get its content "from its own backend
// service or database". Here each block type is its own provider behind this
// interface, with its own query, and the frontend fetches every block
// separately. Moving one into a separate service later would only mean
// replacing its provider with an HTTP call; the frontend wouldn't change.
type Provider interface {
	// Load returns the block's data. config is the block's JSON settings
	// from home_blocks.config, e.g. {"limit": 12}.
	Load(ctx context.Context, config json.RawMessage) (any, error)
}

// providers maps home_blocks.type to the code that fills it.
func providers(db *pgxpool.Pool, cat *catalog.Store) map[string]Provider {
	return map[string]Provider{
		"banner":       bannerProvider{db: db},
		"best_sellers": bestSellers{cat: cat},
		"deals":        deals{cat: cat},
		"new_releases": newReleases{cat: cat},
		"by_type":      byType{cat: cat},
	}
}

// ProductRow is the data of every product block: cards plus where
// "See all" should go.
type ProductRow struct {
	Items     []catalog.Product `json:"items"`
	SeeAllURL string            `json:"see_all_url"`
}

// blockConfig holds every setting any block uses; each provider reads the
// fields it needs.
type blockConfig struct {
	Limit          int    `json:"limit"`
	MinDiscountPct int    `json:"min_discount_pct"`
	Type           string `json:"type"`
}

func parseConfig(raw json.RawMessage) (blockConfig, error) {
	c := blockConfig{Limit: 12}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &c); err != nil {
			return c, fmt.Errorf("block config: %w", err)
		}
	}
	if c.Limit < 1 || c.Limit > 24 {
		c.Limit = 12
	}
	return c, nil
}

// onePerTitle keeps one card per game: the variant that ranks first by
// `rank`. Without it, "Bestsellers" could be three Elden Ring cards in a row
// (PC, PS5, Xbox).
func onePerTitle(rank string) string {
	return `l.id IN (SELECT DISTINCT ON (title) id FROM listing ORDER BY title, ` + rank + `)`
}

// ---- Bestsellers: most sold first (sales_count also goes up with real orders).
type bestSellers struct{ cat *catalog.Store }

func (p bestSellers) Load(ctx context.Context, raw json.RawMessage) (any, error) {
	c, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	items, err := p.cat.Cards(ctx, onePerTitle("sales_count DESC"), "l.sales_count DESC", c.Limit)
	return ProductRow{Items: items, SeeAllURL: "/store?sort=popular"}, err
}

// ---- Hot deals: biggest discount vs. retail price first.
type deals struct{ cat *catalog.Store }

func (p deals) Load(ctx context.Context, raw json.RawMessage) (any, error) {
	c, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	const discount = `(l.msrp_cents - l.price_cents)::float / l.msrp_cents`
	items, err := p.cat.Cards(ctx,
		onePerTitle("(msrp_cents - price_cents)::float / msrp_cents DESC NULLS LAST")+
			` AND l.msrp_cents > 0 AND `+discount+` * 100 >= $1`,
		discount+" DESC", c.Limit, c.MinDiscountPct)
	return ProductRow{Items: items, SeeAllURL: "/store?sort=price_asc"}, err
}

// ---- New releases: newest release date first.
type newReleases struct{ cat *catalog.Store }

func (p newReleases) Load(ctx context.Context, raw json.RawMessage) (any, error) {
	c, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	items, err := p.cat.Cards(ctx, onePerTitle("release_date DESC NULLS LAST"), "l.release_date DESC NULLS LAST", c.Limit)
	return ProductRow{Items: items, SeeAllURL: "/store?sort=newest"}, err
}

// ---- One product type (gift cards, subscriptions…), most popular first.
type byType struct{ cat *catalog.Store }

func (p byType) Load(ctx context.Context, raw json.RawMessage) (any, error) {
	c, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	if c.Type == "" {
		return nil, fmt.Errorf("by_type block needs a \"type\" in its config")
	}
	items, err := p.cat.Cards(ctx, "l.type = $1", "l.sales_count DESC", c.Limit, c.Type)
	return ProductRow{Items: items, SeeAllURL: "/store?type=" + c.Type}, err
}

// ---- Banner carousel: its own table, not products at all.
type bannerProvider struct{ db *pgxpool.Pool }

type Banner struct {
	ID       int     `json:"id"`
	Title    string  `json:"title"`
	Subtitle *string `json:"subtitle"`
	ImageURL *string `json:"image_url"`
	CTALabel string  `json:"cta_label"`
	CTAURL   string  `json:"cta_url"`
}

func (p bannerProvider) Load(ctx context.Context, _ json.RawMessage) (any, error) {
	// Only banners whose optional start/end window includes "now".
	rows, err := p.db.Query(ctx, `
        SELECT id, title, subtitle, image_url, cta_label, cta_url
        FROM banners
        WHERE (starts_at IS NULL OR starts_at <= now())
          AND (ends_at IS NULL OR ends_at > now())
        ORDER BY position`)
	if err != nil {
		return nil, fmt.Errorf("banners: %w", err)
	}
	defer rows.Close()
	items := []Banner{}
	for rows.Next() {
		var b Banner
		if err := rows.Scan(&b.ID, &b.Title, &b.Subtitle, &b.ImageURL, &b.CTALabel, &b.CTAURL); err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return map[string]any{"items": items}, rows.Err()
}
