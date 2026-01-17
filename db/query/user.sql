-- name: CreateUser :one
INSERT INTO users (
    id, 
    name, 
    username,
    email, 
    password,
    phone_number, 
    "address", 
    role
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;

-- name: GetAllUsers :many
SELECT id, "name", username, email, "password",phone_number, "address", "role", created_at, updated_at
FROM users
WHERE deleted_at IS NULL;

-- name: GetUserByUsername :one
SELECT id, "name", username, email, "password",phone_number, "address", "role", created_at, updated_at
FROM users
WHERE username = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT id, "name", username, email, "password",phone_number, "address", "role", created_at, updated_at
FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByIDs :many
SELECT id, "name", username, email, "password",phone_number, "address", "role", created_at, updated_at
FROM users
WHERE id = ANY($1::uuid[]) AND deleted_at IS NULL;

-- name: ExistUsernameorEmail :one
SELECT username, email
FROM users
WHERE (username = $1 OR email = $2) AND deleted_at IS NULL
LIMIT 1;

-- name: UpdateUser :one
UPDATE users
SET
    "name" = $2,
    username = $3,
    email = $4,
    "password" = $5,
    "role" = $6,
    phone_number = $7,
    "address" = $8,
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL RETURNING *;

-- name: DeleteUser :one
UPDATE users
SET deleted_at = now()
WHERE id = $1 AND deleted_at IS NULL RETURNING *;
