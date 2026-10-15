package middlewares

import (
	"net/http"
	"strings"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	contextpkg "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/context"
	appErrors "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services/token"

	"github.com/labstack/echo/v4"
)

type AuthMiddlewareOptions struct {
	TokenService token.TokenService
}

func AuthMiddleware(opts AuthMiddlewareOptions) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			ctx := req.Context()

			authHeader := req.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error: appErrors.ErrInvalidTokenFormat.Error(),
				})
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == "" {
				return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error: appErrors.ErrInvalidTokenFormat.Error(),
				})
			}

			isValid, userID, username, role, errMsg, err := opts.TokenService.ValidateToken(ctx, tokenString)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Error: appErrors.ErrInternalServerError.Error(),
				})
			}

			if !isValid {
				return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error: "Unauthorized: " + errMsg,
				})
			}

			ctx = contextpkg.SetUserIDToContext(ctx, userID)
			ctx = contextpkg.SetUsernameToContext(ctx, username)
			ctx = contextpkg.SetRoleToContext(ctx, role)
			ctx = contextpkg.SetAuthorizationHeaderToContext(ctx, authHeader)

			c.SetRequest(req.WithContext(ctx))

			return next(c)
		}
	}
}

func RequireRoles(allowedRoles ...string) echo.MiddlewareFunc {
	roleSet := make(map[string]struct{})
	for _, r := range allowedRoles {
		roleSet[r] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, err := contextpkg.GetRoleFromContext(c.Request().Context())
			if err != nil {
				return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Unauthorized"})
			}

			if _, allowed := roleSet[role]; !allowed {
				return c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Access denied"})
			}

			return next(c)
		}
	}
}
