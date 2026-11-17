package catalog

// These structs are the JSON the frontend receives. The TypeScript types in
// frontend/src/api/types.ts mirror them.

type Named struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// Product is one card in the store grid.
type Product struct {
	ID              int64   `json:"id"`
	Slug            string  `json:"slug"`
	Name            string  `json:"name"`  // full variant name: "Elden Ring (PC) Steam Key GLOBAL"
	Title           string  `json:"title"` // base title: "Elden Ring"
	Type            string  `json:"type"`  // game | dlc | gift_card | subscription
	ImageURL        *string `json:"image_url"`
	Platform        Named   `json:"platform"`
	Region          Named   `json:"region"`
	PriceCents      int     `json:"price_cents"` // cheapest in-stock offer
	MSRPCents       *int    `json:"msrp_cents"`
	DiscountPercent int     `json:"discount_percent"`
	OfferCount      int     `json:"offer_count"` // how many sellers have it in stock
}

type FacetCount struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ListResult is the body of GET /api/products.
type ListResult struct {
	Items      []Product `json:"items"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PerPage    int       `json:"per_page"`
	TotalPages int       `json:"total_pages"`
	Facets     struct {
		Platforms []FacetCount `json:"platforms"`
		Regions   []FacetCount `json:"regions"`
	} `json:"facets"`
}

// Filters is the body of GET /api/filters.
type Filters struct {
	Platforms []Named `json:"platforms"`
	Regions   []Named `json:"regions"`
	Price     struct {
		MinCents int `json:"min_cents"`
		MaxCents int `json:"max_cents"`
	} `json:"price"`
	Sorts []string `json:"sorts"`
}
