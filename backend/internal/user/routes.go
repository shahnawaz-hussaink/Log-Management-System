package user

import (
	"office-file-sharing/backend/internal/shared/middleware"

	"github.com/labstack/echo/v4"
)

// RegisterRoutes registers user-related endpoints under the Echo group
func RegisterRoutes(g *echo.Group, handler *Handler, jwtSecret []byte) {
	r := g.Group("")
	r.Use(middleware.AuthMiddleware(jwtSecret))
	r.GET("/users", handler.GetUsers)
	r.POST("/send-email", handler.SendManualEmail)
	r.PUT("/profile/password", handler.UpdatePassword)
	r.PUT("/profile/phone", handler.UpdatePhone)
}
