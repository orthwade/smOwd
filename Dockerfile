# Build stage
FROM golang:1.23.3 AS builder

WORKDIR /app

# Copy go.mod and go.sum, then download dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy source code
COPY . .

# Build the Go binary
RUN go build -o main .

# Runtime stage
FROM golang:1.23.3

WORKDIR /app

# Copy the built binary from the builder image
COPY --from=builder /app/main /app/main

# Install PostgreSQL client libraries
RUN apt-get update && apt-get install -y libpq-dev

# Expose the required port
EXPOSE 8080

# Run the Go application
CMD ["/app/main"]
