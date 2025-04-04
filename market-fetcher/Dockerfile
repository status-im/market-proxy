# Build stage
FROM golang:1.20-alpine AS builder

WORKDIR /app
COPY . .

RUN apk add --no-cache gcc musl-dev && \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /market-fetcher

# Runtime stage
FROM alpine:3.18

WORKDIR /app
COPY --from=builder /market-fetcher /usr/local/bin/market-fetcher
COPY config.yaml .

CMD ["/usr/local/bin/market-fetcher"] 