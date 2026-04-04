# Build stage
FROM --platform=$TARGETPLATFORM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /legal-vault ./cmd/msgvault

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /legal-vault /usr/local/bin/legal-vault

# Create directories
RUN mkdir -p /data /keys /config

# Expose ports
EXPOSE 2525 8080

# Run the server
CMD ["legal-vault", "serve-archive"]
