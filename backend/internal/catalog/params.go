package catalog

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
)

// Sort options accepted by GET /api/products?sort=…
const (
	SortRelevance = "relevance" // best search match first; without q it falls back to popular
	SortPriceAsc  = "price_asc"
	SortPriceDesc = "price_desc"
	SortNewest    = "newest"
	SortPopular   = "popular"
)

var validSorts = map[string]bool{
	SortRelevance: true, SortPriceAsc: true, SortPriceDesc: true, SortNewest: true, SortPopular: true,
}

const (
	defaultPerPage = 24
	maxPerPage     = 60
	maxQueryLen    = 100
)

// ListParams is the validated form of the query string.
type ListParams struct {
	Q         string   // search text, already trimmed; "" means no search
	Platforms []string // platform slugs, e.g. ["steam", "xbox"]; empty means all
	Regions   []string // region slugs
	MinCents  *int     // nil means no lower bound
	MaxCents  *int
	Sort      string
	Page      int // 1-based
	PerPage   int
}

// ParseListParams reads and validates the query string. It returns an error
// whose text is safe to show to the client (it becomes the 400 message).
func ParseListParams(v url.Values) (ListParams, error) {
	p := ListParams{
		Q:         strings.TrimSpace(v.Get("q")),
		Platforms: slugList(v["platform"]),
		Regions:   slugList(v["region"]),
		Sort:      v.Get("sort"),
		Page:      1,
		PerPage:   defaultPerPage,
	}

	if len([]rune(p.Q)) > maxQueryLen {
		return p, fmt.Errorf("q must be at most %d characters", maxQueryLen)
	}

	if p.Sort == "" {
		p.Sort = SortRelevance
	}
	if !validSorts[p.Sort] {
		return p, fmt.Errorf("sort must be one of relevance, price_asc, price_desc, newest, popular")
	}

	var err error
	if p.Page, err = intParam(v, "page", 1, 1, 1000); err != nil {
		return p, err
	}
	if p.PerPage, err = intParam(v, "per_page", defaultPerPage, 1, maxPerPage); err != nil {
		return p, err
	}

	if p.MinCents, err = priceParam(v, "min_price"); err != nil {
		return p, err
	}
	if p.MaxCents, err = priceParam(v, "max_price"); err != nil {
		return p, err
	}
	if p.MinCents != nil && p.MaxCents != nil && *p.MinCents > *p.MaxCents {
		return p, fmt.Errorf("min_price cannot be greater than max_price")
	}
	return p, nil
}

// slugList accepts both ?platform=steam&platform=xbox and ?platform=steam,xbox.
func slugList(values []string) []string {
	var out []string
	for _, v := range values {
		for _, s := range strings.Split(v, ",") {
			if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func intParam(v url.Values, name string, def, min, max int) (int, error) {
	raw := v.Get(name)
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("%s must be a whole number between %d and %d", name, min, max)
	}
	return n, nil
}

// priceParam reads a price in euros ("12.5") and returns it in cents (1250).
// Money is integer cents everywhere in the backend.
func priceParam(v url.Values, name string) (*int, error) {
	raw := v.Get(name)
	if raw == "" {
		return nil, nil
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil || f < 0 || f > 100000 || math.IsNaN(f) {
		return nil, fmt.Errorf("%s must be a price in euros between 0 and 100000", name)
	}
	cents := int(math.Round(f * 100))
	return &cents, nil
}
