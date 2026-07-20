package admin

import (
	"errors"
	"office-file-sharing/backend/internal/shared/models"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	GetStats(schoolID *string) (*SystemStats, error)
	GetAllUsers(schoolID *string) ([]UserResponse, error)
	CreateUser(req CreateUserRequest, actorRole string, actorSchoolID *uuid.UUID) (*UserResponse, error)
	UpdateUser(id uuid.UUID, req UpdateUserRequest, actorRole string, actorSchoolID *uuid.UUID) (*UserResponse, error)
	DeleteUser(id uuid.UUID) error
	GetAllDocumentTypes(schoolID *string) ([]DocumentTypeResponse, error)
	CreateDocumentType(req CreateDocTypeRequest) (*DocumentTypeResponse, error)
	UpdateDocumentType(id uuid.UUID, req UpdateDocTypeRequest) (*DocumentTypeResponse, error)
	DeleteDocumentType(id uuid.UUID) error
	GetAllSchools(schoolID *string) ([]SchoolResponse, error)
	UpdateSchool(id uuid.UUID, req UpdateSchoolRequest) (*SchoolResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetStats(schoolID *string) (*SystemStats, error) {
	return s.repo.GetStats(schoolID)
}

func (s *service) GetAllUsers(schoolID *string) ([]UserResponse, error) {
	return s.repo.GetAllUsers(schoolID)
}

func (s *service) CreateUser(req CreateUserRequest, actorRole string, actorSchoolID *uuid.UUID) (*UserResponse, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" {
		return nil, errors.New("name and email are required")
	}
	if req.Password == "" {
		req.Password = "password" // default password
	}

	var targetSchoolID *uuid.UUID
	if actorRole == "School Admin" {
		if req.Role == "DHE" || req.Role == "SuperAdmin" || req.Role == "Admin" {
			return nil, errors.New("school admin cannot assign administrative roles")
		}
		if actorSchoolID == nil {
			return nil, errors.New("school admin must belong to a school")
		}
		targetSchoolID = actorSchoolID
	} else {
		targetSchoolID = req.SchoolID
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := &models.User{
		ID:           uuid.New(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         req.Role,
		SchoolID:     targetSchoolID,
		ClassSection: req.ClassSection,
		Subject:      req.Subject,
		Phone:        req.Phone,
	}

	if err := s.repo.CreateUser(u); err != nil {
		return nil, err
	}

	resp := &UserResponse{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		Role:         u.Role,
		SchoolID:     u.SchoolID,
		ClassSection: u.ClassSection,
		Subject:      u.Subject,
		Phone:        u.Phone,
		CreatedAt:    u.CreatedAt,
	}
	return resp, nil
}

func (s *service) UpdateUser(id uuid.UUID, req UpdateUserRequest, actorRole string, actorSchoolID *uuid.UUID) (*UserResponse, error) {
	u, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// If changing role or email, block if user is referenced in document workflows or logs
	roleChanged := req.Role != "" && req.Role != u.Role
	emailChanged := req.Email != "" && req.Email != u.Email
	if roleChanged || emailChanged {
		repoImpl, ok := s.repo.(*repository)
		if ok {
			var count int64
			repoImpl.db.Model(&models.Document{}).
				Where("uploader_id = ? OR current_owner_id = ? OR referral_owner_id = ?", id, id, id).
				Count(&count)
			if count > 0 {
				return nil, errors.New("cannot change role or email of a user with active or historical documents")
			}

			repoImpl.db.Model(&models.WorkflowHistory{}).
				Where("actor_id = ? OR target_id = ?", id, id).
				Count(&count)
			if count > 0 {
				return nil, errors.New("cannot change role or email of a user with workflow history logs")
			}
		}
	}

	if req.Name != "" {
		u.Name = req.Name
	}
	if req.Email != "" {
		u.Email = req.Email
	}
	if req.Role != "" {
		if actorRole == "School Admin" {
			if req.Role == "DHE" || req.Role == "SuperAdmin" || req.Role == "Admin" {
				return nil, errors.New("school admin cannot assign administrative roles")
			}
		}
		u.Role = req.Role
	}

	// Enforce school restrictions for School Admin / DHE
	if actorRole == "School Admin" {
		if actorSchoolID == nil {
			return nil, errors.New("school admin must belong to a school")
		}
		if u.SchoolID == nil || *u.SchoolID != *actorSchoolID {
			return nil, errors.New("you are not authorized to update users outside your school")
		}
		u.SchoolID = actorSchoolID
	} else {
		// DHE/SuperAdmin can change the school of any user (including changing it to nil/None)
		u.SchoolID = req.SchoolID
	}
	u.ClassSection = req.ClassSection
	u.Subject = req.Subject
	u.Phone = req.Phone

	// Update password only if provided
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		u.PasswordHash = string(hash)
	}

	if err := s.repo.UpdateUser(u); err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		Role:         u.Role,
		SchoolID:     u.SchoolID,
		ClassSection: u.ClassSection,
		Subject:      u.Subject,
		Phone:        u.Phone,
		CreatedAt:    u.CreatedAt,
	}, nil
}

func (s *service) DeleteUser(id uuid.UUID) error {
	repoImpl, ok := s.repo.(*repository)
	if !ok {
		return errors.New("invalid repository type")
	}

	var count int64

	// 1. Check Document table
	repoImpl.db.Model(&models.Document{}).
		Where("uploader_id = ? OR current_owner_id = ? OR referral_owner_id = ?", id, id, id).
		Count(&count)
	if count > 0 {
		return errors.New("cannot delete user: they have active or historical documents")
	}

	// 2. Check WorkflowHistory table
	repoImpl.db.Model(&models.WorkflowHistory{}).
		Where("actor_id = ? OR target_id = ?", id, id).
		Count(&count)
	if count > 0 {
		return errors.New("cannot delete user: they are referenced in workflow history logs")
	}

	// 3. Check DocumentPendingApprover table
	repoImpl.db.Model(&models.DocumentPendingApprover{}).
		Where("user_id = ?", id).
		Count(&count)
	if count > 0 {
		return errors.New("cannot delete user: they are a pending workflow approver")
	}

	// 4. Check Attachment table
	repoImpl.db.Model(&models.Attachment{}).
		Where("uploaded_by = ?", id).
		Count(&count)
	if count > 0 {
		return errors.New("cannot delete user: they uploaded files enclosed in documents")
	}

	return s.repo.DeleteUser(id)
}

func (s *service) GetAllDocumentTypes(schoolID *string) ([]DocumentTypeResponse, error) {
	return s.repo.GetAllDocumentTypes(schoolID)
}

func (s *service) CreateDocumentType(req CreateDocTypeRequest) (*DocumentTypeResponse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("document type name is required")
	}
	if req.WorkflowStages == "" {
		req.WorkflowStages = "[]"
	}
	if req.RequiredFields == "" {
		req.RequiredFields = "[]"
	}
	dt := &models.DocumentType{
		ID:             uuid.New(),
		SchoolID:       req.SchoolID,
		Name:           req.Name,
		Slug:           req.Slug,
		WorkflowStages: req.WorkflowStages,
		RequiredFields: req.RequiredFields,
		Active:         true,
	}

	if err := s.repo.CreateDocumentType(dt); err != nil {
		return nil, err
	}

	return &DocumentTypeResponse{
		ID:             dt.ID,
		SchoolID:       dt.SchoolID,
		Name:           dt.Name,
		Slug:           dt.Slug,
		WorkflowStages: dt.WorkflowStages,
		RequiredFields: dt.RequiredFields,
		SlaHours:       0,
		Active:         dt.Active,
	}, nil
}

