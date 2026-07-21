package auth

import (
	"office-file-sharing/backend/internal/shared/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	GetByEmail(email string) (*models.User, error)
	Create(user *models.User) error
	CheckAdminAccess(roleName string, schoolID *uuid.UUID) bool
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetByEmail(email string) (*models.User, error) {
	var u models.User
	if err := r.db.Preload("School").First(&u, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *repository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *repository) CheckAdminAccess(roleName string, schoolID *uuid.UUID) bool {
	if roleName == "SuperAdmin" || roleName == "Admin" || roleName == "DHE" || roleName == "School Admin" {
		return true
	}

	var role models.Role
	var err error
	if schoolID != nil {
		err = r.db.First(&role, "role_name = ? AND tenant_id = ?", roleName, *schoolID).Error
	}
	if schoolID == nil || err != nil {
		err = r.db.First(&role, "role_name = ? AND tenant_id IS NULL", roleName).Error
	}
	if err != nil {
		err = r.db.First(&role, "role_name = ?", roleName).Error
	}
	if err != nil {
		return false
	}

	curr := &role
	for curr != nil {
		if curr.IsAdminAccess {
			return true
		}
		if curr.ParentRoleID == nil {
			break
		}
		var parent models.Role
		if err := r.db.First(&parent, "id = ?", *curr.ParentRoleID).Error; err != nil {
			break
		}
		curr = &parent
	}
	return false
}
