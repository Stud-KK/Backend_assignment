# Source Asia – Backend Assignment

A REST API service built in Go implementing a rate-limited request endpoint and a product catalog with media management.

> **AI Usage:** I used Claude (Anthropic) to assist with code structure, Go syntax, and this README, as I am new to Go. I understand all design decisions made and can explain them.

---

## Project Structure

```
source-asia-backend/
├── main.go               # Entry point — wires routes to handlers
├── go.mod
├── ratelimiter/
│   └── ratelimiter.go    # Fixed-window rate limiter logic
├── products/
│   └── store.go          # In-memory product store + validation
└── handlers/
    ├── helpers.go         # Shared JSON response helpers
    ├── request.go         # POST /request and GET /stats handlers
    └── products.go        # All product endpoint handlers
```

---

## How to Run

**Requirements:** Go 1.21 or later

```bash
cd source-asia-backend
go run main.go
```

Server starts on **http://localhost:8080**

**Live API:** https://backendassignment-production-452e.up.railway.app

**Live Swagger Docs:** https://backendassignment-production-452e.up.railway.app/docs

## Endpoints Overview

| Method | Path | Description |
|---|---|---|
| POST | `/request` | Submit a rate-limited request |
| GET | `/stats` | Per-user rate limit statistics |
| POST | `/products` | Create a product |
| GET | `/products` | List products (paginated) |
| GET | `/products/:id` | Get full product detail |
| POST | `/products/:id/media` | Append media URLs to a product |

---

## Design Decisions

### Part 1 – Rate Limiter

**Algorithm: Fixed 1-minute window**

Each user gets a fresh window every minute. Once 5 accepted requests are reached, all further requests in that window return `429 Too Many Requests`.

- Fixed window was chosen over sliding window for simplicity and predictability
- The check-and-increment is done inside a `sync.Mutex` — parallel requests for the same `user_id` are serialised so the count never exceeds 5 due to a race condition
- `rejected_total` in `/stats` is cumulative across all windows since server start

**Response code:** `201 Created` on success — because each accepted call creates a new request record.

---

### Part 2 – Product Catalog

**Why media is stored separately from core product fields:**

```
products map  →  id: { id, name, sku, created_at }     ← small, always loaded
media map     →  id: { image_urls[], video_urls[] }     ← large, only loaded on detail
```

`GET /products` (list) never touches the media map — it only calls `len()` on the slices for counts. With 1,000 products and 10 images each, the list endpoint never serialises any of those 10,000 URLs.

`GET /products/:id` (detail) is the only endpoint that loads full media arrays.

**Duplicate SKU returns `409 Conflict` not `400`** — because the request body is valid, the conflict is a state problem not a client mistake.

---

## Data Model

### In Memory

```
products map  →  id: { id, name, sku, created_at }
media map     →  id: { image_urls[], video_urls[] }
skuIndex map  →  sku: id                             (O(1) duplicate check)
order slice   →  [id1, id2, ...]                     (stable pagination)
```

### With PostgreSQL in Production

```sql
CREATE TABLE products (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name       TEXT NOT NULL,
  sku        TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE product_media (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  product_id UUID REFERENCES products(id) ON DELETE CASCADE,
  media_type TEXT CHECK (media_type IN ('image', 'video')),
  url        TEXT NOT NULL,
  position   INT  NOT NULL DEFAULT 0
);

CREATE INDEX ON product_media(product_id);
```

List query would join only `products` with COUNT subqueries — never fetching full media rows. Detail query fetches media for one product only.

---

## Production Limitations

### Part 1 – Rate Limiter

| Limitation | Fix in Production |
|---|---|
| Single instance only — state is in memory | Use Redis with atomic `INCR` + `EXPIRE` |
| Restart loses all state | Persist counters to Redis or a database |
| Fixed window edge case — 10 requests possible in 2 seconds at window boundary | Use sliding window via Redis sorted sets |
| No IP-based limiting | Add IP-based limiting as a second layer |

### Part 2 – Product Catalog

| Limitation | Fix in Production |
|---|---|
| No persistence — data lost on restart | Use PostgreSQL |
| No authentication | Add API key or JWT middleware |
| Sequential IDs are predictable | Use UUIDs |
| No search or filtering | Add full-text search via PostgreSQL or Elasticsearch |
| No soft delete | Add `deleted_at` field |
