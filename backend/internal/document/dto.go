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

type AttachmentResponse struct {
	ID         uuid.UUID `json:"ID"`
	DocumentID uuid.UUID `json:"DocumentID"`
	Filename   string    `json:"Filename"`
	UploadedBy uuid.UUID `json:"UploadedBy"`
	CreatedAt  time.Time `json:"CreatedAt"`
}

type DocumentResponse struct {
	ID             uuid.UUID             `json:"ID"`
	Filename       string                `json:"Filename"`
	FilePath       string                `json:"FilePath"`
	UploaderID     uuid.UUID             `json:"UploaderID"`
	CurrentOwnerID uuid.UUID             `json:"CurrentOwnerID"`
	Status         models.DocumentStatus `json:"Status"`
	Title          string                `json:"Title"`
	Description    string                `json:"Description"`
	UniqueNumber   string                `json:"UniqueNumber"`
	Tags           string                `json:"Tags"`
	Category       string                `json:"Category"`
	Priority       string                `json:"Priority"`
	Direction      string                `json:"Direction"`
	AssignedAt     time.Time             `json:"AssignedAt"`
	ReferralOwnerID *uuid.UUID            `json:"ReferralOwnerID"`
	NotingSheet    string                `json:"NotingSheet"`
	DraftSpace     string                `json:"DraftSpace"`
	CreatedAt      time.Time             `json:"CreatedAt"`
	UpdatedAt      time.Time             `json:"UpdatedAt"`

	Uploader     models.User          `json:"Uploader"`
	CurrentOwner models.User          `json:"CurrentOwner"`
	Attachments  []AttachmentResponse `json:"Attachments"`
	HasActed     bool                 `json:"HasActed"`
}

type HistoryResponse struct {
	ID         uuid.UUID             `json:"ID"`
	DocumentID uuid.UUID             `json:"DocumentID"`
	ActorID    uuid.UUID             `json:"ActorID"`
	TargetID   *uuid.UUID            `json:"TargetID"`
	Action     models.WorkflowAction `json:"Action"`
	Remarks    string                `json:"Remarks"`
	Signature  string                `json:"Signature"`
	CreatedAt  time.Time             `json:"CreatedAt"`

	Actor  models.User  `json:"Actor"`
	Target *models.User `json:"Target"`
}

type DocumentDetailsResponse struct {
	Document DocumentResponse  `json:"document"`
	History  []HistoryResponse `json:"history"`
}

type UserHistoryEntry struct {
	ID            uuid.UUID             `json:"ID"`
	DocumentID    uuid.UUID             `json:"DocumentID"`
	ActorID       uuid.UUID             `json:"ActorID"`
	TargetID      *uuid.UUID            `json:"TargetID"`
	Action        models.WorkflowAction `json:"Action"`
	Remarks       string                `json:"Remarks"`
	Signature     string                `json:"Signature"`
	CreatedAt     time.Time             `json:"CreatedAt"`
	Actor         models.User           `json:"Actor"`
	Target        *models.User          `json:"Target"`
	DocumentTitle string                `json:"DocumentTitle"`
	DocumentNum   string                `json:"DocumentNum"`
	DocumentStatus models.DocumentStatus `json:"DocumentStatus"`
	Category      string                `json:"Category"`
	Priority      string                `json:"Priority"`
}

type FileResponse struct {
	ID             uuid.UUID         `json:"ID"`
	SchoolID       *uuid.UUID        `json:"SchoolID"`
	FileNumber     string            `json:"FileNumber"`
	Title          string            `json:"Title"`
	Description    string            `json:"Description"`
	Category       string            `json:"Category"`
	SubCategory    string            `json:"SubCategory"`
	CreatorID      uuid.UUID         `json:"CreatorID"`
	CurrentOwnerID uuid.UUID         `json:"CurrentOwnerID"`
	Status         models.FileStatus `json:"Status"`
	CreatedAt      time.Time         `json:"CreatedAt"`
	UpdatedAt      time.Time         `json:"UpdatedAt"`

	Creator      models.User        `json:"Creator"`
	CurrentOwner models.User        `json:"CurrentOwner"`
	Receipts     []DocumentResponse `json:"Receipts"`
}

type NoteResponse struct {
	ID              uuid.UUID       `json:"ID"`
	FileID          uuid.UUID       `json:"FileID"`
	AuthorID        uuid.UUID       `json:"AuthorID"`
	Type            models.NoteType `json:"Type"`
	Content         string          `json:"Content"`
	Signature       string          `json:"Signature"`
	IsDiscarded     bool            `json:"IsDiscarded"`
	PublishedFromID *uuid.UUID      `json:"PublishedFromID"`
	CreatedAt       time.Time       `json:"CreatedAt"`
	UpdatedAt       time.Time       `json:"UpdatedAt"`

	Author models.User `json:"Author"`
}

type FileDetailsResponse struct {
	File  FileResponse   `json:"file"`
	Notes []NoteResponse `json:"notes"`
}

type CreateFileRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	SubCategory string `json:"sub_category"`
}

type CreateNoteRequest struct {
	Content string          `json:"content"`
	Type    models.NoteType `json:"type"` // Green or Yellow
}

type PublishNoteRequest struct {
	Signature string `json:"signature"`
}

type ForwardFileRequest struct {
	TargetOwnerID uuid.UUID `json:"target_owner_id"`
}

type AttachReceiptRequest struct {
	ReceiptID uuid.UUID `json:"receipt_id"`
}
