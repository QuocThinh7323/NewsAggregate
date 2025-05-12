# Stage 1: Build
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o article-service .

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/article-service .

CMD ["./article-service"]
