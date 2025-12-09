# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o index ./cmd/index

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/index .

# Create data directory
RUN mkdir -p /data/documents

# Expose port
EXPOSE 8082

# Run the application
CMD ["./index"]

