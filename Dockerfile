# Build stage
FROM golang:1.26-alpine AS builder

RUN apk update && apk add --no-cache build-base

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -a -ldflags="-w -s" -o /app/server ./cmd/server

# Final 
FROM alpine:latest

RUN apk update && apk add --no-cache ffmpeg sqlite

WORKDIR /app

COPY --from=builder /app/server ./

COPY ./web ./web

EXPOSE 8080

CMD ["./server"]
