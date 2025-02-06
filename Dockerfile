FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/admin-api ./admin-api/cmd/api

FROM alpine:3.19

WORKDIR /app

# Install postgresql-client and migrate
RUN apk add --no-cache postgresql-client curl && \
    curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz && \
    mv migrate /usr/local/bin/migrate && \
    chmod +x /usr/local/bin/migrate

# Copy the binary and config
COPY --from=builder /app/bin/admin-api .
COPY --from=builder /app/config/config.yaml ./config/
COPY migrations /migrations

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

CMD ["sh", "-c", "migrate -path=/migrations -database postgres://postgres:postgres@postgres:5432/aiclinic?sslmode=disable up && ./admin-api"] 