// Package models contains shared database structures and models
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type School struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name      string    `gorm:"size:255;not null"`
	Slug      string    `gorm:"size:100;not null;uniqueIndex"`
	Settings  string    `gorm:"type:text"` // JSON configuration for notifications or limits
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SchoolID     *uuid.UUID `gorm:"type:uuid"`
	Name         string     `gorm:"size:255;not null"`
	Email        string     `gorm:"size:255;not null;unique"`
	PasswordHash string     `gorm:"not null"`
	Role         string     `gorm:"size:50;not null;default:'Student'"` // School roles: Student, Teacher, Principal, Parent, Admin
	ClassSection string     `gorm:"size:50"`                            // Scoped access for Student/Teacher (e.g. "10-A")
	Subject      string     `gorm:"size:100"`                           // Assigned subject for Teacher
	Phone        string     `gorm:"size:20"`                            // For notifications
	CreatedAt    time.Time
	UpdatedAt    time.Time

	School *School `gorm:"foreignKey:SchoolID"`
}

// ParentChild defines many-to-many relationship between parent and children
type ParentChild struct {
	ParentID uuid.UUID `gorm:"type:uuid;primaryKey"`
	ChildID  uuid.UUID `gorm:"type:uuid;primaryKey"`

	Parent User `gorm:"foreignKey:ParentID"`
	Child  User `gorm:"foreignKey:ChildID"`
}

type DocumentStatus string

const (
	StatusDraft           DocumentStatus = "Draft"
	StatusPendingApproval DocumentStatus = "Pending Approval"
	StatusApproved        DocumentStatus = "Approved"
	StatusRejected        DocumentStatus = "Rejected"
	StatusSentBack        DocumentStatus = "Sent Back"
	StatusClosed          DocumentStatus = "Closed"
	StatusArchived        DocumentStatus = "Archived"
)

type DocumentType struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SchoolID          uuid.UUID  `gorm:"type:uuid;not null"`
	Name              string     `gorm:"size:100;not null"`
	Slug              string     `gorm:"size:100;not null"`
	WorkflowStages    string     `gorm:"type:text;not null"` // JSON array of stages
	RequiredFields    string     `gorm:"type:text"`          // JSON array of dynamic fields
	SlaHours          int        `gorm:"default:72"`
	NeedsParentCosign bool       `gorm:"default:false"`
	Active            bool       `gorm:"default:true"`
	CreatedAt         time.Time
	UpdatedAt         time.Time

	School School `gorm:"foreignKey:SchoolID"`
}

type Document struct {
	ID              uuid.UUID      `gorm:"type:uuid;primary_key"`
	SchoolID        *uuid.UUID     `gorm:"type:uuid"`
	DocumentTypeID  *uuid.UUID     `gorm:"type:uuid"`
	Filename        string         `gorm:"size:255;not null"`
	FilePath        string         `gorm:"size:1024;not null"`
	UploaderID      uuid.UUID      `gorm:"type:uuid;not null"`
	CurrentOwnerID  uuid.UUID      `gorm:"type:uuid;not null"`
	Status          DocumentStatus `gorm:"size:50;not null"`
	Title           string         `gorm:"size:255"`
	Description     string         `gorm:"type:text"`
	UniqueNumber    string         `gorm:"size:100;uniqueIndex:idx_unique_number_version"`
	Tags            string         `gorm:"size:255"`
	Category        string         `gorm:"size:100"`
	Version         int            `gorm:"not null;default:1;uniqueIndex:idx_unique_number_version"`
	ParentDocID     *uuid.UUID     `gorm:"type:uuid"` // Linked to previous version
	CurrentStage    int            `gorm:"not null;default:1"`
	SlaDeadline     *time.Time
	Metadata        string    `gorm:"type:text"` // JSON object containing form fields
	Priority        string    `gorm:"size:50;default:'Normal'"` // Normal, Urgent, Confidential
	Direction       string    `gorm:"size:50;default:'Inward'"` // Inward, Outward
	TargetClass     string    `gorm:"size:50"` // e.g. "All", "10-A" for circulars
	AssignedAt      time.Time `gorm:"default:now()"`
	ReferralOwnerID *uuid.UUID `gorm:"type:uuid"` // Nullable: stores original owner during refer/detour
	NotingSheet     string    `gorm:"type:text"` // Running commentaries
	DraftSpace      string    `gorm:"type:text"` // Drafted letters/orders
	CreatedAt       time.Time
	UpdatedAt       time.Time

	School       *School       `gorm:"foreignKey:SchoolID"`
	DocumentType *DocumentType `gorm:"foreignKey:DocumentTypeID"`
	Uploader     User          `gorm:"foreignKey:UploaderID"`
	CurrentOwner User          `gorm:"foreignKey:CurrentOwnerID"`
	Attachments  []Attachment  `gorm:"foreignKey:DocumentID"`
}

