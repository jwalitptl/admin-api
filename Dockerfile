# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o admin-api ./cmd/api

# Final stage
FROM alpine:latest
RUN apk add --no-cache netcat-openbsd curl
WORKDIR /app

# Create config directory first
RUN mkdir -p /app/config

# Copy files
COPY --from=builder /app/admin-api .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/config/config.yml /app/config/config.yml

# Make the binary executable
RUN chmod +x /app/admin-api

# Set environment variable to specify config file
ENV CONFIG_FILE=/app/config/config.yml

# Install migrate tool
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-arm64.tar.gz | tar xvz && \
    mv migrate /usr/local/bin/migrate && \
    chmod +x /usr/local/bin/migrate

EXPOSE 8080
CMD ["./admin-api"] 