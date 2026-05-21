# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files first (better Docker layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# Build the binary
RUN go build -o server .

# Run stage — use minimal image for smaller container
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy swagger docs so they're available at runtime
COPY --from=builder /app/docs ./docs

EXPOSE 8080

CMD ["./server"]