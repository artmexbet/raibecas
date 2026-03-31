FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Copy go workspace files
COPY go.work go.work.sum ./

# Copy shared libraries
COPY libs/ libs/

# Copy service code
COPY services/documents/ services/documents/

# Copy all go.mod files for workspace modules
COPY libs/natsw/go.mod libs/natsw/go.sum* ./libs/natsw/
COPY libs/utils/go.mod libs/utils/go.sum* ./libs/utils/
COPY libs/dto/go.mod libs/dto/go.sum* ./libs/dto/
COPY libs/telemetry/go.mod libs/telemetry/go.sum* ./libs/telemetry/
COPY services/auth/go.mod services/auth/go.sum* ./services/auth/
COPY services/gateway/go.mod services/gateway/go.sum* ./services/gateway/
COPY services/chat/go.mod services/chat/go.sum* ./services/chat/
COPY services/users/go.mod services/users/go.sum* ./services/users/

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
