package user

import (
	"github.com/google/uuid"
	"office-file-sharing/backend/internal/admin"
	"office-file-sharing/backend/internal/shared/models"
)

type Service interface {
	GetUsers(actorID uuid.UUID) ([]UserResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetUsers(actorID uuid.UUID) ([]UserResponse, error) {
	// Find actor profile
	var actor models.User
	db := s.repo.(*repository).db
	err := db.First(&actor, "id = ?", actorID).Error
	if err != nil {
		return nil, err
	}

	allUsers, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}

	var filtered []models.User
	for _, u := range allUsers {
		if u.ID == actorID {
			continue // skip self
		}
		// DHE or SuperAdmin or Admin sees everyone
		if admin.HasAdminAccess(db, actor.Role, actor.SchoolID) {
			filtered = append(filtered, u)
		} else if admin.HasRole(db, actor.Role, "School Admin", actor.SchoolID) {
			// School Admin sees:
			// 1. Everyone in their own school
			// 2. DHE / System Admin users (to escalate/forward system files)
			// 3. Other School Admins (to forward/share documents across schools)
			if (u.SchoolID != nil && actor.SchoolID != nil && *u.SchoolID == *actor.SchoolID) ||
				admin.HasRole(db, u.Role, "DHE", u.SchoolID) || admin.HasRole(db, u.Role, "School Admin", u.SchoolID) {
				filtered = append(filtered, u)
			}
		} else {
			// Everyone else sees people in their own school/office
			if u.SchoolID != nil && actor.SchoolID != nil && *u.SchoolID == *actor.SchoolID {
				filtered = append(filtered, u)
			}
		}
	}

	responses := make([]UserResponse, len(filtered))
	for i, u := range filtered {
		responses[i] = UserResponse{
			ID:        u.ID,
			Name:      u.Name,
			Email:     u.Email,
			Role:      u.Role,
			SchoolID:  u.SchoolID,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
			School:    u.School,
		}
	}
	return responses, nil
}
