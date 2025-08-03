# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY . .

RUN go build -ldflags="-s -w" -o app .

# Final image (alpine)
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/app .
COPY .env .

ENTRYPOINT ["./app"]
