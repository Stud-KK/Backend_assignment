package products

import (
	"fmt"
	"net/url"
	"sync"
	"time"
)

const (
	MaxURLsPerRequest = 20
	MaxURLLength      = 2048
)

type Product struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SKU       string    `json:"sku"`
	CreatedAt time.Time `json:"created_at"`
}

type ProductMedia struct {
	ImageURLs []string
	VideoURLs []string
}

type ListItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	SKU          string    `json:"sku"`
	ImageCount   int       `json:"image_count"`
	VideoCount   int       `json:"video_count"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"` // first image URL if present
	CreatedAt    time.Time `json:"created_at"`
}

type Detail struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SKU       string    `json:"sku"`
	ImageURLs []string  `json:"image_urls"`
	VideoURLs []string  `json:"video_urls"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	mu sync.RWMutex

	products map[string]*Product      // id → core product fields
	media    map[string]*ProductMedia // id → media URLs (kept separate for performance)
	skuIndex map[string]string        // sku → id, used for O(1) duplicate SKU checks
	order    []string                 // insertion order, used for stable pagination
	nextID   int                      // auto-increment counter for generating IDs
}

// NewStore returns an initialised, empty product store.
func NewStore() *Store {
	return &Store{
		products: make(map[string]*Product),
		media:    make(map[string]*ProductMedia),
		skuIndex: make(map[string]string),
	}
}

func (s *Store) Create(name, sku string, images, videos []string) (*Detail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.skuIndex[sku]; exists {
		return nil, fmt.Errorf("duplicate sku")
	}

	s.nextID++
	id := fmt.Sprintf("prod_%d", s.nextID)

	p := &Product{
		ID:        id,
		Name:      name,
		SKU:       sku,
		CreatedAt: time.Now().UTC(),
	}

	if images == nil {
		images = []string{}
	}
	if videos == nil {
		videos = []string{}
	}

	s.products[id] = p
	s.media[id] = &ProductMedia{ImageURLs: images, VideoURLs: videos}
	s.skuIndex[sku] = id
	s.order = append(s.order, id) // maintain insertion order for pagination

	return &Detail{
		ID:        p.ID,
		Name:      p.Name,
		SKU:       p.SKU,
		ImageURLs: images,
		VideoURLs: videos,
		CreatedAt: p.CreatedAt,
	}, nil
}

// List returns a paginated page of ListItems.
// Crucially, it only reads len(media.ImageURLs) and len(media.VideoURLs)
// — it never copies or serialises the actual URL strings.
func (s *Store) List(offset, limit int) ([]ListItem, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.order)
	result := []ListItem{}

	if offset >= total {
		return result, total
	}

	end := offset + limit
	if end > total {
		end = total
	}

	for _, id := range s.order[offset:end] {
		p := s.products[id]
		m := s.media[id]

		item := ListItem{
			ID:         p.ID,
			Name:       p.Name,
			SKU:        p.SKU,
			ImageCount: len(m.ImageURLs), // O(1) — no URL serialisation
			VideoCount: len(m.VideoURLs),
			CreatedAt:  p.CreatedAt,
		}
		// Use the first image as the thumbnail if one exists.
		if len(m.ImageURLs) > 0 {
			item.ThumbnailURL = m.ImageURLs[0]
		}
		result = append(result, item)
	}
	return result, total
}

func (s *Store) GetByID(id string) (*Detail, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.products[id]
	if !ok {
		return nil, false
	}
	m := s.media[id]

	// Guard against nil slices so JSON always outputs [] not null.
	imgs := m.ImageURLs
	if imgs == nil {
		imgs = []string{}
	}
	vids := m.VideoURLs
	if vids == nil {
		vids = []string{}
	}

	return &Detail{
		ID:        p.ID,
		Name:      p.Name,
		SKU:       p.SKU,
		ImageURLs: imgs,
		VideoURLs: vids,
		CreatedAt: p.CreatedAt,
	}, true
}

func (s *Store) AddMedia(id string, images, videos []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.media[id]
	if !ok {
		return false
	}
	m.ImageURLs = append(m.ImageURLs, images...)
	m.VideoURLs = append(m.VideoURLs, videos...)
	return true
}

func ValidateURLs(urls []string) error {
	if len(urls) > MaxURLsPerRequest {
		return fmt.Errorf("too many URLs: max %d per field", MaxURLsPerRequest)
	}
	for _, u := range urls {
		if len(u) > MaxURLLength {
			return fmt.Errorf("URL exceeds max length of %d chars", MaxURLLength)
		}
		parsed, err := url.ParseRequestURI(u)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("invalid URL (must start with http:// or https://): %s", u)
		}
	}
	return nil
}
