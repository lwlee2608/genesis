package handler

import (
	"net/http"

	"github.com/lwlee2608/project-00/internal/api/http/dto"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Check(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, dto.HealthResponse{Status: "ok"})
}
