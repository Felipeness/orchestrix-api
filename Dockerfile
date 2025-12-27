# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/worker ./cmd/worker

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

# Copy binaries
COPY --from=builder /bin/api /api
COPY --from=builder /bin/worker /worker

# Use non-root user
USER nonroot:nonroot

# Expose port
EXPOSE 8080

# Default to API
ENTRYPOINT ["/api"]
