package routes

import (
	"github.com/labstack/echo/v4"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/handlers"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/middlewares"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services/token"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func InitRoutes(e *echo.Echo, userHandler *handlers.UserHandler, authHandler *handlers.AuthHandler, tokenService token.TokenService) {
	e.Static("/static", "template")

	e.GET("/swagger/*", echoSwagger.WrapHandler)

	api := e.Group("/api/v1")

	public := api.Group("/accounts")
	public.POST("/register", authHandler.Register)
	public.POST("/login", authHandler.Login)
	public.POST("/refresh", authHandler.RefreshSession)

	jwtAuthMiddleware := middlewares.AuthMiddleware(middlewares.AuthMiddlewareOptions{
		TokenService: tokenService,
	})

	protected := api.Group("/accounts")
	protected.Use(jwtAuthMiddleware)
	{
		// all users
		protected.GET("/profile", userHandler.GetUserProfile)
		protected.PATCH("", userHandler.UpdateUser)
		protected.DELETE("/:id", userHandler.DeleteUser)
		protected.POST("/logout", authHandler.Logout)

		// admin
		protected.GET("", userHandler.GetAllUsers, middlewares.RequireRoles("admin"))
		protected.GET("/:id", userHandler.GetUserByID, middlewares.RequireRoles("admin"))

	}
}
