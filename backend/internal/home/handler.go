// Package home serves the data-driven homepage.
//
//	GET /api/home/layout       -> which blocks to show, in order (from home_blocks)
//	GET /api/home/blocks/{id}  -> one block's data, from that block type's provider
//
// The frontend asks for the layout, then loads every block separately. One
// slow or broken block shows an error card instead of breaking the page, and
// blocks can be reordered, retitled or switched off in the database without
// deploying anything.
package home

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"eneba/backend/internal/catalog"
	"eneba/backend/internal/httpx"
)

type Handler struct {
	db        *pgxpool.Pool
	providers map[string]Provider
}

func NewHandler(db *pgxpool.Pool, cat *catalog.Store) *Handler {
	return &Handler{db: db, providers: providers(db, cat)}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/home/layout", h.layout)
	r.Get("/home/blocks/{id}", h.block)
}

// BlockInfo is one entry of the layout.
type BlockInfo struct {
	ID    int    `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

// GET /api/home/layout -> {"blocks": [{id, type, title}, ...]}
// Only active blocks whose type has a provider are listed.
func (h *Handler) layout(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(),
		`SELECT id, type, title FROM home_blocks WHERE is_active ORDER BY position, id`)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer rows.Close()
	blocks := []BlockInfo{}
	for rows.Next() {
		var b BlockInfo
		if err := rows.Scan(&b.ID, &b.Type, &b.Title); err != nil {
			httpx.Internal(w, r, err)
			return
		}
		if _, known := h.providers[b.Type]; known {
			blocks = append(blocks, b)
		}
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"blocks": blocks})
}

// GET /api/home/blocks/{id} -> {id, type, title, data}
func (h *Handler) block(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, "Block not found")
		return
	}
	info, config, err := h.loadBlock(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, "Block not found")
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	provider, ok := h.providers[info.Type]
	if !ok {
		httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, "Unknown block type")
		return
	}
	data, err := provider.Load(r.Context(), config)
	if err != nil {
		httpx.Internal(w, r, fmt.Errorf("block %d (%s): %w", id, info.Type, err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"id": info.ID, "type": info.Type, "title": info.Title, "data": data})
}

func (h *Handler) loadBlock(ctx context.Context, id int) (BlockInfo, json.RawMessage, error) {
	var b BlockInfo
	var config json.RawMessage
	err := h.db.QueryRow(ctx, `SELECT id, type, title, config FROM home_blocks WHERE id = $1 AND is_active`, id).
		Scan(&b.ID, &b.Type, &b.Title, &config)
	return b, config, err
}
