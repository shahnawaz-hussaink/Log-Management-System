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
	ActiveState string  `gorm:"size:50;default:'active';not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SchoolID     *uuid.UUID `gorm:"type:uuid"`
	Name         string     `gorm:"size:255;not null"`
	Email        string     `gorm:"size:255;not null;unique"`
	PasswordHash string     `gorm:"not null"`
	Role         string     `gorm:"size:50;not null;default:'vocational'"` // Office roles: vocational, non-teaching, Teaching staff, School Admin, DHE
	ClassSection string     `gorm:"size:50"`                               // Scoped access for Teaching staff/vocational (e.g. "Department A")
	Subject      string     `gorm:"size:100"`                           // Assigned subject for Teacher
	Phone        string     `gorm:"size:20"`                            // For notifications
	Avatar       string     `gorm:"type:text"`                          // Profile picture base64 string
	ActiveState  string     `gorm:"size:50;default:'active';not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time

	School *School `gorm:"foreignKey:SchoolID"`
}

// Removed ParentChild model as per new employee-only requirements

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
	SchoolID          *uuid.UUID `gorm:"type:uuid"`
	Name              string     `gorm:"size:100;not null"`
	Slug              string     `gorm:"size:100;not null"`
	WorkflowStages    string     `gorm:"type:text;not null"` // JSON array of stages
	RequiredFields    string     `gorm:"type:text"`          // JSON array of dynamic fields
	Active            bool       `gorm:"default:true"`
	CreatorRoleID     *uuid.UUID `gorm:"type:uuid"`
	CreatedAt         time.Time
	UpdatedAt         time.Time

	School      *School `gorm:"foreignKey:SchoolID"`
	CreatorRole *Role   `gorm:"foreignKey:CreatorRoleID"`
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
	CurrentStage    int            `gorm:"not null;default:1"`
	Metadata        string    `gorm:"type:text"` // JSON object containing form fields
	Priority        string    `gorm:"size:50;default:'Normal'"` // Normal, Urgent, Confidential
	Direction       string    `gorm:"size:50;default:'Inward'"` // Inward, Outward
	TargetClass     string    `gorm:"size:255"`                 // e.g. "All", "10-A,10-B" for circulars/broadcasts
	AssignedAt      time.Time `gorm:"default:now()"`
	ReferralOwnerID *uuid.UUID `gorm:"type:uuid"` // Nullable: stores original owner during refer/detour
	NotingSheet     string    `gorm:"type:text"` // Running commentaries
	DraftSpace      string    `gorm:"type:text"` // Drafted letters/orders
	FileID          *uuid.UUID `gorm:"type:uuid;index"` // Associated file (if attached)
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
	DocumentID *uuid.UUID     `gorm:"type:uuid"` // Nullable for files
	FileID     *uuid.UUID     `gorm:"type:uuid"` // Nullable for documents
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
	Document Document `gorm:"foreignKey:DocumentID;constraint:-"`
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

type FileStatus string
const (
	FileStatusOpen     FileStatus = "Open"
	FileStatusInReview FileStatus = "In Review"
	FileStatusClosed   FileStatus = "Closed"
	FileStatusArchived FileStatus = "Archived"
)

type File struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SchoolID       *uuid.UUID `gorm:"type:uuid"`
	FileNumber     string     `gorm:"size:100;uniqueIndex"` // e.g. EDU/2026/0001
	Title          string     `gorm:"size:255;not null"`
	Description    string     `gorm:"type:text"`
	Category       string     `gorm:"size:100;not null;default:'General'"`
	SubCategory    string     `gorm:"size:100;not null;default:'Misc'"`
	CreatorID      uuid.UUID  `gorm:"type:uuid;not null"`
	CurrentOwnerID uuid.UUID  `gorm:"type:uuid;not null"` // Who currently acts on the file
	Status         FileStatus `gorm:"size:50;not null;default:'Open'"`
	Priority       string     `gorm:"size:50;default:'Normal'"`
	ArchivedByID   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	School       *School    `gorm:"foreignKey:SchoolID"`
	Creator      User       `gorm:"foreignKey:CreatorID"`
	CurrentOwner User       `gorm:"foreignKey:CurrentOwnerID"`
	ArchivedBy   *User      `gorm:"foreignKey:ArchivedByID"`
	Receipts     []Document `gorm:"foreignKey:FileID"`
}

type NoteType string
const (
	NoteTypeGreen  NoteType = "Green"
	NoteTypeYellow NoteType = "Yellow"
)

type Note struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	FileID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	AuthorID        uuid.UUID  `gorm:"type:uuid;not null"`
	Type            NoteType   `gorm:"size:20;not null"`
	Content         string     `gorm:"type:text;not null"`
	Signature       string     `gorm:"type:text"` // Digital signature token snapshot
	IsDiscarded     bool       `gorm:"default:false"`
	PublishedFromID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt       time.Time
	UpdatedAt       time.Time

	File   File `gorm:"foreignKey:FileID"`
	Author User `gorm:"foreignKey:AuthorID"`
}

func (base *File) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return
}

func (base *Note) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return
}

type Role struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RoleName      string     `gorm:"size:100;not null;uniqueIndex:idx_role_name_tenant"`
	IsAdminAccess bool       `gorm:"default:false;not null"`
	ParentRoleID  *uuid.UUID `gorm:"type:uuid"`
	TenantID      *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_role_name_tenant"`
	CreatedBy     string     `gorm:"size:255;not null"`
	Path          string     `gorm:"type:text;not null"`

	ParentRole *Role   `gorm:"foreignKey:ParentRoleID"`
	Tenant     *School `gorm:"foreignKey:TenantID"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (base *Role) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return
}

type Organization struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationName string     `gorm:"size:255;not null"`
	Type             string     `gorm:"size:100;not null"` // e.g. "school"
	ParentOrgID      *uuid.UUID `gorm:"type:uuid"`
	PointOfContactID *uuid.UUID `gorm:"type:uuid"` // Links to User (Admin)
	CreatedBy        string     `gorm:"size:255;not null"` // SuperAdmin
	TenantID         *uuid.UUID `gorm:"type:uuid"` // tenantId if applicable
	ActiveState      string     `gorm:"size:50;default:'active';not null"`
	CreatedAt        time.Time
	UpdatedAt        time.Time

	ParentOrg      *Organization `gorm:"foreignKey:ParentOrgID"`
	PointOfContact *User         `gorm:"foreignKey:PointOfContactID"`
}

func (base *Organization) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return
}

type PeerConnection struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SenderRoleID uuid.UUID `gorm:"type:uuid;not null;index"`
	TargetRoleID uuid.UUID `gorm:"type:uuid;not null;index"`
	Status       string    `gorm:"size:50;not null;default:'pending'"` // pending, accepted, rejected, revoked
	CreatedAt    time.Time
	UpdatedAt    time.Time

	SenderRole *Role `gorm:"foreignKey:SenderRoleID"`
	TargetRole *Role `gorm:"foreignKey:TargetRoleID"`
}

func (base *PeerConnection) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return
}
