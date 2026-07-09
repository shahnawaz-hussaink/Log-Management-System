package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name         string    `gorm:"size:255;not null"`
	Email        string    `gorm:"size:255;not null;unique"`
	PasswordHash string    `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type DocumentStatus string

const (
	StatusDraft           DocumentStatus = "Draft"
	StatusPendingApproval DocumentStatus = "Pending Approval"
	StatusApproved        DocumentStatus = "Approved"
	StatusRejected        DocumentStatus = "Rejected"
	StatusSentBack        DocumentStatus = "Sent Back"
)

type Document struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key"`
	Filename       string    `gorm:"size:255;not null"`
	FilePath       string    `gorm:"size:1024;not null"`
	UploaderID     uuid.UUID `gorm:"type:uuid;not null"`
	CurrentOwnerID uuid.UUID `gorm:"type:uuid;not null"`
	Status         DocumentStatus `gorm:"size:50;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	Uploader     User `gorm:"foreignKey:UploaderID"`
	CurrentOwner User `gorm:"foreignKey:CurrentOwnerID"`
}

type WorkflowAction string

const (
	ActionUploaded     WorkflowAction = "Uploaded"
	ActionSubmitted    WorkflowAction = "Submitted"
	ActionApproved     WorkflowAction = "Approved"
	ActionRejected     WorkflowAction = "Rejected"
	ActionSentBack     WorkflowAction = "Sent Back"
	ActionFileReplaced WorkflowAction = "File Replaced"
)

type WorkflowHistory struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key"`
	DocumentID uuid.UUID      `gorm:"type:uuid;not null"`
	ActorID    uuid.UUID      `gorm:"type:uuid;not null"`
	TargetID   *uuid.UUID     `gorm:"type:uuid"` // Nullable for end states
	Action     WorkflowAction `gorm:"size:50;not null"`
	Remarks    string         `gorm:"type:text"`
	CreatedAt  time.Time

	Document Document `gorm:"foreignKey:DocumentID"`
	Actor    User     `gorm:"foreignKey:ActorID"`
	Target   *User    `gorm:"foreignKey:TargetID"`
}

func (base *User) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return
}

func (base *Document) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return
}

func (base *WorkflowHistory) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return
}
