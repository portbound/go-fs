# --- Build Stage ---
FROM golang:1.25-alpine AS builder

# Install C build tools needed for CGO
RUN apk update && apk add --no-cache build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -a -ldflags="-w -s" -o /app/server ./cmd/server

FROM alpine:latest

RUN apk update && apk add --no-cache ffmpeg 

# Set the working directory inside the container
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/server .

# Copy the web directory which contains static assets
COPY web ./web

# Expose the port the application will run on
EXPOSE 8080

# Command to run the application
CMD ["./server"]
