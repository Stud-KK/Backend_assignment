package main

import (
	"fmt"
	"net/http"
	"strings"

	httpSwagger "github.com/swaggo/http-swagger"

	"source-asia-backend/handlers"
	"source-asia-backend/products"
	"source-asia-backend/ratelimiter"
)

func main() {
	// Initialise the rate limiter (Part 1) and product store (Part 2).
	// Both are safe for concurrent use — their internal state is
	// protected by mutexes inside each package.
	rl := ratelimiter.New()
	ps := products.NewStore()

	mux := http.NewServeMux()

	// ── Swagger UI ──────────────────────────────────────────────────────
	// Serves interactive API documentation at /docs
	mux.HandleFunc("/docs/", httpSwagger.Handler(
		httpSwagger.URL("/docs/swagger.yaml"),
	))

	// Serve the raw swagger.yaml file so Swagger UI can load it
	mux.Handle("/docs/swagger.yaml", http.FileServer(http.Dir(".")))

	// ── Part 1: Rate-limited requests ──────────────────────────────────
	mux.HandleFunc("/request", handlers.HandleRequest(rl))
	mux.HandleFunc("/stats", handlers.HandleStats(rl))

	// ── Part 2: Product catalog ─────────────────────────────────────────

	// /products handles both POST (create) and GET (list).
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.HandleCreateProduct(ps)(w, r)
		case http.MethodGet:
			handlers.HandleListProducts(ps)(w, r)
		default:
			handlers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	// /products/{id} and /products/{id}/media
	mux.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// POST /products/{id}/media — append media URLs to a product
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/media") {
			handlers.HandleAddMedia(ps)(w, r)
			return
		}

		// GET /products/{id} — full product detail including all URLs
		if r.Method == http.MethodGet {
			handlers.HandleGetProduct(ps)(w, r)
			return
		}

		handlers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	fmt.Println("🚀  Server running on http://localhost:8080")
	fmt.Println()
	fmt.Println("  📖  Swagger Docs  →  http://localhost:8080/docs/")
	fmt.Println()
	fmt.Println("  Part 1 – Rate Limiter")
	fmt.Println("    POST  /request")
	fmt.Println("    GET   /stats")
	fmt.Println()
	fmt.Println("  Part 2 – Product Catalog")
	fmt.Println("    POST  /products")
	fmt.Println("    GET   /products?limit=20&offset=0")
	fmt.Println("    GET   /products/:id")
	fmt.Println("    POST  /products/:id/media")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Println("Server error:", err)
	}
}
