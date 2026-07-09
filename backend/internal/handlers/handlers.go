package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"office-file-sharing/backend/internal/db"
	"office-file-sharing/backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// In a real app we'd hash passwords and use JWT. Keeping it simple here or mocking.
// Actually, let's implement basic user fetch for now.

func GetUsers(c echo.Context) error {
	var users []models.User
	if err := db.DB.Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch users"})
	}
	return c.JSON(http.StatusOK, users)
}

func UploadDocument(c echo.Context) error {
	// Parse multipart form
	uploaderIDStr := c.FormValue("uploader_id")
	targetOwnerIDStr := c.FormValue("target_owner_id")

	uploaderID, err := uuid.Parse(uploaderIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid uploader ID"})
	}

	targetOwnerID, err := uuid.Parse(targetOwnerIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid target owner ID"})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "File is required"})
	}

	// Save file
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read file"})
	}
	defer src.Close()

	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create upload directory"})
	}

	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
	filePath := filepath.Join(uploadDir, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save file"})
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save file content"})
	}

	title := c.FormValue("title")
	if title == "" {
		title = file.Filename
	}
	description := c.FormValue("description")
	category := c.FormValue("category")
	if category == "" {
		category = "Document"
	}
	tags := c.FormValue("tags")

	// Generate a unique number following the convention: DOC-YYYYMMDD-XXXXXX (last 6 chars of a new UUID)
	now := time.Now()
	uniqueNumber := fmt.Sprintf("DOC-%s-%s", now.Format("20060102"), strings.ToUpper(uuid.New().String()[:6]))

	// Create DB Record
	doc := models.Document{
		ID:             uuid.New(),
		Filename:       file.Filename,
		FilePath:       filePath,
		UploaderID:     uploaderID,
		CurrentOwnerID: targetOwnerID,
		Status:         models.StatusPendingApproval,
		Title:          title,
		Description:    description,
		UniqueNumber:   uniqueNumber,
		Tags:           tags,
		Category:       category,
	}

	if err := db.DB.Create(&doc).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save document record"})
	}

	// Create Workflow History
	history := models.WorkflowHistory{
		DocumentID: doc.ID,
		ActorID:    uploaderID,
		TargetID:   &targetOwnerID,
		Action:     models.ActionSubmitted,
		Remarks:    "Initial Submission",
	}

	db.DB.Create(&history)

	return c.JSON(http.StatusOK, doc)
}

func GetDocuments(c echo.Context) error {
	userIDStr := c.QueryParam("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}

	search := c.QueryParam("search")

	var documents []models.Document
	query := db.DB.Preload("Uploader").Preload("CurrentOwner").Where("uploader_id = ? OR current_owner_id = ?", userID, userID)

	if search != "" {
		searchLike := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(title) LIKE ? OR LOWER(description) LIKE ? OR LOWER(unique_number) LIKE ? OR LOWER(tags) LIKE ? OR LOWER(category) LIKE ?",
			searchLike, searchLike, searchLike, searchLike, searchLike,
		)
	}

	if err := query.Find(&documents).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch documents"})
	}

	return c.JSON(http.StatusOK, documents)
}

func GetDocumentDetails(c echo.Context) error {
	idStr := c.Param("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid document ID"})
	}

	var doc models.Document
	if err := db.DB.Preload("Uploader").Preload("CurrentOwner").First(&doc, "id = ?", docID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Document not found"})
	}

	var history []models.WorkflowHistory
	db.DB.Preload("Actor").Preload("Target").Where("document_id = ?", docID).Order("created_at asc").Find(&history)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"document": doc,
		"history":  history,
	})
}

func DownloadDocument(c echo.Context) error {
	idStr := c.Param("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid document ID"})
	}

	var doc models.Document
	if err := db.DB.First(&doc, "id = ?", docID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Document not found"})
	}

	return c.File(doc.FilePath)
}

type ActionRequest struct {
	ActorID  uuid.UUID  `json:"actor_id"`
	TargetID *uuid.UUID `json:"target_id"` // Used if sending back or routing elsewhere
	Action   string     `json:"action"` // Approve, Reject, Sent Back
	Remarks  string     `json:"remarks"`
}

