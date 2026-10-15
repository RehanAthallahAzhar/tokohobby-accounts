package token

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/repositories"
)

type JWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`

	// Claims standar bawaan spesifikasi JWT (Issuer, Expired, JWT ID/jti, dll)
	jwt.RegisteredClaims
}

type jwtTokenService struct {
	jwtSecret        string
	jwtIssuer        string
	jwtAudience      []string
	jwtBlacklistRepo repositories.JWTBlacklistRepository
}

func NewJWTTokenService(jwtSecret, jwtIssuer string, jwtAudience []string, jwtBlacklistRepo repositories.JWTBlacklistRepository) TokenService {
	return &jwtTokenService{
		jwtSecret:        jwtSecret,
		jwtIssuer:        jwtIssuer,
		jwtAudience:      jwtAudience,
		jwtBlacklistRepo: jwtBlacklistRepo,
	}
}

type TokenService interface {
	GenerateAccessToken(ctx context.Context, user *models.GenerateAccessTokenRequest) (string, error)
	GenerateRefreshToken(ctx context.Context) (string, error)
	ValidateToken(ctx context.Context, tokenString string) (isValid bool, userID uuid.UUID, username string, role string, errorMessage string, err error)
	BlacklistToken(ctx context.Context, jti string, expiration time.Duration) error
}

func (s *jwtTokenService) GenerateAccessToken(ctx context.Context, req *models.GenerateAccessTokenRequest) (string, error) {
	claims := &JWTClaims{
		UserID:   req.UserID,
		Username: req.Username,
		Role:     req.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
			Issuer:    s.jwtIssuer,
			Subject:   req.Username,
			Audience:  jwt.ClaimStrings(s.jwtAudience),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}
	return signedToken, nil
}

func (s *jwtTokenService) GenerateRefreshToken(ctx context.Context) (string, error) {
	// Only using UUID, because we’ll validate it directly in Redis later, rather than checking a cryptographic signature.
	return uuid.New().String(), nil
}

func (s *jwtTokenService) ValidateToken(ctx context.Context, tokenString string) (isValid bool, userID uuid.UUID, username string, role string, errorMessage string, err error) {
	token, parseErr := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			slog.WarnContext(ctx, "Unexpected signing method", "alg", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	}, jwt.WithIssuer(s.jwtIssuer))
	if parseErr != nil {
		slog.WarnContext(ctx, "Failed to parse JWT", "error", parseErr)
		return false, uuid.Nil, "", "", "Token invalid or expired", nil // Nil err agar middleware return 401
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		slog.WarnContext(ctx, "Invalid JWT claims")
		return false, uuid.Nil, "", "", "Invalid token", nil
	}
	jti := claims.ID
	if jti != "" {
		isBlacklisted, err := s.jwtBlacklistRepo.IsBlacklisted(ctx, jti)
		if err != nil {
			// Ini baru benar-benar 500 karena Redis/DB bermasalah
			slog.ErrorContext(ctx, "Failed to check token blacklist", "error", err)
			return false, uuid.Nil, "", "", "Internal server error during token validation", err
		}
		if isBlacklisted {
			slog.WarnContext(ctx, "Token has been revoked", "jti", jti)
			return false, uuid.Nil, "", "", "Token has been revoked", nil
		}
	}

	return true, claims.UserID, claims.Username, claims.Role, "", nil
}

func (s *jwtTokenService) BlacklistToken(ctx context.Context, jti string, expiration time.Duration) error {
	return s.jwtBlacklistRepo.AddToBlacklist(ctx, jti, expiration)
}
