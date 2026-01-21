# Build Stage
FROM golang:1.25-alpine AS builder

# Install git for fetching dependencies
RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 creates a statically linked binary (no libc dependency)
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

# Run Stage
FROM alpine:latest

WORKDIR /root/

# Install ca-certificates for DB SSL connection
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/server .

# Copy docs folder for Swagger to work (if embedded, this might be optional but safest to keep structure)
COPY --from=builder /app/docs ./docs

# Expose port
EXPOSE 8080

# Command to run
CMD ["./server"]
