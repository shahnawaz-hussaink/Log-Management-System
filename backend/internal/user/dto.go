package user

import (
	"time"

	"github.com/google/uuid"
	"office-file-sharing/backend/internal/shared/models"
)

type UserResponse struct {
	ID        uuid.UUID  `json:"ID"`
	Name      string     `json:"Name"`
	Email     string     `json:"Email"`
	Role      string     `json:"Role"`
	SchoolID  *uuid.UUID `json:"SchoolID"`
	CreatedAt time.Time      `json:"CreatedAt"`
	UpdatedAt time.Time      `json:"UpdatedAt"`
	School    *models.School `json:"School,omitempty"`
}
