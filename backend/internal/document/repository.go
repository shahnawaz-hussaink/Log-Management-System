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
	GetHistoryByUserID(userID uuid.UUID) ([]models.WorkflowHistory, error)
	CountSignatures(docID uuid.UUID) (int, error)
	GetDocumentTypeByID(id uuid.UUID) (*models.DocumentType, error)
	GetDocumentTypeBySlug(schoolID uuid.UUID, slug string) (*models.DocumentType, error)
	CreatePendingApprover(approver *models.DocumentPendingApprover) error
	GetPendingApprovers(docID uuid.UUID, stage int) ([]models.DocumentPendingApprover, error)
	MarkApproverStatus(docID, userID uuid.UUID, stage int, status string) error
	GetParentByStudent(studentID uuid.UUID) (*models.User, error)
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
	case "Admin":
		// Admin can see everything within the school
		if user.SchoolID != nil {
			query = query.Where("school_id = ?", *user.SchoolID)
		}

	case "Principal":
		// Principal can see:
		// 1. Documents they uploaded/own
		// 2. Documents where they are in history
		// 3. All Circulars
		query = query.Where(
			"uploader_id = ? OR current_owner_id = ? OR id IN (SELECT document_id FROM workflow_histories WHERE actor_id = ?) OR category = 'Circular'",
			userID, userID, userID,
		)

	case "Teacher":
		// Teacher can see:
		// 1. Documents they uploaded/own
		// 2. Documents in their class/section uploaded by students
		// 3. Documents where they are in history
		// 4. Circulars targeted at their class or All
		if user.ClassSection != "" {
			query = query.Where(
				"uploader_id = ? OR current_owner_id = ? OR id IN (SELECT document_id FROM workflow_histories WHERE actor_id = ?) OR uploader_id IN (SELECT id FROM users WHERE class_section = ? AND role = 'Student') OR (category = 'Circular' AND (target_class = 'All' OR target_class = ?))",
				userID, userID, userID, user.ClassSection, user.ClassSection,
			)
		} else {
			query = query.Where(
				"uploader_id = ? OR current_owner_id = ? OR id IN (SELECT document_id FROM workflow_histories WHERE actor_id = ?) OR (category = 'Circular' AND target_class = 'All')",
				userID, userID, userID,
			)
		}

	case "Parent":
		// Parent can see:
		// 1. Documents uploaded by their children
		// 2. Documents they own / pending review
		// 3. Circulars targeted at their children's classes or All
		query = query.Where(
			"uploader_id IN (SELECT child_id FROM parent_children WHERE parent_id = ?) OR current_owner_id = ? OR (category = 'Circular' AND (target_class = 'All' OR target_class IN (SELECT class_section FROM users WHERE id IN (SELECT child_id FROM parent_children WHERE parent_id = ?))))",
			userID, userID, userID,
		)

	default: // Student or other fallback
		// Student can only see their own submissions + relevant Circulars
		query = query.Where(
			"uploader_id = ? OR current_owner_id = ? OR (category = 'Circular' AND (target_class = 'All' OR target_class = ?))",
			userID, userID, user.ClassSection,
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
	err := r.db.First(&dt, "school_id = ? AND slug = ?", schoolID, slug).Error
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

func (r *repository) MarkApproverStatus(docID, userID uuid.UUID, stage int, status string) error {
	now := time.Now()
	return r.db.Model(&models.DocumentPendingApprover{}).
		Where("document_id = ? AND user_id = ? AND stage = ?", docID, userID, stage).
		Updates(map[string]interface{}{
			"status":    status,
			"signed_at": &now,
		}).Error
}

func (r *repository) GetParentByStudent(studentID uuid.UUID) (*models.User, error) {
	var parent models.User
	err := r.db.Table("users").
		Joins("JOIN parent_children ON parent_children.parent_id = users.id").
		Where("parent_children.child_id = ?", studentID).
		First(&parent).Error
	if err != nil {
		return nil, err
	}
	return &parent, nil
}

