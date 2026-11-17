package orders

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"eneba/backend/internal/auth"
	"eneba/backend/internal/httpx"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler { return &Handler{store: store} }

// Routes mounts the cart and order endpoints. Everything except the quote
// sits behind requireUser (401 without a session).
func (h *Handler) Routes(r chi.Router, requireUser func(http.Handler) http.Handler) {
	r.Post("/cart/quote", h.quote)
	r.Group(func(r chi.Router) {
		r.Use(requireUser)
		r.Post("/orders", h.create)
		r.Get("/orders", h.list)
		r.Get("/orders/{id}", h.get)
		r.Post("/orders/{id}/pay", h.pay)
	})
}

// POST /api/cart/quote  {items: [{offer_id, qty}]}
// Public: anyone can price a cart, signed in or not.
func (h *Handler) quote(w http.ResponseWriter, r *http.Request) {
	var req cartRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	items, err := normalizeItems(req.Items)
	if err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	q, err := h.store.Quote(r.Context(), items)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, q)
}

// POST /api/orders  {items}  with header Idempotency-Key: <random id>
// 201 for a new order, 200 when the same key was already used.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("Idempotency-Key")
	if len(key) < 8 || len(key) > 100 {
		httpx.BadRequest(w, "Idempotency-Key header (8–100 characters) is required")
		return
	}
	var req cartRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	items, err := normalizeItems(req.Items)
	if err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	order, created, err := h.store.CreateOrder(r.Context(), auth.CurrentUser(r).ID, key, items)
	var stockErr *OutOfStockError
	switch {
	case errors.As(err, &stockErr):
		httpx.Error(w, http.StatusConflict, "out_of_stock",
			"Some items in your cart are no longer available in that quantity. Please review your cart.")
	case err != nil:
		httpx.Internal(w, r, err)
	case created:
		httpx.JSON(w, http.StatusCreated, order)
	default:
		httpx.JSON(w, http.StatusOK, order)
	}
}

// GET /api/orders: the signed-in user's orders, newest first.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orders, err := h.store.List(r.Context(), auth.CurrentUser(r).ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": orders})
}

// GET /api/orders/{id}: one order; keys included once it's paid.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	order, err := h.store.Get(r.Context(), auth.CurrentUser(r).ID, chi.URLParam(r, "id"))
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, "Order not found")
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}

type payRequest struct {
	CardNumber string `json:"card_number"`
}

// POST /api/orders/{id}/pay  {card_number}
func (h *Handler) pay(w http.ResponseWriter, r *http.Request) {
	var req payRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	order, err := h.store.Pay(r.Context(), auth.CurrentUser(r).ID, chi.URLParam(r, "id"), req.CardNumber)
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, "Order not found")
	case errors.Is(err, ErrInvalidCard):
		httpx.Error(w, http.StatusBadRequest, "invalid_card",
			"This is a demo shop: use test card 4242 4242 4242 4242 (or 4000 0000 0000 0002 to see a decline)")
	case errors.Is(err, ErrDeclined):
		httpx.Error(w, http.StatusPaymentRequired, "payment_declined",
			"Your card was declined. The keys were released; you can check out again from your cart.")
	case errors.Is(err, ErrReservationExpired):
		httpx.Error(w, http.StatusConflict, "reservation_expired",
			"Your 15-minute reservation ran out and the keys were released. Please check out again.")
	case errors.Is(err, ErrNotPending):
		httpx.Error(w, http.StatusConflict, "order_not_pending", "This order can no longer be paid")
	case err != nil:
		httpx.Internal(w, r, err)
	default:
		httpx.JSON(w, http.StatusOK, order)
	}
}
