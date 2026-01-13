package routes

import (
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/handlers"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/middlewares"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services/token"
	"github.com/labstack/echo/v4"
)

func InitRoutes(e *echo.Echo, handler *handlers.UserHandler, tokenService token.TokenService) {
	e.Static("/static", "template")

	api := e.Group("/api")

	public := api.Group("/accounts")
	public.POST("/register", handler.RegisterUser)
	public.POST("/login", handler.Login)
	public.POST("/refresh", handler.RefreshSession)

	jwtAuthMiddleware := middlewares.AuthMiddleware(middlewares.AuthMiddlewareOptions{
		TokenService: tokenService,
	})

	protected := api.Group("/accounts")
	protected.Use(jwtAuthMiddleware)
	{
		// all users
		protected.GET("/profile", handler.GetUserProfile)
		protected.PUT("/", handler.UpdateUser)
		protected.DELETE("/:id", handler.DeleteUser)
		protected.POST("/logout", handler.Logout)

		// admin
		protected.GET("/", handler.GetAllUsers, middlewares.RequireRoles("admin"))
		protected.GET("/:id", handler.GetUserByID, middlewares.RequireRoles("admin"))
	}
}
