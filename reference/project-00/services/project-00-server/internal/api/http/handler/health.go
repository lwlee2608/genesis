package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lwlee2608/project-00/internal/api/http/dto"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Check(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, dto.HealthResponse{Status: "ok"})
}
