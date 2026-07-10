package document

import (
	"office-file-sharing/backend/internal/shared/middleware"

	"github.com/labstack/echo/v4"
)

// RegisterRoutes registers all document workflows under AuthMiddleware
func RegisterRoutes(g *echo.Group, handler *Handler, jwtSecret []byte) {
	r := g.Group("")
	r.Use(middleware.AuthMiddleware(jwtSecret))

	r.POST("/documents", handler.Upload)
	r.GET("/documents", handler.List)
	r.GET("/documents/:id", handler.GetDetails)
	r.GET("/documents/:id/download", handler.Download)
	r.PUT("/documents/:id/replace", handler.Replace)
	r.POST("/documents/:id/action", handler.TakeAction)
}
