FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go workspace files
COPY go.work go.work.sum ./

# Copy shared libraries
COPY libs/ libs/

# Copy service code
COPY services/documents/ services/documents/

WORKDIR /app/services/documents

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/documents ./cmd/documents

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/documents /app/documents

# Expose port (if needed for health checks)
EXPOSE 8080

CMD ["/app/documents"]
