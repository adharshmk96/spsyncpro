package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

type healthResponse struct {
	Status string `json:"status"`
}

// Health returns API health status.
//
// @Summary      Health check
// @Description  Returns whether the API is running
// @Tags         health
// @Produce      json
// @Success      200  {object}  healthResponse
// @Router       /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, healthResponse{Status: "ok"})
}
