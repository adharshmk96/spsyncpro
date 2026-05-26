package routes

import (
	"spsyncapi/internal/handlers"

	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine) {
	healthHandler := handlers.NewHealthHandler()

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler.Health)
	}
}
