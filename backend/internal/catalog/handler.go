package catalog

import (
	"net/http"

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
	r.Get("/filters", h.filters)
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
