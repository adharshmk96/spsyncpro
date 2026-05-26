package routes

import (
	"log/slog"

	"spsyncapi/internal/auth"
	"spsyncapi/internal/handlers"
	"spsyncapi/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Deps carries all handler and middleware dependencies needed at route registration.
type Deps struct {
	AuthHandler *handlers.AuthHandler
	AuthService *auth.Service
	JWTConfig   auth.JWTConfig
	Logger      *slog.Logger
}

// Register wires all routes onto the provided engine.
func Register(router *gin.Engine, deps Deps) {
	healthHandler := handlers.NewHealthHandler()

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler.Health)

		// Public auth endpoints — no JWT required.
		v1.POST("/register", deps.AuthHandler.Register)
		v1.POST("/login", deps.AuthHandler.Login)
		v1.POST("/forgot-password", deps.AuthHandler.ForgotPassword)
		v1.POST("/reset-password", deps.AuthHandler.ResetPassword)

		// Protected endpoints — JWT + session validation required.
		protected := v1.Group("")
		protected.Use(middleware.Authentication(deps.AuthService, deps.JWTConfig, deps.Logger))
		{
			protected.GET("/me", deps.AuthHandler.Me)
			protected.POST("/logout", deps.AuthHandler.Logout)
			protected.POST("/change-password", deps.AuthHandler.ChangePassword)
		}
	}
}
