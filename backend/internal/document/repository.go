package document

import (
	"time"

	"office-file-sharing/backend/internal/shared/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)


type Repository interface {
	Create(doc *models.Document) error
	Save(doc *models.Document) error
	GetByID(id uuid.UUID) (*models.Document, error)
	ListByUser(userID uuid.UUID, search string) ([]models.Document, error)
	CreateHistory(history *models.WorkflowHistory) error
	GetHistoryByDocumentID(docID uuid.UUID) ([]models.WorkflowHistory, error)
	GetHistoryByFileID(fileID uuid.UUID) ([]models.WorkflowHistory, error)
	GetHistoryByUserID(userID uuid.UUID) ([]models.WorkflowHistory, error)
	CountSignatures(docID uuid.UUID) (int, error)
	GetDocumentTypeByID(id uuid.UUID) (*models.DocumentType, error)
	GetDocumentTypeBySlug(schoolID uuid.UUID, slug string) (*models.DocumentType, error)
	CreatePendingApprover(approver *models.DocumentPendingApprover) error
	GetPendingApprovers(docID uuid.UUID, stage int) ([]models.DocumentPendingApprover, error)
	GetPendingApproverByStage(docID uuid.UUID, stage int) (*models.DocumentPendingApprover, error)
	MarkApproverStatus(docID, userID uuid.UUID, stage int, status string) error
	GetSubmissionsByRefDocID(refDocID uuid.UUID) ([]models.Document, error)
	CreateFile(file *models.File) error
	SaveFile(file *models.File) error
	GetFileByID(id uuid.UUID) (*models.File, error)
	ListFilesByUser(userID uuid.UUID, search string) ([]models.File, error)
	CreateNote(note *models.Note) error
	SaveNote(note *models.Note) error
	GetNoteByID(id uuid.UUID) (*models.Note, error)
	GetNotesByFileID(fileID uuid.UUID) ([]models.Note, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(doc *models.Document) error {
	return r.db.Create(doc).Error
}

func (r *repository) Save(doc *models.Document) error {
	return r.db.Omit("Uploader", "CurrentOwner").Save(doc).Error
}

func (r *repository) GetByID(id uuid.UUID) (*models.Document, error) {
	var doc models.Document
	if err := r.db.Preload("Uploader").Preload("CurrentOwner").Preload("Attachments").First(&doc, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *repository) ListByUser(userID uuid.UUID, search string) ([]models.Document, error) {
	var user models.User
	if err := r.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}

	var documents []models.Document
	query := r.db.Preload("Uploader").Preload("CurrentOwner").Preload("Attachments")

	// Apply RBAC filters based on Greenwood High School roles
	switch user.Role {
	case "DHE":
		// DHE can see everything within the school
		if user.SchoolID != nil {
			query = query.Where("school_id = ?", *user.SchoolID)
		}

	case "School Admin":
		// School Admin can see:
		// 1. Documents they uploaded/own
		// 2. Documents where they are in history
		query = query.Where(
			"uploader_id = ? OR current_owner_id = ? OR id IN (SELECT document_id FROM workflow_histories WHERE actor_id = ?)",
			userID, userID, userID,
		)

	case "Teaching staff":
		// Teaching staff can see:
		// 1. Documents they uploaded/own
		// 2. Documents in their department uploaded by vocational staff
		// 3. Documents where they are in history
		if user.ClassSection != "" {
			query = query.Where(
				"uploader_id = ? OR current_owner_id = ? OR id IN (SELECT document_id FROM workflow_histories WHERE actor_id = ?) OR uploader_id IN (SELECT id FROM users WHERE class_section = ? AND role = 'vocational')",
				userID, userID, userID, user.ClassSection,
			)
		} else {
			query = query.Where(
				"uploader_id = ? OR current_owner_id = ? OR id IN (SELECT document_id FROM workflow_histories WHERE actor_id = ?)",
				userID, userID, userID,
			)
		}

	case "non-teaching":
		// non-teaching can see:
		// 1. Documents they own / pending review
		// 2. Documents where they are in history
		query = query.Where(
			"uploader_id = ? OR current_owner_id = ? OR id IN (SELECT document_id FROM workflow_histories WHERE actor_id = ?)",
			userID, userID, userID,
		)

	default: // vocational or other fallback
		// vocational can see:
		// 1. Documents they own
		// 2. Documents where they are in history
		query = query.Where(
			"uploader_id = ? OR current_owner_id = ? OR id IN (SELECT document_id FROM workflow_histories WHERE actor_id = ?)",
			userID, userID, userID,
		)
	}

	if search != "" {
		searchLike := "%" + search + "%"
		query = query.Where(
			"LOWER(title) LIKE LOWER(?) OR LOWER(description) LIKE LOWER(?) OR LOWER(unique_number) LIKE LOWER(?) OR LOWER(tags) LIKE LOWER(?) OR LOWER(category) LIKE LOWER(?)",
			searchLike, searchLike, searchLike, searchLike, searchLike,
		)
	}

	if err := query.Find(&documents).Error; err != nil {
		return nil, err
	}
	return documents, nil
}

func (r *repository) CreateHistory(history *models.WorkflowHistory) error {
	return r.db.Create(history).Error
}

func (r *repository) GetHistoryByDocumentID(docID uuid.UUID) ([]models.WorkflowHistory, error) {
	var history []models.WorkflowHistory
	err := r.db.Preload("Actor").Preload("Target").Where("document_id = ?", docID).Order("created_at asc").Find(&history).Error
	if err != nil {
		return nil, err
	}
	return history, nil
}

func (r *repository) GetHistoryByFileID(fileID uuid.UUID) ([]models.WorkflowHistory, error) {
	var history []models.WorkflowHistory
	err := r.db.Preload("Actor").Preload("Target").Where("file_id = ?", fileID).Order("created_at asc").Find(&history).Error
	if err != nil {
		return nil, err
	}
	return history, nil
}

func (r *repository) GetHistoryByUserID(userID uuid.UUID) ([]models.WorkflowHistory, error) {
	var history []models.WorkflowHistory
	err := r.db.Preload("Actor").Preload("Target").Preload("Document").Preload("Document.Uploader").Preload("Document.CurrentOwner").
		Where("actor_id = ? OR target_id = ?", userID, userID).
		Order("created_at desc").
		Find(&history).Error
	if err != nil {
		return nil, err
	}
	return history, nil
}

func (r *repository) CountSignatures(docID uuid.UUID) (int, error) {
	var count int64
	err := r.db.Model(&models.WorkflowHistory{}).Where("document_id = ? AND signature IS NOT NULL AND signature != ''", docID).Count(&count).Error
	return int(count), err
}

func (r *repository) GetDocumentTypeByID(id uuid.UUID) (*models.DocumentType, error) {
	var dt models.DocumentType
	err := r.db.First(&dt, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &dt, nil
}

func (r *repository) GetDocumentTypeBySlug(schoolID uuid.UUID, slug string) (*models.DocumentType, error) {
	var dt models.DocumentType
	err := r.db.First(&dt, "(school_id = ? OR school_id IS NULL) AND slug = ?", schoolID, slug).Error
	if err != nil {
		return nil, err
	}
	return &dt, nil
}

func (r *repository) CreatePendingApprover(approver *models.DocumentPendingApprover) error {
	return r.db.Create(approver).Error
}

func (r *repository) GetPendingApprovers(docID uuid.UUID, stage int) ([]models.DocumentPendingApprover, error) {
	var approvers []models.DocumentPendingApprover
	err := r.db.Preload("User").Where("document_id = ? AND stage = ?", docID, stage).Find(&approvers).Error
	if err != nil {
		return nil, err
	}
	return approvers, nil
}

func (r *repository) GetPendingApproverByStage(docID uuid.UUID, stage int) (*models.DocumentPendingApprover, error) {
	var approver models.DocumentPendingApprover
	err := r.db.Preload("User").Where("document_id = ? AND stage = ?", docID, stage).First(&approver).Error
	if err != nil {
		return nil, err
	}
	return &approver, nil
}

func (r *repository) GetSubmissionsByRefDocID(refDocID uuid.UUID) ([]models.Document, error) {
	var submissions []models.Document
	err := r.db.Preload("Uploader").Preload("CurrentOwner").Where("ref_document_id = ?", refDocID).Order("created_at desc").Find(&submissions).Error
	if err != nil {
		return nil, err
	}
	return submissions, nil
}

func (r *repository) MarkApproverStatus(docID, userID uuid.UUID, stage int, status string) error {
	now := time.Now()
	return r.db.Model(&models.DocumentPendingApprover{}).
		Where("document_id = ? AND user_id = ? AND stage = ?", docID, userID, stage).
		Updates(map[string]interface{}{
			"status":    status,
			"signed_at": &now,
		}).Error
}

func (r *repository) CreateFile(file *models.File) error {
	return r.db.Create(file).Error
}

func (r *repository) SaveFile(file *models.File) error {
	return r.db.Omit("Creator", "CurrentOwner", "Receipts").Save(file).Error
}

func (r *repository) GetFileByID(id uuid.UUID) (*models.File, error) {
	var file models.File
	err := r.db.Preload("Creator").Preload("CurrentOwner").Preload("Receipts").First(&file, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *repository) ListFilesByUser(userID uuid.UUID, search string) ([]models.File, error) {
	var files []models.File
	query := r.db.Preload("Creator").Preload("CurrentOwner").
		Order("created_at desc")

	// Filter: only show files created by the user, OR currently owned by the user, OR where they have written a Note
	query = query.Where(
		"creator_id = ? OR current_owner_id = ? OR id IN (SELECT file_id FROM notes WHERE author_id = ?)",
		userID, userID, userID,
	)

	if search != "" {
		query = query.Where("title ILIKE ? OR file_number ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	err := query.Find(&files).Error
	return files, err
}

func (r *repository) CreateNote(note *models.Note) error {
	return r.db.Create(note).Error
}

func (r *repository) SaveNote(note *models.Note) error {
	return r.db.Omit("File", "Author").Save(note).Error
}

func (r *repository) GetNoteByID(id uuid.UUID) (*models.Note, error) {
	var note models.Note
	err := r.db.Preload("Author").First(&note, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &note, nil
}

func (r *repository) GetNotesByFileID(fileID uuid.UUID) ([]models.Note, error) {
	var notes []models.Note
	err := r.db.Preload("Author").Preload("Author.School").Where("file_id = ? AND is_discarded = false", fileID).Order("created_at asc").Find(&notes).Error
	return notes, err
}
