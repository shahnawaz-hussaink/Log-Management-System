package user

import (
	"fmt"
	"net/http"

	"office-file-sharing/backend/internal/shared/config"
	"office-file-sharing/backend/internal/shared/email"
	"office-file-sharing/backend/internal/shared/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Handler struct {
	service Service
	db      *gorm.DB
}

func NewHandler(service Service, db *gorm.DB) *Handler {
	return &Handler{service: service, db: db}
}

func (h *Handler) GetUsers(c echo.Context) error {
	actorIDStr := c.Get("user_id").(string)
	actorID, err := uuid.Parse(actorIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID in token"})
	}

	users, err := h.service.GetUsers(actorID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch users"})
	}
	return c.JSON(http.StatusOK, users)
}

func (h *Handler) SendManualEmail(c echo.Context) error {
	type Request struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	var req Request
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	if req.To == "" || req.Subject == "" || req.Body == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Fields 'to', 'subject', and 'body' are required"})
	}

	cfg := config.Load()
	if cfg.SMTPHost == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "SMTP server is not configured in backend .env file"})
	}

	err := email.SendMail(cfg, []string{req.To}, req.Subject, req.Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to send email: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Email sent successfully"})
}

// UpdatePassword allows authenticated user to change their own password
func (h *Handler) UpdatePassword(c echo.Context) error {
	type Request struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	actorIDStr := c.Get("user_id").(string)
	actorID, err := uuid.Parse(actorIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID in token"})
	}

	var req Request
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "current_password and new_password are required"})
	}
	if len(req.NewPassword) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "New password must be at least 8 characters"})
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", actorID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Current password is incorrect"})
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to hash new password"})
	}

	if err := h.db.Model(&user).Update("password_hash", string(newHash)).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update password"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Password updated successfully"})
}

// UpdatePhone allows authenticated user to update their phone number
func (h *Handler) UpdatePhone(c echo.Context) error {
	type Request struct {
		Phone string `json:"phone"`
	}

	actorIDStr := c.Get("user_id").(string)
	actorID, err := uuid.Parse(actorIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID in token"})
	}

	var req Request
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}
	if req.Phone == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "phone is required"})
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", actorID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	if err := h.db.Model(&user).Update("phone", req.Phone).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update phone number"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Phone number updated successfully"})
}
