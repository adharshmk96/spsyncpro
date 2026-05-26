package routes

import (
	"log/slog"

	_ "spsyncapi/docs"
	"spsyncapi/internal/auth"
	"spsyncapi/internal/handlers"
	"spsyncapi/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Deps carries all handler and middleware dependencies needed at route registration.
type Deps struct {
	AuthHandler         *handlers.AuthHandler
	OrganizationHandler *handlers.OrganizationHandler
	BucketStoreHandler  *handlers.BucketStoreHandler
	AuthService         *auth.Service
	JWTConfig           auth.JWTConfig
	Logger              *slog.Logger
}

// Register wires all routes onto the provided engine.
func Register(router *gin.Engine, deps Deps) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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

			protected.POST("/organizations", deps.OrganizationHandler.Create)
			protected.GET("/organizations", deps.OrganizationHandler.List)
			protected.GET("/organizations/:id", deps.OrganizationHandler.Get)
			protected.PUT("/organizations/:id", deps.OrganizationHandler.Update)
			protected.DELETE("/organizations/:id", deps.OrganizationHandler.Delete)

			protected.POST("/bucket-stores", deps.BucketStoreHandler.Create)
			protected.GET("/bucket-stores", deps.BucketStoreHandler.List)
			protected.GET("/bucket-stores/:id", deps.BucketStoreHandler.Get)
			protected.PUT("/bucket-stores/:id", deps.BucketStoreHandler.Update)
			protected.DELETE("/bucket-stores/:id", deps.BucketStoreHandler.Delete)
		}
	}
}
