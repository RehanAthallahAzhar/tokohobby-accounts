# Accounts Service

User authentication and account management microservice.

## Features

- ✅ User registration & login
- ✅ JWT token authentication
- ✅ Session management with Redis
- ✅ Password hashing (bcrypt)
- ✅ gRPC & REST APIs
- ✅ Database migrations

## API Endpoints

### REST
- `POST /api/v1/register` - Register new user
- `POST /api/v1/login` - Login
- `POST /api/v1/refresh` - Refresh token
- `POST /api/v1/logout` - Logout
- `GET /api/v1/profile` - Get user profile

### gRPC
- `ValidateToken` - Validate JWT token
- `GetUserByID` - Get user details

## Quick Start

```bash
docker compose up -d accounts-db redis-db
docker compose up -d accounts-service
```

## Environment Variables

```bash
ENV=production
LOGRUS_LEVEL=info
DB_HOST=accounts-db
DB_PORT=5432
DB_USER=user
DB_PASSWORD=supersecret123
DB_NAME=accounts
JWT_SECRET=your-secret-key
JWT_AUDIENCE=tokohobby-users
REDIS_HOST=redis-db:6379
```

## Database Schema

- `users` - User accounts
- `refresh_tokens` - Session tokens (Redis)

## Development

```bash
# Run locally
go run cmd/main.go

# Run tests
go test ./...

# Build
go build -o accounts-service cmd/main.go
```
