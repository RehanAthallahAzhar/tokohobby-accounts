# --- Stage 1: Build ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy local dependencies first (messaging and protos)
COPY messaging /app/messaging
COPY protos /app/protos

# Copy accounts service
COPY accounts /app/accounts
WORKDIR /app/accounts

# Download dependencies (will use local messaging & protos via replace directives)
RUN go mod download

# Build aplikasi
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/web/main.go


# --- Stage 2: Final Image ---
FROM alpine:latest

WORKDIR /app

# Copy binary yang sudah di-build dari stage 'builder'
COPY --from=builder /app/server .

# Copy folder migrasi dari stage 'builder' ke stage final
COPY --from=builder /app/accounts/db/migrations ./db/migrations

# Expose port yang digunakan oleh aplikasi Anda di dalam container
EXPOSE 8080

# Command untuk menjalankan aplikasi saat container dimulai
CMD ["./server"]