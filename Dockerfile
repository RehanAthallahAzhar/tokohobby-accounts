# Build
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Copy local dependencies
COPY messaging /app/messaging
COPY protos /app/protos

# Copy accounts service
COPY accounts /app/accounts
WORKDIR /app/accounts

# Download dependencies
RUN go mod download

# Build aplikasi
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/web/main.go


# Final Image
FROM alpine:latest

WORKDIR /app

# Copy binary 
COPY --from=builder /app/server .

# Copy folder migrasi
COPY --from=builder /app/accounts/db/migrations ./db/migrations

# Expose port 
EXPOSE 8080

# Run aplikasi
CMD ["./server"]