# Build stage
FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.work go.work.sum* ./

COPY libs/natsw/go.mod libs/natsw/go.sum* ./libs/natsw/
COPY libs/utils/go.mod libs/utils/go.sum* ./libs/utils/
COPY libs/dto/go.mod libs/dto/go.sum* ./libs/dto/
COPY libs/telemetry/go.mod libs/telemetry/go.sum* ./libs/telemetry/
COPY services/auth/go.mod services/auth/go.sum* ./services/auth/
COPY services/gateway/go.mod services/gateway/go.sum* ./services/gateway/
COPY services/chat/go.mod services/chat/go.sum* ./services/chat/
COPY services/users/go.mod services/users/go.sum* ./services/users/
COPY services/documents/go.mod services/documents/go.sum* ./services/documents/

RUN go work sync && go mod download

COPY libs/ ./libs/
COPY services/gateway/ ./services/gateway/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./services/gateway/cmd/gateway

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]