type Attachment struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	DocumentID uuid.UUID `gorm:"type:uuid;not null;index"`
	Filename   string    `gorm:"size:255;not null"`
	FilePath   string    `gorm:"size:1024;not null"`
	UploadedBy uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt  time.Time

	Document Document `gorm:"foreignKey:DocumentID"`
	Uploader User     `gorm:"foreignKey:UploadedBy"`
}

type DocumentPendingApprover struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	DocumentID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_doc_user_stage"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_doc_user_stage"`
	Stage      int       `gorm:"not null;uniqueIndex:idx_doc_user_stage"`
	Status     string    `gorm:"size:20;not null;default:'Pending'"` // Pending, Approved, Rejected, Skipped
	SignedAt   *time.Time
	CreatedAt  time.Time

	Document Document `gorm:"foreignKey:DocumentID"`
	User     User     `gorm:"foreignKey:UserID"`
}

type WorkflowAction string

const (
	ActionUploaded     WorkflowAction = "Uploaded"
	ActionSubmitted    WorkflowAction = "Submitted"
	ActionApproved     WorkflowAction = "Approved"
	ActionRejected     WorkflowAction = "Rejected"
	ActionSentBack     WorkflowAction = "Sent Back"
	ActionFileReplaced WorkflowAction = "File Replaced"
	ActionResubmitted  WorkflowAction = "Resubmitted"
	ActionForwarded    WorkflowAction = "Forwarded"
)

type WorkflowHistory struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key"`
	SchoolID   *uuid.UUID     `gorm:"type:uuid"`
	DocumentID uuid.UUID      `gorm:"type:uuid;not null"`
	ActorID    uuid.UUID      `gorm:"type:uuid;not null"`
	TargetID   *uuid.UUID     `gorm:"type:uuid"` // Nullable for end states
	Action     WorkflowAction `gorm:"size:50;not null"`
	Remarks    string         `gorm:"type:text"`
	Signature  string         `gorm:"type:text"` // Base64 signature image data URL
	ActorRole  string         `gorm:"size:50"`   // Snapshot of actor's role
	ActorIP    string         `gorm:"size:45"`   // IPv4/IPv6 address for audit
	Stage      int            `gorm:"default:1"`
	Version    int            `gorm:"default:1"`
	EventType  string         `gorm:"size:50"` // state_transition, escalation, viewed, sla_breach
	Metadata   string         `gorm:"type:text"`
	CreatedAt  time.Time

	School   *School  `gorm:"foreignKey:SchoolID"`
	Document Document `gorm:"foreignKey:DocumentID"`
	Actor    User     `gorm:"foreignKey:ActorID"`
	Target   *User    `gorm:"foreignKey:TargetID"`
}

func (base *School) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return
}

func (base *User) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return
}

func (base *DocumentType) BeforeCreate(tx *gorm.DB) (err error) {
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

func (base *DocumentPendingApprover) BeforeCreate(tx *gorm.DB) (err error) {
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
type Notification struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SchoolID    *uuid.UUID `gorm:"type:uuid"`
	RecipientID uuid.UUID  `gorm:"type:uuid;not null"`
	DocumentID  *uuid.UUID `gorm:"type:uuid"`
	Channel     string     `gorm:"size:20;not null"`  // email, in_app, sms
	Template    string     `gorm:"size:100;not null"` // action_required, approved, rejected, etc.
	Payload     string     `gorm:"type:text;not null"` // JSON structure
	Status      string     `gorm:"size:20;not null;default:'pending'"` // pending, sent, failed
	CreatedAt   time.Time
	SentAt      *time.Time

	School    *School   `gorm:"foreignKey:SchoolID"`
	Recipient User      `gorm:"foreignKey:RecipientID"`
	Document  *Document `gorm:"foreignKey:DocumentID"`
}

func (base *Notification) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return
}
