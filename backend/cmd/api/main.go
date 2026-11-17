// Command api is the HTTP server for the marketplace.
//
// main only wires things together: read config, open the database pool,
// build the router, start listening. The real work lives in internal/.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"eneba/backend/internal/auth"
	"eneba/backend/internal/catalog"
	"eneba/backend/internal/httpx"
)

func main() {
	port := getenv("PORT", "8080")
	dbURL := getenv("DATABASE_URL", "postgres://eneba:eneba@localhost:5432/eneba?sslmode=disable")

	// A pool keeps a few open connections and hands them out per query,
	// so requests don't pay the cost of connecting every time.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("database config: %v", err)
	}
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("cannot reach database (is Postgres running? try `docker compose up -d db`): %v", err)
	}
	defer db.Close()

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           newRouter(db),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("api listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}

func newRouter(db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	// Middleware runs on every request, in this order.
	r.Use(middleware.RequestID)                 // tag each request with an id (shows in logs)
	r.Use(middleware.RealIP)                    // trust X-Forwarded-For from Render/Vercel
	r.Use(middleware.Logger)                    // one log line per request
	r.Use(httpx.Recoverer)                      // a panic becomes a JSON 500
	r.Use(middleware.Timeout(10 * time.Second)) // cancel slow requests (and their DB queries)

	r.NotFound(httpx.NotFound)
	r.MethodNotAllowed(httpx.MethodNotAllowed)

	catalogHandler := catalog.NewHandler(catalog.NewStore(db))
	authHandler := auth.NewHandler(auth.NewStore(db))

	r.Route("/api", func(r chi.Router) {
		r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
			if err := db.Ping(r.Context()); err != nil {
				httpx.Error(w, http.StatusServiceUnavailable, "db_unavailable", "Database is not reachable")
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
		catalogHandler.Routes(r)
		authHandler.Routes(r)
	})
	return r
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
