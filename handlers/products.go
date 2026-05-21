package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"source-asia-backend/products"
)

func HandleCreateProduct(s *products.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name      string   `json:"name"`
			SKU       string   `json:"sku"`
			ImageURLs []string `json:"image_urls"`
			VideoURLs []string `json:"video_urls"`
		}

		if err := decodeJSON(r, &body); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if strings.TrimSpace(body.Name) == "" {
			WriteError(w, http.StatusBadRequest, "name is required and must be non-empty")
			return
		}
		if strings.TrimSpace(body.SKU) == "" {
			WriteError(w, http.StatusBadRequest, "sku is required and must be non-empty")
			return
		}

		// Validate URLs before hitting the store to fail fast with a clear message.
		if err := products.ValidateURLs(body.ImageURLs); err != nil {
			WriteError(w, http.StatusBadRequest, "image_urls: "+err.Error())
			return
		}
		if err := products.ValidateURLs(body.VideoURLs); err != nil {
			WriteError(w, http.StatusBadRequest, "video_urls: "+err.Error())
			return
		}

		detail, err := s.Create(body.Name, body.SKU, body.ImageURLs, body.VideoURLs)
		if err != nil {
			// The only error Create returns is a duplicate SKU.
			WriteError(w, http.StatusConflict, "a product with this SKU already exists")
			return
		}

		// 201 Created — a new resource was created.
		WriteJSON(w, http.StatusCreated, detail)
	}
}

// HandleListProducts handles GET /products.
//
// Returns a paginated list of products WITHOUT their full image_urls /
// video_urls arrays. The response includes image_count, video_count, and
// an optional thumbnail_url (first image) so a UI grid can render
// without loading every URL for every product.
//
// Query params:
//   - limit  (default 20, max 100)
//   - offset (default 0)
func HandleListProducts(s *products.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit := 20 // sensible default page size
		offset := 0

		if l := q.Get("limit"); l != "" {
			v, err := strconv.Atoi(l)
			if err != nil || v < 1 {
				WriteError(w, http.StatusBadRequest, "limit must be a positive integer")
				return
			}
			// Cap at 100 to prevent accidentally huge responses.
			if v > 100 {
				v = 100
			}
			limit = v
		}
		if o := q.Get("offset"); o != "" {
			v, err := strconv.Atoi(o)
			if err != nil || v < 0 {
				WriteError(w, http.StatusBadRequest, "offset must be a non-negative integer")
				return
			}
			offset = v
		}

		items, total := s.List(offset, limit)
		WriteJSON(w, http.StatusOK, map[string]any{
			"total":    total, // total products in the store (for pagination UI)
			"offset":   offset,
			"limit":    limit,
			"products": items,
		})
	}
}

// HandleGetProduct handles GET /products/{id}.
//
// Returns the full product detail including all image_urls and video_urls.
// This is intentionally the only endpoint that loads the full media arrays.
func HandleGetProduct(s *products.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := extractID(r.URL.Path)

		detail, ok := s.GetByID(id)
		if !ok {
			WriteError(w, http.StatusNotFound, "product not found: "+id)
			return
		}
		WriteJSON(w, http.StatusOK, detail)
	}
}

// HandleAddMedia handles POST /products/{id}/media.
//
// Appends new image and/or video URLs to an existing product.
// At least one of image_urls or video_urls must be non-empty.
// All URL validation rules (scheme, length, count) still apply.
func HandleAddMedia(s *products.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the product id from a path like /products/prod_1/media
		path := strings.TrimPrefix(r.URL.Path, "/products/")
		id := strings.TrimSuffix(path, "/media")

		// Check the product exists before parsing the body.
		if _, ok := s.GetByID(id); !ok {
			WriteError(w, http.StatusNotFound, "product not found: "+id)
			return
		}

		var body struct {
			ImageURLs []string `json:"image_urls"`
			VideoURLs []string `json:"video_urls"`
		}
		if err := decodeJSON(r, &body); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		// Require at least one URL — an empty body is not useful.
		if len(body.ImageURLs) == 0 && len(body.VideoURLs) == 0 {
			WriteError(w, http.StatusBadRequest, "at least one of image_urls or video_urls must be provided")
			return
		}

		if err := products.ValidateURLs(body.ImageURLs); err != nil {
			WriteError(w, http.StatusBadRequest, "image_urls: "+err.Error())
			return
		}
		if err := products.ValidateURLs(body.VideoURLs); err != nil {
			WriteError(w, http.StatusBadRequest, "video_urls: "+err.Error())
			return
		}

		s.AddMedia(id, body.ImageURLs, body.VideoURLs)

		// Return the updated full product so the caller can confirm
		// what was added without needing a separate GET.
		detail, _ := s.GetByID(id)
		WriteJSON(w, http.StatusOK, detail)
	}
}

// extractID pulls the product id segment from paths like /products/prod_1
// or /products/prod_1/media.
func extractID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
