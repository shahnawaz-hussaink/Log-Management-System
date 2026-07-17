package user

import (
	"office-file-sharing/backend/internal/shared/models"
	"gorm.io/gorm"
)

type Repository interface {
	GetAll() ([]models.User, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetAll() ([]models.User, error) {
	var users []models.User
	if err := r.db.Preload("School").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