func DocumentAction(c echo.Context) error {
	idStr := c.Param("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid document ID"})
	}

	var req ActionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	var doc models.Document
	if err := db.DB.First(&doc, "id = ?", docID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Document not found"})
	}

	if doc.CurrentOwnerID != req.ActorID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "You are not the current owner of this document"})
	}

	var newStatus models.DocumentStatus
	var nextOwnerID uuid.UUID
	wfAction := models.WorkflowAction(req.Action)

	switch wfAction {
	case models.ActionApproved:
		newStatus = models.StatusApproved
		nextOwnerID = req.ActorID // Keeps ownership or could transfer to system/uploader
	case models.ActionRejected:
		newStatus = models.StatusRejected
		nextOwnerID = doc.UploaderID // Returns to uploader
	case models.ActionSentBack:
		newStatus = models.StatusSentBack
		nextOwnerID = doc.UploaderID // Returns to uploader
	case models.ActionForwarded, "Forward":
		newStatus = models.StatusPendingApproval
		if req.TargetID == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Target ID is required to forward document"})
		}
		nextOwnerID = *req.TargetID
		wfAction = models.ActionForwarded
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid action"})
	}

	doc.Status = newStatus
	doc.CurrentOwnerID = nextOwnerID
	db.DB.Save(&doc)

	history := models.WorkflowHistory{
		DocumentID: doc.ID,
		ActorID:    req.ActorID,
		TargetID:   &nextOwnerID,
		Action:     wfAction,
		Remarks:    req.Remarks,
	}
	db.DB.Create(&history)

	return c.JSON(http.StatusOK, doc)
}

func ReplaceDocument(c echo.Context) error {
	idStr := c.Param("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid document ID"})
	}

	uploaderIDStr := c.FormValue("uploader_id")
	targetOwnerIDStr := c.FormValue("target_owner_id")
	remarks := c.FormValue("remarks")

	uploaderID, err := uuid.Parse(uploaderIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid uploader ID"})
	}

	targetOwnerID, err := uuid.Parse(targetOwnerIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid target owner ID"})
	}

	var doc models.Document
	if err := db.DB.First(&doc, "id = ?", docID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Document not found"})
	}

	if doc.Status != models.StatusSentBack {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Document must be 'Sent Back' to be replaced or resubmitted"})
	}

	if doc.UploaderID != uploaderID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Only the uploader can replace or resubmit this document"})
	}

	file, err := c.FormFile("file")
	fileReplaced := true

	if err != nil {
		if err == http.ErrMissingFile || err == http.ErrNotMultipart {
			fileReplaced = false
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to retrieve file"})
		}
	}

	if fileReplaced {
		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read file"})
		}
		defer src.Close()

		uploadDir := "./uploads"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create upload directory"})
		}

		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
		filePath := filepath.Join(uploadDir, filename)

		dst, err := os.Create(filePath)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save file"})
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save file content"})
		}

		doc.Filename = file.Filename
		doc.FilePath = filePath
	}

	// Update Document Status and Owner
	doc.Status = models.StatusPendingApproval
	doc.CurrentOwnerID = targetOwnerID
	db.DB.Save(&doc)

	// Determine action and remarks
	var wfAction models.WorkflowAction
	historyRemarks := remarks
	if fileReplaced {
		wfAction = models.ActionFileReplaced
		if historyRemarks == "" {
			historyRemarks = "File Replaced and Resubmitted"
		}
	} else {
		wfAction = models.ActionResubmitted
		if historyRemarks == "" {
			historyRemarks = "Document Resubmitted"
		}
	}

	// Create Workflow History
	history := models.WorkflowHistory{
		DocumentID: doc.ID,
		ActorID:    uploaderID,
		TargetID:   &targetOwnerID,
		Action:     wfAction,
		Remarks:    historyRemarks,
	}
	db.DB.Create(&history)

	return c.JSON(http.StatusOK, doc)
}

// LoginRequest with email and password
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	var user models.User
	if err := db.DB.First(&user, "email = ?", req.Email).Error; err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid email or password"})
	}

	// Verify password hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid email or password"})
	}

	// Generate JWT token
	claims := &JWTCustomClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(JWTSecretKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate authentication token"})
	}

	// Remove password hash from response
	user.PasswordHash = ""

	return c.JSON(http.StatusOK, map[string]interface{}{
		"token": tokenString,
		"user":  user,
	})
}
