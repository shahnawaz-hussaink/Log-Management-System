package document

import (
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
	CountSignatures(docID uuid.UUID) (int, error)
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
	return r.db.Save(doc).Error
}

func (r *repository) GetByID(id uuid.UUID) (*models.Document, error) {
	var doc models.Document
	if err := r.db.Preload("Uploader").Preload("CurrentOwner").First(&doc, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *repository) ListByUser(userID uuid.UUID, search string) ([]models.Document, error) {
	var documents []models.Document
	
	// Securely query documents where user is uploader, current owner, OR has interacted with it in workflow history
	query := r.db.Preload("Uploader").Preload("CurrentOwner").
		Where("uploader_id = ? OR current_owner_id = ? OR id IN (SELECT document_id FROM workflow_histories WHERE actor_id = ?)", userID, userID, userID)

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

func (r *repository) CountSignatures(docID uuid.UUID) (int, error) {
	var count int64
	err := r.db.Model(&models.WorkflowHistory{}).Where("document_id = ? AND signature IS NOT NULL AND signature != ''", docID).Count(&count).Error
	return int(count), err
}
