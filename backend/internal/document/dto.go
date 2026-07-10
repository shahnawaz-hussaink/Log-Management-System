package document

import (
	"time"

	"office-file-sharing/backend/internal/shared/models"

	"github.com/google/uuid"
)

type ActionRequest struct {
	ActorID   uuid.UUID  `json:"actor_id"`
	TargetID  *uuid.UUID `json:"target_id"` // Used if sending back or routing elsewhere
	Action    string     `json:"action"`    // Approve, Reject, Sent Back
	Remarks   string     `json:"remarks"`
	Signature string     `json:"signature"`
}

type DocumentResponse struct {
	ID             uuid.UUID             `json:"id"`
	Filename       string                `json:"filename"`
	FilePath       string                `json:"file_path"`
	UploaderID     uuid.UUID             `json:"uploader_id"`
	CurrentOwnerID uuid.UUID             `json:"current_owner_id"`
	Status         models.DocumentStatus `json:"status"`
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	UniqueNumber   string                `json:"unique_number"`
	Tags           string                `json:"tags"`
	Category       string                `json:"category"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`

	Uploader     models.User `json:"uploader"`
	CurrentOwner models.User `json:"current_owner"`
}

type HistoryResponse struct {
	ID         uuid.UUID             `json:"id"`
	DocumentID uuid.UUID             `json:"document_id"`
	ActorID    uuid.UUID             `json:"actor_id"`
	TargetID   *uuid.UUID            `json:"target_id"`
	Action     models.WorkflowAction `json:"action"`
	Remarks    string                `json:"remarks"`
	Signature  string                `json:"signature"`
	CreatedAt  time.Time             `json:"created_at"`

	Actor  models.User  `json:"actor"`
	Target *models.User `json:"target"`
}

type DocumentDetailsResponse struct {
	Document DocumentResponse  `json:"document"`
	History  []HistoryResponse `json:"history"`
}
