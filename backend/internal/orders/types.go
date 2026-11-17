package orders

import "time"

// CartItem is one line of a cart, as the browser sends it: which seller's
// offer and how many. Prices are never taken from the browser; the server
// looks them up every time.
type CartItem struct {
	OfferID int64 `json:"offer_id"`
	Qty     int   `json:"qty"`
}

type cartRequest struct {
	Items []CartItem `json:"items"`
}

// ProductRef is the little bit of product info a cart or order line shows.
type ProductRef struct {
	Slug     string  `json:"slug"`
	Name     string  `json:"name"`
	Title    string  `json:"title"`
	ImageURL *string `json:"image_url"`
	Platform string  `json:"platform"`
	Region   string  `json:"region"`
}

// QuoteLine is one cart line priced by the server (POST /api/cart/quote).
type QuoteLine struct {
	OfferID        int64       `json:"offer_id"`
	Qty            int         `json:"qty"`
	Status         string      `json:"status"`  // ok | not_enough_stock | sold_out | unavailable
	Product        *ProductRef `json:"product"` // null when the offer no longer exists
	SellerName     string      `json:"seller_name"`
	PriceCents     int         `json:"price_cents"`
	Stock          int         `json:"stock"`
	LineTotalCents int         `json:"line_total_cents"`
}

type Quote struct {
	Items        []QuoteLine `json:"items"`
	TotalCents   int         `json:"total_cents"`   // only lines with status "ok"
	AllAvailable bool        `json:"all_available"` // true when every line can be bought
}

// Order is the body of GET /api/orders/{id} (and of create/pay responses).
type Order struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"` // pending | paid | failed | expired
	TotalCents int       `json:"total_cents"`
	Currency   string    `json:"currency"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"` // when an unpaid order's reservation runs out
	// SecondsLeft is the time left on the reservation, measured on the
	// database's clock. The checkout countdown uses this instead of comparing
	// expires_at with the browser's clock, because the two clocks can
	// disagree (e.g. a Docker VM whose clock drifted after sleep).
	SecondsLeft int         `json:"seconds_left"`
	PaidAt      *time.Time  `json:"paid_at"`
	PaymentRef  *string     `json:"payment_ref"`
	Items       []OrderItem `json:"items"`
}

type OrderItem struct {
	ID             int64      `json:"id"`
	OfferID        int64      `json:"offer_id"`
	Product        ProductRef `json:"product"`
	SellerName     string     `json:"seller_name"`
	UnitPriceCents int        `json:"unit_price_cents"` // copied at purchase time
	Quantity       int        `json:"quantity"`
	Keys           []string   `json:"keys"` // only filled once the order is paid
}