func (s *service) UpdateDocumentType(id uuid.UUID, req UpdateDocTypeRequest) (*DocumentTypeResponse, error) {
	dt, err := s.repo.GetDocumentTypeByID(id)
	if err != nil {
		return nil, errors.New("document type not found")
	}

	// If changing stages/fields, block if any active documents use this type
	if req.WorkflowStages != "" || req.RequiredFields != "" {
		repoImpl, ok := s.repo.(*repository)
		if ok {
			var activeCount int64
			repoImpl.db.Model(&models.Document{}).
				Where("document_type_id = ? AND status NOT IN ?", id, []string{string(models.StatusClosed), string(models.StatusArchived), string(models.StatusRejected)}).
				Count(&activeCount)
			if activeCount > 0 {
				return nil, errors.New("cannot edit workflow stages or required fields: active documents exist of this type")
			}
		}
	}

	if req.Name != "" {
		dt.Name = req.Name
	}
	if req.Slug != "" {
		dt.Slug = req.Slug
	}
	if req.WorkflowStages != "" {
		dt.WorkflowStages = req.WorkflowStages
	}
	if req.RequiredFields != "" {
		dt.RequiredFields = req.RequiredFields
	}
	dt.Active = req.Active

	if err := s.repo.UpdateDocumentType(dt); err != nil {
		return nil, err
	}

	return &DocumentTypeResponse{
		ID:                dt.ID,
		SchoolID:          dt.SchoolID,
		Name:              dt.Name,
		Slug:              dt.Slug,
		WorkflowStages:    dt.WorkflowStages,
		RequiredFields:    dt.RequiredFields,
		SlaHours:          0,
		Active:            dt.Active,
	}, nil
}

func (s *service) DeleteDocumentType(id uuid.UUID) error {
	repoImpl, ok := s.repo.(*repository)
	if !ok {
		return errors.New("invalid repository type")
	}

	var count int64
	repoImpl.db.Model(&models.Document{}).
		Where("document_type_id = ?", id).
		Count(&count)
	if count > 0 {
		return errors.New("cannot delete document type: it is referenced by existing documents")
	}

	return s.repo.DeleteDocumentType(id)
}

func (s *service) GetAllSchools(schoolID *string) ([]SchoolResponse, error) {
	return s.repo.GetAllSchools(schoolID)
}

func (s *service) UpdateSchool(id uuid.UUID, req UpdateSchoolRequest) (*SchoolResponse, error) {
	school, err := s.repo.GetSchoolByID(id)
	if err != nil {
		return nil, errors.New("school not found")
	}

	if req.Name != "" {
		school.Name = req.Name
	}
	if req.Slug != "" {
		school.Slug = req.Slug
	}
	if req.Settings != "" {
		school.Settings = req.Settings
	}

	if err := s.repo.UpdateSchool(school); err != nil {
		return nil, err
	}

	return &SchoolResponse{
		ID:        school.ID,
		Name:      school.Name,
		Slug:      school.Slug,
		Settings:  school.Settings,
		CreatedAt: school.CreatedAt,
	}, nil
}
