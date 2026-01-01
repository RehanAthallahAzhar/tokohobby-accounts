package routes

import (
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/handlers"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/middlewares"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services/token"
	"github.com/labstack/echo/v4"
)

func InitRoutes(e *echo.Echo, api *handlers.UserHandler, tokenService token.TokenService) {
	e.Static("/static", "template")

	e.POST("/api/v1/accounts/register", api.RegisterUser)
	e.POST("/api/v1/accounts/login", api.Login)
	e.POST("/api/v1/accounts/refresh", api.RefreshSession)
	e.POST("/api/v1/accounts/logout", api.Logout)

	jwtAuthMiddleware := middlewares.AuthMiddleware(middlewares.AuthMiddlewareOptions{
		TokenService: tokenService,
	})

	accountProtectedGroup := e.Group("/api/v1/accounts")
	accountProtectedGroup.Use(jwtAuthMiddleware)
	{
		// all users
		accountProtectedGroup.GET("/profile", api.GetUserProfile)
		accountProtectedGroup.PUT("/update", api.UpdateUser)
		accountProtectedGroup.DELETE("/delete/:id", api.DeleteUser)

		// admin
		accountProtectedGroup.GET("/list", api.GetAllUsers, middlewares.RequireRoles("admin"))
		accountProtectedGroup.GET("/:id", api.GetUserById, middlewares.RequireRoles("admin"))
	}
}
