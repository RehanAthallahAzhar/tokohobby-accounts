package context

import (
	"context"

	"github.com/google/uuid"

	apperrors "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"
)

type contextKey string

const UserIDKey contextKey = "userID"
const RoleKey contextKey = "role"
const UsernameKey contextKey = "username"
const SessionIDKey contextKey = "sessionID"
const AuthorizationHeaderKey contextKey = "authorizationHeader"

func GetUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	val := ctx.Value(UserIDKey)
	if val == nil {
		return uuid.Nil, apperrors.ErrFailedToGetUserIDFromContext
	}

	id, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil, apperrors.ErrFailedToGetUserIDFromContext
	}

	return id, nil
}

func GetRoleFromContext(ctx context.Context) (string, error) {
	val := ctx.Value(RoleKey)
	if val == nil {
		return "", apperrors.ErrFailedToGetRoleFromContext
	}

	role, ok := val.(string)
	if !ok {
		return "", apperrors.ErrFailedToGetRoleFromContext
	}

	return role, nil
}
func GetAuthorizationHeaderFromContext(ctx context.Context) (string, error) {
	val := ctx.Value(AuthorizationHeaderKey)
	if val == nil {
		return "", apperrors.ErrFailedToGetAuthorizationHeaderFromContext
	}

	authHeader, ok := val.(string)
	if !ok {
		return "", apperrors.ErrFailedToGetAuthorizationHeaderFromContext
	}

	return authHeader, nil
}

func GetUsernameFromContext(ctx context.Context) (string, error) {
	val := ctx.Value(UsernameKey)
	if val == nil {
		return "", apperrors.ErrFailedToGetUsernameFromContext
	}

	username, ok := val.(string)
	if !ok {
		return "", apperrors.ErrFailedToGetUsernameFromContext
	}

	return username, nil
}

func SetRoleToContext(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, RoleKey, role)
}

func SetUsernameToContext(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, UsernameKey, username)
}

func SetUserIDToContext(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, UserIDKey, id)
}

func SetAuthorizationHeaderToContext(ctx context.Context, authHeader string) context.Context {
	return context.WithValue(ctx, AuthorizationHeaderKey, authHeader)
}

// func GetSessionOrRequestID(ctx context.Context) string {
// 	val := ctx.Value(SessionIDKey)
// 	if val != nil {
// 		return val.(string)
// 	}
// 	return trace.SpanFromContext(ctx).SpanContext().TraceID().String()
// }

// func SetSessionOrRequestID(ctx context.Context, sessionID string) context.Context {
// 	return context.WithValue(ctx, SessionIDKey, sessionID)
// }
