package document

import (
	"net/http"

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

	targetOwnerIDStr := c.FormValue("target_owner_id")
	targetOwnerID, err := uuid.Parse(targetOwnerIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid target owner ID"})
	}

	title := c.FormValue("title")
	description := c.FormValue("description")
	category := c.FormValue("category")
	tags := c.FormValue("tags")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "File is required"})
	}

	res, err := h.service.Upload(uploaderID, targetOwnerID, title, description, category, tags, fileHeader)
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

	fileHeader, _ := c.FormFile("file") // Optional file

	res, err := h.service.Replace(docID, userID, targetOwnerID, title, description, category, tags, fileHeader, remarks)
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
