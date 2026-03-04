# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o vault-service ./cmd/server

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for TLS
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/vault-service .

# Create data directory for file storage
RUN mkdir -p /data/vault

EXPOSE 8083

CMD ["./vault-service"]
