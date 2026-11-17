package catalog

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"eneba/backend/internal/httpx"
)

// Handler turns HTTP requests into Store calls and Store results into JSON.
// It contains no SQL and no business rules, only parsing and responding.
type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler { return &Handler{store: store} }

// Routes mounts the catalog endpoints on r.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/products", h.listProducts)
	r.Get("/products/{slug}", h.product)
	r.Get("/products/{slug}/offers", h.offers)
	r.Get("/filters", h.filters)
	r.Get("/search/suggest", h.suggest)
}

// GET /api/products?q=&platform=&region=&min_price=&max_price=&sort=&page=&per_page=
func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	params, err := ParseListParams(r.URL.Query())
	if err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	res, err := h.store.ListProducts(r.Context(), params)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// GET /api/filters
func (h *Handler) filters(w http.ResponseWriter, r *http.Request) {
	f, err := h.store.Filters(r.Context())
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, f)
}

// GET /api/products/{slug}
func (h *Handler) product(w http.ResponseWriter, r *http.Request) {
	p, err := h.store.ProductBySlug(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, "Product not found")
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

// GET /api/products/{slug}/offers
func (h *Handler) offers(w http.ResponseWriter, r *http.Request) {
	offers, err := h.store.OffersBySlug(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, "Product not found")
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": offers})
}

// GET /api/search/suggest?q=eld&limit=6
// Fewer than 2 characters returns an empty list: one letter matches almost
// everything and isn't worth a database query on every keystroke.
func (h *Handler) suggest(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) > maxQueryLen {
		httpx.BadRequest(w, "q must be at most 100 characters")
		return
	}
	limit := 6
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 10 {
			httpx.BadRequest(w, "limit must be a whole number between 1 and 10")
			return
		}
		limit = n
	}
	if len([]rune(q)) < 2 {
		httpx.JSON(w, http.StatusOK, map[string]any{"items": []Product{}})
		return
	}
	items, err := h.store.Suggest(r.Context(), q, limit)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}
