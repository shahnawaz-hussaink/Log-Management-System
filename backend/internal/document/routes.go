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
	r.GET("/document-types", handler.GetDocumentTypes)
	r.GET("/documents/:id", handler.GetDetails)
	r.GET("/documents/:id/submissions", handler.GetSubmissions)
	r.GET("/documents/:id/download", handler.Download)
	r.GET("/documents/:id/preview-pdf", handler.PreviewPDF)
	r.GET("/attachments/:id/download", handler.DownloadAttachment)
	r.PUT("/documents/:id/replace", handler.Replace)
	r.POST("/documents/:id/action", handler.TakeAction)

	r.POST("/documents/:id/notes", handler.AppendNote)
	r.PUT("/documents/:id/draft", handler.SaveDraft)
	r.POST("/documents/:id/attachments", handler.AddAttachment)
	r.GET("/notifications", handler.GetNotifications)
	r.GET("/reports", handler.GetReports)
	r.GET("/my-history", handler.GetMyHistory)

	// Files endpoints
	r.POST("/files", handler.CreateFile)
	r.GET("/files", handler.ListFiles)
	r.GET("/files/:id", handler.GetFileDetails)
	r.POST("/files/:id/forward", handler.ForwardFile)
	r.POST("/files/:id/attach-receipt", handler.AttachReceipt)
	r.PUT("/files/:id/close", handler.CloseFile)
	r.PUT("/files/:id/archive", handler.ArchiveFile)

	// Notes endpoints
	r.POST("/files/:id/notes", handler.CreateNote)
	r.PUT("/notes/:id", handler.UpdateNote)
	r.POST("/notes/:id/publish", handler.PublishNote)
}
