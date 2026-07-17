package document

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"office-file-sharing/backend/internal/shared/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Upload(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	uploaderID, _ := uuid.Parse(authenticatedUserIDStr)

	category := c.FormValue("category")

	targetOwnerIDsStr := c.FormValue("target_owner_ids")
	var targetOwnerIDs []uuid.UUID
	var err error
	if category != "Circular" && category != "Assignment Broadcast" {
		ids := strings.Split(targetOwnerIDsStr, ",")
		for _, idStr := range ids {
			idStr = strings.TrimSpace(idStr)
			if idStr == "" {
				continue
			}
			id, parseErr := uuid.Parse(idStr)
			if parseErr != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid target owner ID in chain"})
			}
			targetOwnerIDs = append(targetOwnerIDs, id)
		}
		if len(targetOwnerIDs) == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "At least one target owner is required"})
		}
	}

	title := c.FormValue("title")
	description := c.FormValue("description")
	tags := c.FormValue("tags")
	priority := c.FormValue("priority")
	direction := c.FormValue("direction")
	targetClass := c.FormValue("target_class")
	
	refDocIDStr := c.FormValue("ref_document_id")
	var refDocID *uuid.UUID
	if refDocIDStr != "" {
		id, parseErr := uuid.Parse(refDocIDStr)
		if parseErr == nil {
			refDocID = &id
		}
	}

	fileHeader, err := c.FormFile("file")
	if err != nil && category != "Assignment Broadcast" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "File is required"})
	}

	res, err := h.service.Upload(uploaderID, targetOwnerIDs, title, description, category, tags, priority, direction, targetClass, refDocID, fileHeader)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, res)
}

func (h *Handler) List(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	search := c.QueryParam("search")

	res, err := h.service.List(userID, search)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch documents"})
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) GetDetails(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	idStr := c.Param("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid document ID"})
	}

	res, err := h.service.GetDetails(docID, userID)
	if err != nil {
		if err.Error() == "you are not authorized to view or access this document" {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) GetSubmissions(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	idStr := c.Param("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid document ID"})
	}

	res, err := h.service.GetSubmissions(docID, userID)
	if err != nil {
		if err.Error() == "you are not authorized to view this document (outside school scope)" {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) Download(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	idStr := c.Param("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid document ID"})
	}

	filePath, err := h.service.GetFilePathForDownload(docID, userID)
	if err != nil {
		if err.Error() == "you are not authorized to view or access this document" {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "Not Found", "debug_path": filePath})
	}
	
	// Convert to absolute path since c.File seems to sometimes fail with relative paths depending on echo version
	absPath, absErr := filepath.Abs(filePath)
	if absErr == nil {
		filePath = absPath
	}

	return c.File(filePath)
}

func (h *Handler) PreviewPDF(c echo.Context) error {
	log.Printf("[Preview Handler] Request received for previewing document")
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	idStr := c.Param("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[Preview Handler] Invalid document ID: %s", idStr)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid document ID"})
	}

	log.Printf("[Preview Handler] Calling service GetPreviewPDFPath for doc ID %s, user ID %s", docID, userID)
	filePath, isTemp, err := h.service.GetPreviewPDFPath(docID, userID)
	if err != nil {
		log.Printf("[Preview Handler] GetPreviewPDFPath failed for doc ID %s: %v", docID, err)
		if err.Error() == "you are not authorized to view or access this document" {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	log.Printf("[Preview Handler] PDF file path resolved: %s (isTemp: %t)", filePath, isTemp)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("[Preview Handler] Resolved PDF file path does not exist on disk: %s", filePath)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "PDF file not found on disk"})
	}

	absPath, absErr := filepath.Abs(filePath)
	if absErr == nil {
		filePath = absPath
	}

	// If it is a temporary file, delete the temporary directory after the response is sent.
	if isTemp {
		defer func() {
			tempDir := filepath.Dir(filePath)
			log.Printf("[Preview Handler] Deferred Cleanup: Removing temporary directory %s", tempDir)
			if err := os.RemoveAll(tempDir); err != nil {
				log.Printf("[Preview Handler] Deferred Cleanup warning: failed to remove temp directory %s: %v", tempDir, err)
			}
		}()
	}

	log.Printf("[Preview Handler] Streaming PDF: serving %s with Content-Type: application/pdf", filePath)
	c.Response().Header().Set("Content-Type", "application/pdf")
	return c.File(filePath)
}

func (h *Handler) DownloadAttachment(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	idStr := c.Param("id")
	attID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid attachment ID"})
	}

	filePath, err := h.service.GetAttachmentFilePathForDownload(attID, userID)
	if err != nil {
		if err.Error() == "you are not authorized to view or access this document" {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.JSON(http.StatusNotFound, map[string]string{"message": "Not Found", "debug_path": filePath})
	}

	absPath, absErr := filepath.Abs(filePath)
	if absErr == nil {
		filePath = absPath
	}

	return c.File(filePath)
}

func (h *Handler) Replace(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	idStr := c.Param("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid document ID"})
	}

	targetOwnerIDStr := c.FormValue("target_owner_id")
	targetOwnerID, err := uuid.Parse(targetOwnerIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid target owner ID"})
	}

	remarks := c.FormValue("remarks")
	title := c.FormValue("title")
	description := c.FormValue("description")
	category := c.FormValue("category")
	tags := c.FormValue("tags")
	priority := c.FormValue("priority")
	direction := c.FormValue("direction")

	fileHeader, _ := c.FormFile("file") // Optional file

	res, err := h.service.Replace(docID, userID, targetOwnerID, title, description, category, tags, priority, direction, fileHeader, remarks)
	if err != nil {
		if err.Error() == "only the original uploader is authorized to replace or resubmit this document" {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) TakeAction(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	idStr := c.Param("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid document ID"})
	}

	var req ActionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	res, err := h.service.TakeAction(docID, userID, req)
	if err != nil {
		if err.Error() == "you are not authorized to act on this document as you are not the current owner" {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, res)
}

func (h *Handler) GetDocumentTypes(c echo.Context) error {
	var list []models.DocumentType
	// Fetch all active document types
	err := h.service.(*service).repo.(*repository).db.Find(&list, "active = ?", true).Error
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch document types"})
	}
	return c.JSON(http.StatusOK, list)
}


func (h *Handler) AppendNote(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	idStr := c.Param("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid document ID"})
	}

	var req struct {
		Note string `json:"note"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	actorIP := c.RealIP()
	if actorIP == "" {
		actorIP = "127.0.0.1"
	}

	res, err := h.service.AppendNote(docID, userID, req.Note, actorIP)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) SaveDraft(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	idStr := c.Param("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid document ID"})
	}

	var req struct {
		Draft string `json:"draft"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	res, err := h.service.SaveDraft(docID, userID, req.Draft)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) AddAttachment(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	idStr := c.Param("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid document ID"})
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Attachment file is required"})
	}

	res, err := h.service.AddAttachment(docID, userID, fileHeader)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, res)
}

func (h *Handler) GetNotifications(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	res, err := h.service.GetNotifications(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) GetReports(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	var user models.User
	err := h.service.(*service).repo.(*repository).db.First(&user, "id = ?", userID).Error
	if err != nil || user.Role != "Admin" || user.SchoolID == nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized to view school reports"})
	}

	res, err := h.service.GetReports(*user.SchoolID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) GetMyHistory(c echo.Context) error {
	authenticatedUserIDStr := c.Get("user_id").(string)
	userID, _ := uuid.Parse(authenticatedUserIDStr)

	res, err := h.service.GetMyHistory(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, res)
}
