package main

import (
	"fmt"
	"net/http"

	"source-asia-backend/handlers"
	"source-asia-backend/products"
	"source-asia-backend/ratelimiter"
)

func main() {

	rl := ratelimiter.New()
	ps := products.NewStore()

	mux := http.NewServeMux()

	// ── Part 1: Rate-limited requests ──────────────────────────────────
	mux.HandleFunc("/request", handlers.HandleRequest(rl))
	mux.HandleFunc("/stats", handlers.HandleStats(rl))

	// ── Part 2: Product catalog ─────────────────────────────────────────

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

	mux.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if r.Method == http.MethodPost && len(path) > len("/products/") {
			handlers.HandleAddMedia(ps)(w, r)
			return
		}

		if r.Method == http.MethodGet {
			handlers.HandleGetProduct(ps)(w, r)
			return
		}

		handlers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	fmt.Println("Server running on http://localhost:8080")
	fmt.Println()
	fmt.Println("  Part 1 Rate Limiter")
	fmt.Println("    POST  /request")
	fmt.Println("    GET   /stats")
	fmt.Println()
	fmt.Println("  Part 2 Product Catalog")
	fmt.Println("    POST  /products")
	fmt.Println("    GET   /products?limit=20&offset=0")
	fmt.Println("    GET   /products/:id")
	fmt.Println("    POST  /products/:id/media")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Println("Server error:", err)
	}
}
