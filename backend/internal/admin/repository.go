package admin

import (
	"office-file-sharing/backend/internal/shared/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	GetStats(schoolID *string) (*SystemStats, error)
	GetAllUsers(schoolID *string) ([]UserResponse, error)
	GetUserByID(id uuid.UUID) (*models.User, error)
	CreateUser(user *models.User) error
	UpdateUser(user *models.User) error
	DeleteUser(id uuid.UUID) error
	GetAllDocumentTypes(schoolID *string) ([]DocumentTypeResponse, error)
	CreateDocumentType(dt *models.DocumentType) error
	GetDocumentTypeByID(id uuid.UUID) (*models.DocumentType, error)
	UpdateDocumentType(dt *models.DocumentType) error
	DeleteDocumentType(id uuid.UUID) error
	GetAllSchools(schoolID *string) ([]SchoolResponse, error)
	GetSchoolByID(id uuid.UUID) (*models.School, error)
	UpdateSchool(school *models.School) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetStats(schoolID *string) (*SystemStats, error) {
	var stats SystemStats

	userQuery := r.db.Model(&models.User{})
	docQuery := r.db.Model(&models.Document{})
	dtQuery := r.db.Model(&models.DocumentType{})
	schoolQuery := r.db.Model(&models.School{})

	if schoolID != nil {
		userQuery = userQuery.Where("school_id = ?", *schoolID)
		docQuery = docQuery.Where("school_id = ?", *schoolID)
		dtQuery = dtQuery.Where("school_id = ?", *schoolID)
		schoolQuery = schoolQuery.Where("id = ?", *schoolID)
	}

	userQuery.Count(&stats.TotalUsers)
	docQuery.Count(&stats.TotalDocuments)
	schoolQuery.Count(&stats.TotalSchools)
	dtQuery.Count(&stats.TotalDocumentTypes)

	docQueryPending := r.db.Model(&models.Document{}).Where("status = ?", models.StatusPendingApproval)
	if schoolID != nil {
		docQueryPending = docQueryPending.Where("school_id = ?", *schoolID)
	}
	docQueryPending.Count(&stats.PendingDocuments)

	docQueryApproved := r.db.Model(&models.Document{}).Where("status = ?", models.StatusApproved)
	if schoolID != nil {
		docQueryApproved = docQueryApproved.Where("school_id = ?", *schoolID)
	}
	docQueryApproved.Count(&stats.ApprovedDocuments)

	docQueryActive := r.db.Model(&models.Document{}).Where("status NOT IN ?", []string{string(models.StatusClosed), string(models.StatusArchived), string(models.StatusRejected)})
	if schoolID != nil {
		docQueryActive = docQueryActive.Where("school_id = ?", *schoolID)
	}
	docQueryActive.Count(&stats.ActiveDocuments)
	
	stats.SLABreaches = 0

	return &stats, nil
}

func (r *repository) GetAllUsers(schoolID *string) ([]UserResponse, error) {
	var users []models.User
	query := r.db.Preload("School").Order("created_at desc")
	if schoolID != nil {
		query = query.Where("school_id = ?", *schoolID)
	}
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}

	var resp []UserResponse
	for _, u := range users {
		schoolName := ""
		if u.School != nil {
			schoolName = u.School.Name
		}
		resp = append(resp, UserResponse{
			ID:           u.ID,
			Name:         u.Name,
			Email:        u.Email,
			Role:         u.Role,
			SchoolID:     u.SchoolID,
			SchoolName:   schoolName,
			ClassSection: u.ClassSection,
			Subject:      u.Subject,
			Phone:        u.Phone,
			CreatedAt:    u.CreatedAt,
		})
	}
	return resp, nil
}

func (r *repository) GetUserByID(id uuid.UUID) (*models.User, error) {
	var u models.User
	if err := r.db.First(&u, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *repository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *repository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *repository) DeleteUser(id uuid.UUID) error {
	return r.db.Delete(&models.User{}, "id = ?", id).Error
}

func (r *repository) GetAllDocumentTypes(schoolID *string) ([]DocumentTypeResponse, error) {
	var docTypes []models.DocumentType
	query := r.db.Preload("School").Order("name asc")
	if schoolID != nil {
		query = query.Where("school_id = ?", *schoolID)
	}
	if err := query.Find(&docTypes).Error; err != nil {
		return nil, err
	}

	var resp []DocumentTypeResponse
	for _, dt := range docTypes {
		resp = append(resp, DocumentTypeResponse{
			ID:                dt.ID,
			SchoolID:          dt.SchoolID,
			SchoolName:        dt.School.Name,
			Name:              dt.Name,
			Slug:              dt.Slug,
			WorkflowStages:    dt.WorkflowStages,
			RequiredFields:    dt.RequiredFields,
			SlaHours:          0,
			Active:            dt.Active,
		})
	}
	return resp, nil
}

func (r *repository) CreateDocumentType(dt *models.DocumentType) error {
	return r.db.Create(dt).Error
}

func (r *repository) GetDocumentTypeByID(id uuid.UUID) (*models.DocumentType, error) {
	var dt models.DocumentType
	if err := r.db.First(&dt, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &dt, nil
}

func (r *repository) UpdateDocumentType(dt *models.DocumentType) error {
	return r.db.Save(dt).Error
}

func (r *repository) DeleteDocumentType(id uuid.UUID) error {
	return r.db.Delete(&models.DocumentType{}, "id = ?", id).Error
}

func (r *repository) GetAllSchools(schoolID *string) ([]SchoolResponse, error) {
	var schools []models.School
	query := r.db.Order("name asc")
	if schoolID != nil {
		query = query.Where("id = ?", *schoolID)
	}
	if err := query.Find(&schools).Error; err != nil {
		return nil, err
	}

	var resp []SchoolResponse
	for _, s := range schools {
		resp = append(resp, SchoolResponse{
			ID:        s.ID,
			Name:      s.Name,
			Slug:      s.Slug,
			Settings:  s.Settings,
			CreatedAt: s.CreatedAt,
		})
	}
	return resp, nil
}

func (r *repository) GetSchoolByID(id uuid.UUID) (*models.School, error) {
	var s models.School
	if err := r.db.First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) UpdateSchool(school *models.School) error {
	return r.db.Save(school).Error
}
