# Build stage
FROM docker.io/library/golang:1.20-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/deepempower ./cmd/server

# Final stage
FROM docker.io/library/alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/bin/deepempower /usr/local/bin/deepempower

# Copy configuration files
COPY --from=builder /app/configs /etc/deepempower/configs

# Set working directory
WORKDIR /usr/local/bin

# Expose port
EXPOSE 8080

# Set environment variables
ENV CONFIG_PATH=/etc/deepempower/configs

# Run the application
CMD ["deepempower"]