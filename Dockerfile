# --- Build Stage ---
FROM golang:1.25-alpine AS builder

# Install C build tools needed for CGO
RUN apk update && apk add --no-cache build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application with CGO enabled
# CGO_ENABLED=1 is the default, so we just need to ensure it's not disabled.
RUN go build -a -ldflags="-w -s" -o /app/server ./cmd/server

# --- Final Stage ---
FROM alpine:latest

RUN apk update && apk add --no-cache ffmpeg \
    && which ffmpeg \
    && echo $PATH

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
