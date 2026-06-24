FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o backend ./cmd/api

FROM alpine:latest
RUN apk add --no-cache openssl ca-certificates curl
WORKDIR /app
COPY --from=builder /app/backend .
COPY docker-entrypoint.sh .
COPY web/dashboard ./web/dashboard
RUN chmod +x docker-entrypoint.sh
EXPOSE 8080
ENTRYPOINT ["./docker-entrypoint.sh"]
