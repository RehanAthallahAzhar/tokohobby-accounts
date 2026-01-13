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
- `POST /api/register` - Register new user
- `POST /api/login` - Login
- `POST /api/refresh` - Refresh token
- `POST /api/logout` - Logout
- `GET /api/profile` - Get user profile

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
