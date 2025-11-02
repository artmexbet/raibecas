# Build stage
FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum* ./
COPY ../../libs/proto /libs/proto

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/auth

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Expose port
EXPOSE 8081

# Run the application
CMD ["./main"]
