# Build stage
FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go.work and go.work.sum (for workspace support)
COPY go.work go.work.sum* ./

# Copy all go.mod files for workspace modules
COPY libs/natsw/go.mod libs/natsw/go.sum* ./libs/natsw/
COPY libs/utils/go.mod libs/utils/go.sum* ./libs/utils/
COPY services/auth/go.mod services/auth/go.sum* ./services/auth/
COPY services/gateway/go.mod services/gateway/go.sum* ./services/gateway/
COPY services/chat/go.mod services/chat/go.sum* ./services/chat/

# Download dependencies
RUN go work sync && go mod download

# Copy source code for libs and auth service
COPY libs/ ./libs/
COPY services/auth/ ./services/auth/

# Build the application from workspace
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./services/auth/cmd/auth

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy migrations
COPY --from=builder /app/services/auth/migrations ./migrations

# Expose port
EXPOSE 8081

# Run the application
CMD ["./main"]
