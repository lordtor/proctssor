# Microservice Template for BPMN Workflow Platform

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service .

# Final stage
FROM alpine:3.19

# Install ca-certificates
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -u 1000 bpmn

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/service .

# Change ownership
RUN chown -R bpmn:bpmn /app

# Switch to non-root user
USER bpmn

# Expose port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# Run the application
CMD ["./service"]
