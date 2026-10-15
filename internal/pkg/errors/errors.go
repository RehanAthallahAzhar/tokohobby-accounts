package errors

import "errors"

var (
	// Client
	ErrInvalidRequestPayload = errors.New("invalid request payload")
	ErrInvalidQuery          = errors.New("invalid query parameters")
	ErrClientCanceled        = errors.New("client request canceled")

	// Server
	ErrRequestTimeout      = errors.New("request timeout")
	ErrServiceUnavailable  = errors.New("service unavailable")
	ErrInternalServerError = errors.New("internal server error")

	// Auth
	ErrInvalidUserSession    = errors.New("invalid user session")
	ErrInvalidToken          = errors.New("invalid token")
	ErrExpiredToken          = errors.New("expired token")
	ErrInvalidTokenRole      = errors.New("expired token role")
	ErrFailedToGenerateToken = errors.New("failed to generate token")
	ErrInvalidTokenFormat    = errors.New("invalid token format")
	ErrMissingJTI            = errors.New("missing JTI, cannot revoke token")
	ErrFailedToRevokeToken   = errors.New("failed to revoke token")
	ErrTokenNotFound         = errors.New("token not found")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrAccountDisabled       = errors.New("account disabled")
	ErrSessionError          = errors.New("session error")

	// Domain
	ErrUserAlreadyExists     = errors.New("user already exists")
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrNotFound              = errors.New("not found")
	ErrForbidden             = errors.New("forbidden")
	ErrUserNotFound          = errors.New("user not found")
	ErrResourceNotFound      = errors.New("resource not found")
	ErrAlreadyExists         = errors.New("already exists")

	// Context
	ErrFailedToGetIDFromContext                  = errors.New("failed to get ID from context")
	ErrFailedToGetUserIDFromContext              = errors.New("failed to get UserID from context")
	ErrFailedToGetRoleFromContext                = errors.New("failed to get role from context")
	ErrFailedToGetUsernameFromContext            = errors.New("failed to get username from context")
	ErrFailedToGetAuthorizationHeaderFromContext = errors.New("failed to get authorization header from context")

	ErrUnprocessableEntity   = errors.New("unprocessable entity")
	ErrDatabaseConnection    = errors.New("database connection error")
	ErrRedisConnection       = errors.New("redis connection error")
	ErrThirdPartyUnavailable = errors.New("third party unavailable")
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
}

func (e AppError) Error() string {
	return e.Message
}

func (e ValidationErrors) Error() string {
	return "validation failed"
}

/*

### 📌 Error Handling Best Practice per Layer

| Layer       	   | Gunakan `errors.New(...)`                            		| Gunakan `fmt.Errorf(...)`                            |
|-----------------|-------------------------------------------------------------|------------------------------------------------------|
| **Repo**        | Untuk error domain yang dikenal (contoh: `ErrUserNotFound`) | Untuk memberikan detail/debug error DB (ex: GORM)     |
| **Service**     | Untuk validasi input/domain (contoh: `ErrInvalidInput`)     | Untuk *wrap* error dari repository dengan konteks     |
| **Controller**  | Untuk membentuk respon HTTP user-friendly                | ❌ Tidak digunakan (cukup gunakan error dari service) |



*/
