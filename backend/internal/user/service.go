package user

import (
	"github.com/google/uuid"
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

	// Load admin roles map
	var adminRoles []models.Role
	db.Where("is_admin_access = ?", true).Find(&adminRoles)
	adminRolesMap := make(map[string]bool)
	for _, r := range adminRoles {
		adminRolesMap[r.RoleName] = true
	}

	isAdminRole := func(roleName string) bool {
		if roleName == "SuperAdmin" || roleName == "Admin" || roleName == "DHE" || roleName == "School Admin" {
			return true
		}
		return adminRolesMap[roleName]
	}

	// Resolve actor's organization hierarchy
	var actorOrg models.Organization
	var siblingTenantIDs []uuid.UUID
	var parentTenantID *uuid.UUID
	var childTenantIDs []uuid.UUID

	if actor.SchoolID != nil {
		if err := db.First(&actorOrg, "tenant_id = ?", *actor.SchoolID).Error; err == nil {
			// Sibling and Parent Tenant IDs
			if actorOrg.ParentOrgID != nil {
				var siblingOrgs []models.Organization
				db.Find(&siblingOrgs, "parent_org_id = ? AND id != ?", *actorOrg.ParentOrgID, actorOrg.ID)
				for _, o := range siblingOrgs {
					if o.TenantID != nil {
						siblingTenantIDs = append(siblingTenantIDs, *o.TenantID)
					}
				}

				var parentOrg models.Organization
				if err := db.First(&parentOrg, "id = ?", *actorOrg.ParentOrgID).Error; err == nil {
					parentTenantID = parentOrg.TenantID
				}
			}

			// Child Tenant IDs
			var childOrgs []models.Organization
			db.Find(&childOrgs, "parent_org_id = ?", actorOrg.ID)
			for _, o := range childOrgs {
				if o.TenantID != nil {
					childTenantIDs = append(childTenantIDs, *o.TenantID)
				}
			}
		}
	}

	var filtered []models.User
	for _, u := range allUsers {
		if u.ID == actorID {
			continue // skip self
		}

		// SuperAdmin / Admin can see everyone
		if actor.Role == "SuperAdmin" || actor.Role == "Admin" {
			filtered = append(filtered, u)
			continue
		}

		// Target user is a System-level Admin (SuperAdmin / Admin), they should be visible to any admin
		if (u.Role == "SuperAdmin" || u.Role == "Admin") && isAdminRole(actor.Role) {
			filtered = append(filtered, u)
			continue
		}

		// Everyone can see colleagues in their own organization
		if u.SchoolID != nil && actor.SchoolID != nil && *u.SchoolID == *actor.SchoolID {
			filtered = append(filtered, u)
			continue
		}

		// If actor is an Admin of their organization (point of contact), they can see:
		// 1. Sibling organization admins (same level)
		// 2. Parent organization admins (upstream)
		// 3. Child organization admins (downstream)
		if actor.SchoolID != nil && isAdminRole(actor.Role) {
			isTargetAdmin := isAdminRole(u.Role)
			if isTargetAdmin {
				// Sibling check
				if len(siblingTenantIDs) > 0 && u.SchoolID != nil {
					isSibling := false
					for _, tid := range siblingTenantIDs {
						if *u.SchoolID == tid {
							isSibling = true
							break
						}
					}
					if isSibling {
						filtered = append(filtered, u)
						continue
					}
				}

				// Parent check
				if parentTenantID != nil && u.SchoolID != nil && *u.SchoolID == *parentTenantID {
					filtered = append(filtered, u)
					continue
				}

				// Child check
				if len(childTenantIDs) > 0 && u.SchoolID != nil {
					isChild := false
					for _, tid := range childTenantIDs {
						if *u.SchoolID == tid {
							isChild = true
							break
						}
					}
					if isChild {
						filtered = append(filtered, u)
						continue
					}
				}
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
