package admin

import (
	"errors"
	"office-file-sharing/backend/internal/shared/models"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service interface {
	GetStats(schoolID *string) (*SystemStats, error)
	GetAllUsers(actorRole string, actorSchoolID *uuid.UUID) ([]UserResponse, error)
	CreateUser(req CreateUserRequest, actorRole string, actorSchoolID *uuid.UUID) (*UserResponse, error)
	UpdateUser(id uuid.UUID, req UpdateUserRequest, actorRole string, actorSchoolID *uuid.UUID) (*UserResponse, error)
	DeleteUser(id uuid.UUID, actorUserID uuid.UUID) error
	GetAllDocumentTypes(actorRole string, actorSchoolID *uuid.UUID) ([]DocumentTypeResponse, error)
	CreateDocumentType(req CreateDocTypeRequest, actorRole string, actorSchoolID *uuid.UUID) (*DocumentTypeResponse, error)
	UpdateDocumentType(id uuid.UUID, req UpdateDocTypeRequest) (*DocumentTypeResponse, error)
	DeleteDocumentType(id uuid.UUID) error
	GetAllSchools(schoolID *string) ([]SchoolResponse, error)
	UpdateSchool(id uuid.UUID, req UpdateSchoolRequest) (*SchoolResponse, error)
	GetAllRoles(actorRole string, actorSchoolID *uuid.UUID) ([]RoleResponse, error)
	CreateRole(req CreateRoleRequest, actorRole string, actorSchoolID *uuid.UUID) (*RoleResponse, error)
	UpdateRole(id uuid.UUID, req UpdateRoleRequest, actorRole string, actorSchoolID *uuid.UUID) (*RoleResponse, error)
	DeleteRole(id uuid.UUID, actorRole string, actorSchoolID *uuid.UUID) error

	// Organization CRUD (SuperAdmin and DHE Admins)
	GetAllOrganizations(actorRole string, actorSchoolID *uuid.UUID) ([]OrganizationResponse, error)
	CreateOrganization(req CreateOrganizationRequest, actorRole string, actorSchoolID *uuid.UUID) (*OrganizationResponse, error)
	UpdateOrganization(id uuid.UUID, req UpdateOrganizationRequest, actorRole string, actorSchoolID *uuid.UUID) (*OrganizationResponse, error)
	DeleteOrganization(id uuid.UUID, actorRole string, actorSchoolID *uuid.UUID) error

	// Peer Connections (same-level role sharing)
	GetPeerConnections(actorRole string, actorSchoolID *uuid.UUID) ([]PeerConnectionResponse, error)
	RequestPeerConnection(req CreatePeerConnectionRequest, actorRole string, actorSchoolID *uuid.UUID) (*PeerConnectionResponse, error)
	AcceptPeerConnection(connectionID uuid.UUID, actorRole string, actorSchoolID *uuid.UUID) (*PeerConnectionResponse, error)
	RejectPeerConnection(connectionID uuid.UUID, actorRole string, actorSchoolID *uuid.UUID) (*PeerConnectionResponse, error)
	RevokePeerConnection(connectionID uuid.UUID, actorRole string, actorSchoolID *uuid.UUID) (*PeerConnectionResponse, error)
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

func (s *service) GetAllUsers(actorRole string, actorSchoolID *uuid.UUID) ([]UserResponse, error) {
	allUsers, err := s.repo.GetAllUsers(nil)
	if err != nil {
		return nil, err
	}

	if actorRole == "SuperAdmin" {
		return allUsers, nil
	}

	if actorSchoolID == nil {
		return nil, errors.New("actor organization context required")
	}

	repoImpl, ok := s.repo.(*repository)
	if !ok {
		return nil, errors.New("repository conversion failed")
	}

	var actorOrg models.Organization
	if err := repoImpl.db.Where("tenant_id = ?", *actorSchoolID).First(&actorOrg).Error; err != nil {
		var filtered []UserResponse
		for _, u := range allUsers {
			if u.SchoolID != nil && *u.SchoolID == *actorSchoolID {
				filtered = append(filtered, u)
			}
		}
		return filtered, nil
	}

	var childOrgs []models.Organization
	repoImpl.db.Where("parent_org_id = ?", actorOrg.ID).Find(&childOrgs)

	allowedSchoolIDs := map[uuid.UUID]bool{
		*actorSchoolID: true,
	}
	for _, child := range childOrgs {
		if child.TenantID != nil {
			allowedSchoolIDs[*child.TenantID] = true
		}
	}

	var filtered []UserResponse
	for _, u := range allUsers {
		if u.SchoolID != nil {
			if allowedSchoolIDs[*u.SchoolID] {
				filtered = append(filtered, u)
			}
		}
	}

	return filtered, nil
}

func (s *service) CreateUser(req CreateUserRequest, actorRole string, actorSchoolID *uuid.UUID) (*UserResponse, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" {
		return nil, errors.New("name and email are required")
	}
	if req.Password == "" {
		req.Password = "password" // default password
	}

	var targetSchoolID *uuid.UUID
	if actorRole == "SuperAdmin" {
		targetSchoolID = req.SchoolID
	} else {
		if actorSchoolID == nil {
			return nil, errors.New("actor organization context required")
		}
		targetSchoolID = actorSchoolID

		// Get actor's role record
		actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
		if err != nil {
			return nil, errors.New("actor role not found in hierarchy")
		}

		if !actorRoleRec.IsAdminAccess {
			return nil, errors.New("access denied: administrative access required to create users")
		}

		// Get target role record
		targetRoleRec, err := s.repo.GetRoleByName(req.Role, targetSchoolID)
		if err != nil {
			return nil, errors.New("target role not found in hierarchy")
		}

		// Validate target role is either same level or direct child
		isSameRole := targetRoleRec.RoleName == actorRoleRec.RoleName
		isDirectChild := targetRoleRec.ParentRoleID != nil && *targetRoleRec.ParentRoleID == actorRoleRec.ID
		if !isSameRole && !isDirectChild {
			return nil, errors.New("unauthorized role assignment: you can only assign users to your own role or to a direct child role")
		}
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

	if strings.EqualFold(u.Role, "SuperAdmin") {
		return nil, errors.New("the system SuperAdmin user cannot be updated from the user management screen")
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
	var targetSchoolID *uuid.UUID
	if actorRole == "SuperAdmin" {
		targetSchoolID = req.SchoolID
		u.SchoolID = targetSchoolID
	} else {
		if actorSchoolID == nil {
			return nil, errors.New("actor organization context required")
		}
		targetSchoolID = actorSchoolID
		u.SchoolID = targetSchoolID
	}

	if req.Role != "" && req.Role != u.Role {
		if actorRole != "SuperAdmin" {
			// Get actor's role record
			actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
			if err != nil {
				return nil, errors.New("actor role not found in hierarchy")
			}

			if !actorRoleRec.IsAdminAccess {
				return nil, errors.New("access denied: administrative access required to assign roles")
			}

			// Get target role record
			targetRoleRec, err := s.repo.GetRoleByName(req.Role, targetSchoolID)
			if err != nil {
				return nil, errors.New("target role not found in hierarchy")
			}

			// Validate target role is either same level or direct child
			isSameRole := targetRoleRec.RoleName == actorRoleRec.RoleName
			isDirectChild := targetRoleRec.ParentRoleID != nil && *targetRoleRec.ParentRoleID == actorRoleRec.ID
			if !isSameRole && !isDirectChild {
				return nil, errors.New("unauthorized role assignment: you can only assign users to your own role or to a direct child role")
			}
		}
		u.Role = req.Role
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

func (s *service) DeleteUser(id uuid.UUID, actorUserID uuid.UUID) error {
	if id == actorUserID {
		return errors.New("cannot delete yourself")
	}

	u, err := s.repo.GetUserByID(id)
	if err != nil {
		return errors.New("user not found")
	}
	if strings.EqualFold(u.Role, "SuperAdmin") {
		return errors.New("the system SuperAdmin user cannot be deleted")
	}

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

func (s *service) GetAllDocumentTypes(actorRole string, actorSchoolID *uuid.UUID) ([]DocumentTypeResponse, error) {
	repoImpl, ok := s.repo.(*repository)
	if !ok {
		return nil, errors.New("repository conversion failed")
	}

	var docTypes []models.DocumentType
	if err := repoImpl.db.Preload("School").Preload("CreatorRole").Order("name asc").Find(&docTypes).Error; err != nil {
		return nil, err
	}

	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	if err != nil && actorRole != "SuperAdmin" {
		return nil, errors.New("actor role not found in system hierarchy")
	}

	// Fetch active peer connections if non-SuperAdmin
	var peerConnections []models.PeerConnection
	if actorRole != "SuperAdmin" && actorRoleRec != nil {
		repoImpl.db.Where("status = ?", "accepted").
			Where("sender_role_id = ? OR target_role_id = ?", actorRoleRec.ID, actorRoleRec.ID).
			Find(&peerConnections)
	}

	isPeerOf := func(creatorRoleID uuid.UUID) bool {
		for _, conn := range peerConnections {
			if (conn.SenderRoleID == actorRoleRec.ID && conn.TargetRoleID == creatorRoleID) ||
				(conn.TargetRoleID == actorRoleRec.ID && conn.SenderRoleID == creatorRoleID) {
				return true
			}
		}
		return false
	}

	var resp []DocumentTypeResponse
	for _, dt := range docTypes {
		visible := false
		if actorRole == "SuperAdmin" {
			visible = true
		} else if dt.SchoolID == nil {
			visible = true
		} else {
			// 1. Legacy/default scoping: no creator role set, but belongs to actor's school
			if dt.CreatorRoleID == nil {
				if actorSchoolID != nil && *dt.SchoolID == *actorSchoolID {
					visible = true
				}
			} else {
				// 2. Visible if owned by current role
				if *dt.CreatorRoleID == actorRoleRec.ID {
					visible = true
				} else if dt.CreatorRole != nil && strings.HasPrefix(actorRoleRec.Path, dt.CreatorRole.Path) {
					// 3. Visible if inherited from ancestors (creator path is prefix of actor path)
					visible = true
				} else if isPeerOf(*dt.CreatorRoleID) {
					// 4. Visible if shared via peer connection
					visible = true
				}
			}
		}

		if visible {
			schoolName := "Global (All Organizations)"
			if dt.SchoolID != nil && dt.School.Name != "" {
				schoolName = dt.School.Name
			}
			resp = append(resp, DocumentTypeResponse{
				ID:             dt.ID,
				SchoolID:       dt.SchoolID,
				SchoolName:     schoolName,
				Name:           dt.Name,
				Slug:           dt.Slug,
				WorkflowStages: dt.WorkflowStages,
				RequiredFields: dt.RequiredFields,
				SlaHours:       0,
				Active:         dt.Active,
			})
		}
	}
	return resp, nil
}

func (s *service) CreateDocumentType(req CreateDocTypeRequest, actorRole string, actorSchoolID *uuid.UUID) (*DocumentTypeResponse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("document type name is required")
	}
	if req.WorkflowStages == "" {
		req.WorkflowStages = "[]"
	}
	if req.RequiredFields == "" {
		req.RequiredFields = "[]"
	}

	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	var creatorRoleID *uuid.UUID
	if err == nil && actorRoleRec != nil {
		creatorRoleID = &actorRoleRec.ID

		// Enforce same-level peer file attribute uniqueness constraint
		repoImpl, ok := s.repo.(*repository)
		if ok {
			var activeConns []models.PeerConnection
			repoImpl.db.Where("status = ?", "accepted").
				Where("sender_role_id = ? OR target_role_id = ?", actorRoleRec.ID, actorRoleRec.ID).
				Find(&activeConns)

			for _, conn := range activeConns {
				peerRoleID := conn.SenderRoleID
				if conn.SenderRoleID == actorRoleRec.ID {
					peerRoleID = conn.TargetRoleID
				}

				var peerRole models.Role
				if err := repoImpl.db.First(&peerRole, "id = ?", peerRoleID).Error; err == nil && peerRole.TenantID != nil {
					var conflictCount int64
					repoImpl.db.Model(&models.DocumentType{}).
						Where("school_id = ?", *peerRole.TenantID).
						Where("name = ? OR slug = ?", req.Name, req.Slug).
						Count(&conflictCount)
					if conflictCount > 0 {
						return nil, errors.New("cannot create file type: conflicting name/slug '" + req.Name + "' exists in peer-connected branch")
					}
				}
			}
		}
	}

	var schoolID *uuid.UUID
	if req.SchoolID != nil && *req.SchoolID != uuid.Nil {
		schoolID = req.SchoolID
	} else if actorSchoolID != nil {
		schoolID = actorSchoolID
	}

	dt := &models.DocumentType{
		ID:             uuid.New(),
		SchoolID:       schoolID,
		Name:           req.Name,
		Slug:           req.Slug,
		WorkflowStages: req.WorkflowStages,
		RequiredFields: req.RequiredFields,
		Active:         true,
		CreatorRoleID:  creatorRoleID,
	}

	if err := s.repo.CreateDocumentType(dt); err != nil {
		return nil, err
	}

	schoolName := "Global (All Organizations)"
	if dt.SchoolID != nil {
		if s, err := s.repo.GetSchoolByID(*dt.SchoolID); err == nil && s != nil {
			schoolName = s.Name
		}
	}

	return &DocumentTypeResponse{
		ID:             dt.ID,
		SchoolID:       dt.SchoolID,
		SchoolName:     schoolName,
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

	schoolName := "Global (All Organizations)"
	if dt.SchoolID != nil {
		if s, err := s.repo.GetSchoolByID(*dt.SchoolID); err == nil && s != nil {
			schoolName = s.Name
		}
	}

	return &DocumentTypeResponse{
		ID:             dt.ID,
		SchoolID:       dt.SchoolID,
		SchoolName:     schoolName,
		Name:           dt.Name,
		Slug:           dt.Slug,
		WorkflowStages: dt.WorkflowStages,
		RequiredFields: dt.RequiredFields,
		SlaHours:       0,
		Active:         dt.Active,
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

func (s *service) GetAllRoles(actorRole string, actorSchoolID *uuid.UUID) ([]RoleResponse, error) {
	allRoles, err := s.repo.GetAllRoles(nil)
	if err != nil {
		return nil, err
	}

	if actorRole == "SuperAdmin" {
		resp := make([]RoleResponse, len(allRoles))
		for i, r := range allRoles {
			parentName := ""
			if r.ParentRoleID != nil {
				p, err := s.repo.GetRoleByID(*r.ParentRoleID)
				if err == nil {
					parentName = p.RoleName
				}
			}
			resp[i] = RoleResponse{
				ID:             r.ID,
				RoleName:       r.RoleName,
				IsAdminAccess:  r.IsAdminAccess,
				ParentRoleID:   r.ParentRoleID,
				ParentRoleName: parentName,
				TenantID:       r.TenantID,
				OrgName:        r.OrgName,
				CreatedBy:      r.CreatedBy,
				Path:           r.Path,
				CreatedAt:      r.CreatedAt,
				UpdatedAt:      r.UpdatedAt,
			}
		}
		return resp, nil
	}

	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	if err != nil {
		return nil, errors.New("unauthorized: actor role not found in system hierarchy")
	}

	var filtered []RoleResponse
	for _, r := range allRoles {
		if strings.HasPrefix(r.Path, actorRoleRec.Path) {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

func (s *service) CreateRole(req CreateRoleRequest, actorRole string, actorSchoolID *uuid.UUID) (*RoleResponse, error) {
	if strings.TrimSpace(req.RoleName) == "" {
		return nil, errors.New("role name is required")
	}

	// Enforce global SuperAdmin uniqueness
	if strings.EqualFold(req.RoleName, "SuperAdmin") {
		return nil, errors.New("creation of SuperAdmin role is prohibited")
	}

	// Find actor's role record to verify boundaries
	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	if err != nil {
		return nil, errors.New("unauthorized: actor role not found in system hierarchy")
	}

	newID := uuid.New()
	var parentRoleID *uuid.UUID
	var path string
	var parentRole *models.Role

	if actorRole == "SuperAdmin" {
		if req.ParentRoleID != nil {
			var err error
			parentRole, err = s.repo.GetRoleByID(*req.ParentRoleID)
			if err != nil {
				return nil, errors.New("parent role not found")
			}
			parentRoleID = req.ParentRoleID
			path = parentRole.Path + newID.String() + "/"
		} else {
			path = "/" + newID.String() + "/"
		}
	} else {
		// Non-SuperAdmins: automatically parent to their own role
		if !actorRoleRec.IsAdminAccess {
			return nil, errors.New("cannot create role: administrative access (isAdminAccess = true) is required to create child roles")
		}
		parentRole = actorRoleRec
		parentRoleID = &actorRoleRec.ID
		path = actorRoleRec.Path + newID.String() + "/"
	}

	// Determine tenant scope
	var targetTenantID *uuid.UUID
	if parentRole != nil && parentRole.TenantID != nil {
		targetTenantID = parentRole.TenantID
	} else if actorRole == "SuperAdmin" {
		targetTenantID = req.TenantID
	} else {
		targetTenantID = actorSchoolID
	}

	// Scoped uniqueness check
	var schoolIDStr *string
	if targetTenantID != nil {
		val := targetTenantID.String()
		schoolIDStr = &val
	}
	existingRoles, err := s.repo.GetAllRoles(schoolIDStr)
	if err == nil {
		for _, er := range existingRoles {
			if strings.EqualFold(er.RoleName, req.RoleName) {
				return nil, errors.New("role name must be unique within this tenant scope")
			}
		}
	}

	// Max tree depth check (10 levels)
	depth := strings.Count(path, "/") - 1
	if depth > 10 {
		return nil, errors.New("maximum tree hierarchy depth of 10 exceeded")
	}

	// Elevation protection
	if req.IsAdminAccess && !actorRoleRec.IsAdminAccess {
		return nil, errors.New("cannot elevate administrative access above your own role level")
	}

	role := &models.Role{
		ID:            newID,
		RoleName:      req.RoleName,
		IsAdminAccess: req.IsAdminAccess,
		ParentRoleID:  parentRoleID,
		TenantID:      targetTenantID,
		CreatedBy:     actorRole,
		Path:          path,
	}

	if err := s.repo.CreateRole(role); err != nil {
		return nil, err
	}

	parentName := ""
	if role.ParentRoleID != nil {
		p, err := s.repo.GetRoleByID(*role.ParentRoleID)
		if err == nil {
			parentName = p.RoleName
		}
	}

	orgName := "System Level"
	if role.TenantID != nil {
		tenantSchool, err := s.repo.GetSchoolByID(*role.TenantID)
		if err == nil {
			orgName = tenantSchool.Name
		}
	}

	return &RoleResponse{
		ID:             role.ID,
		RoleName:       role.RoleName,
		IsAdminAccess:  role.IsAdminAccess,
		ParentRoleID:   role.ParentRoleID,
		ParentRoleName: parentName,
		TenantID:       role.TenantID,
		OrgName:        orgName,
		CreatedBy:      role.CreatedBy,
		Path:           role.Path,
		CreatedAt:      role.CreatedAt,
		UpdatedAt:      role.UpdatedAt,
	}, nil
}

func (s *service) UpdateRole(id uuid.UUID, req UpdateRoleRequest, actorRole string, actorSchoolID *uuid.UUID) (*RoleResponse, error) {
	role, err := s.repo.GetRoleByID(id)
	if err != nil {
		return nil, errors.New("role not found")
	}

	// SuperAdmin role protections
	if strings.EqualFold(role.RoleName, "SuperAdmin") {
		return nil, errors.New("the system SuperAdmin role cannot be modified")
	}
	if strings.EqualFold(req.RoleName, "SuperAdmin") && !strings.EqualFold(role.RoleName, "SuperAdmin") {
		return nil, errors.New("cannot rename role to SuperAdmin")
	}

	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	if err != nil {
		return nil, errors.New("unauthorized: actor role not found in system hierarchy")
	}

	// Subtree boundary validation: must be a direct child of actor's role
	if actorRole != "SuperAdmin" {
		if role.ParentRoleID == nil || *role.ParentRoleID != actorRoleRec.ID {
			return nil, errors.New("access denied: you can only update roles that are direct children of your own role")
		}
		if !actorRoleRec.IsAdminAccess {
			return nil, errors.New("access denied: administrative access (isAdminAccess = true) is required to manage roles")
		}
	}

	// Scoped uniqueness check if name is changing
	if req.RoleName != "" && !strings.EqualFold(req.RoleName, role.RoleName) {
		var schoolIDStr *string
		if role.TenantID != nil {
			val := role.TenantID.String()
			schoolIDStr = &val
		}
		existingRoles, err := s.repo.GetAllRoles(schoolIDStr)
		if err == nil {
			for _, er := range existingRoles {
				if strings.EqualFold(er.RoleName, req.RoleName) {
					return nil, errors.New("role name must be unique within this tenant scope")
				}
			}
		}
		role.RoleName = req.RoleName
	}

	// Elevation protection
	if req.IsAdminAccess && !actorRoleRec.IsAdminAccess {
		return nil, errors.New("cannot elevate administrative access above your own role level")
	}
	role.IsAdminAccess = req.IsAdminAccess

	// Tenant reassignment validation
	if actorRole == "SuperAdmin" {
		role.TenantID = req.TenantID
	}

	// Handle parent reparenting and recursive materialized path updates
	parentChanged := false
	if req.ParentRoleID != nil {
		if role.ParentRoleID == nil || *req.ParentRoleID != *role.ParentRoleID {
			newParentRole, err := s.repo.GetRoleByID(*req.ParentRoleID)
			if err != nil {
				return nil, errors.New("proposed parent role not found")
			}

			// Verify actor has access to new parent role (must be the actor's role itself to remain a direct child)
			if actorRole != "SuperAdmin" {
				if newParentRole.ID != actorRoleRec.ID {
					return nil, errors.New("cannot reparent role: you can only parent roles to your own role")
				}
			}

			// Prevent circular cycles
			isCircular := newParentRole.ID == role.ID || strings.Contains(newParentRole.Path, "/"+role.ID.String()+"/")
			if isCircular {
				return nil, errors.New("circular hierarchy reference detected")
			}

			// Inherit organization from parent role
			role.TenantID = newParentRole.TenantID

			oldPath := role.Path
			newPath := newParentRole.Path + role.ID.String() + "/"

			// Max tree depth check
			depth := strings.Count(newPath, "/") - 1
			if depth > 10 {
				return nil, errors.New("maximum tree hierarchy depth of 10 exceeded")
			}

			role.ParentRoleID = req.ParentRoleID
			role.Path = newPath
			parentChanged = true

			// Cascade path and TenantID update to all descendants recursively
			repoImpl, ok := s.repo.(*repository)
			if ok {
				var descendants []models.Role
				if err := repoImpl.db.Where("path LIKE ?", oldPath+"%").Find(&descendants).Error; err == nil {
					for _, desc := range descendants {
						desc.Path = strings.Replace(desc.Path, oldPath, newPath, 1)
						desc.TenantID = role.TenantID
						repoImpl.db.Save(&desc)
					}
				}
			}
		}
	} else if role.ParentRoleID != nil {
		// Parent changed to nil
		if actorRole != "SuperAdmin" {
			return nil, errors.New("parent role is required for non-SuperAdmins")
		}

		oldPath := role.Path
		newPath := "/" + role.ID.String() + "/"

		role.ParentRoleID = nil
		role.Path = newPath
		parentChanged = true

		repoImpl, ok := s.repo.(*repository)
		if ok {
			var descendants []models.Role
			if err := repoImpl.db.Where("path LIKE ?", oldPath+"%").Find(&descendants).Error; err == nil {
				for _, desc := range descendants {
					desc.Path = strings.Replace(desc.Path, oldPath, newPath, 1)
					desc.TenantID = role.TenantID
					repoImpl.db.Save(&desc)
				}
			}
		}
	}
	_ = parentChanged

	if err := s.repo.UpdateRole(role); err != nil {
		return nil, err
	}

	parentName := ""
	if role.ParentRoleID != nil {
		p, err := s.repo.GetRoleByID(*role.ParentRoleID)
		if err == nil {
			parentName = p.RoleName
		}
	}

	orgName := "System Level"
	if role.TenantID != nil {
		tenantSchool, err := s.repo.GetSchoolByID(*role.TenantID)
		if err == nil {
			orgName = tenantSchool.Name
		}
	}

	return &RoleResponse{
		ID:             role.ID,
		RoleName:       role.RoleName,
		IsAdminAccess:  role.IsAdminAccess,
		ParentRoleID:   role.ParentRoleID,
		ParentRoleName: parentName,
		TenantID:       role.TenantID,
		OrgName:        orgName,
		CreatedBy:      role.CreatedBy,
		Path:           role.Path,
		CreatedAt:      role.CreatedAt,
		UpdatedAt:      role.UpdatedAt,
	}, nil
}

func (s *service) DeleteRole(id uuid.UUID, actorRole string, actorSchoolID *uuid.UUID) error {
	role, err := s.repo.GetRoleByID(id)
	if err != nil {
		return errors.New("role not found")
	}

	// SuperAdmin role protections
	if strings.EqualFold(role.RoleName, "SuperAdmin") {
		return errors.New("the system SuperAdmin role cannot be deleted")
	}

	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	if err != nil {
		return errors.New("unauthorized: actor role not found in system hierarchy")
	}

	// Subtree boundary validation: must be a direct child of actor's role
	if actorRole != "SuperAdmin" {
		if role.ParentRoleID == nil || *role.ParentRoleID != actorRoleRec.ID {
			return errors.New("access denied: you can only delete roles that are direct children of your own role")
		}
		if !actorRoleRec.IsAdminAccess {
			return errors.New("access denied: administrative access (isAdminAccess = true) is required to manage roles")
		}
	}

	// Ensure no users are assigned to this role
	hasUsers, err := s.repo.CheckUsersWithRole(role.RoleName)
	if err != nil {
		return err
	}
	if hasUsers {
		return errors.New("cannot delete role: role is currently assigned to active users")
	}

	// Ensure no descendant child roles exist
	hasChildren, err := s.repo.CheckRoleHasChildren(role.ID)
	if err != nil {
		return err
	}
	if hasChildren {
		return errors.New("cannot delete role: delete descendant child roles first")
	}

	return s.repo.DeleteRole(id)
}

// ── Organization CRUD (SuperAdmin only) ──────────────────────────────────────

func (s *service) GetAllOrganizations(actorRole string, actorSchoolID *uuid.UUID) ([]OrganizationResponse, error) {
	orgs, err := s.repo.GetAllOrganizations()
	if err != nil {
		return nil, err
	}

	var filtered []models.Organization
	if actorRole == "SuperAdmin" {
		filtered = orgs
	} else {
		if actorSchoolID == nil {
			return nil, errors.New("actor organization context required")
		}
		var actorOrg models.Organization
		repoImpl, ok := s.repo.(*repository)
		if !ok {
			return nil, errors.New("repository conversion failed")
		}
		repoImpl.db.Where("tenant_id = ?", *actorSchoolID).First(&actorOrg)

		// Recursive helper function to check if checkOrgID is a descendant of ancestorOrgID
		isDescendant := func(checkOrgID uuid.UUID, ancestorOrgID uuid.UUID) bool {
			currID := checkOrgID
			visited := make(map[uuid.UUID]bool)
			for {
				if visited[currID] {
					break // prevent infinite loops in cycles
				}
				visited[currID] = true

				var o models.Organization
				if err := repoImpl.db.Select("id, parent_org_id").Where("id = ?", currID).First(&o).Error; err != nil {
					return false
				}
				if o.ParentOrgID == nil {
					return false
				}
				if *o.ParentOrgID == ancestorOrgID {
					return true
				}
				currID = *o.ParentOrgID
			}
			return false
		}

		for _, org := range orgs {
			if org.ID == actorOrg.ID || isDescendant(org.ID, actorOrg.ID) {
				filtered = append(filtered, org)
			}
		}
	}

	resp := make([]OrganizationResponse, len(filtered))
	for i, org := range filtered {
		parentName := ""
		if org.ParentOrg != nil {
			parentName = org.ParentOrg.OrganizationName
		}
		pocName := ""
		if org.PointOfContact != nil {
			pocName = org.PointOfContact.Name
		}
		resp[i] = OrganizationResponse{
			ID:               org.ID,
			OrganizationName: org.OrganizationName,
			Type:             org.Type,
			ParentOrgID:      org.ParentOrgID,
			ParentOrgName:    parentName,
			PointOfContactID: org.PointOfContactID,
			PointOfContact:   pocName,
			CreatedBy:        org.CreatedBy,
			TenantID:         org.TenantID,
			CreatedAt:        org.CreatedAt,
			UpdatedAt:        org.UpdatedAt,
		}
	}
	return resp, nil
}

func (s *service) CreateOrganization(req CreateOrganizationRequest, actorRole string, actorSchoolID *uuid.UUID) (*OrganizationResponse, error) {
	// Ensure the actor has admin access (either SuperAdmin, or a role with IsAdminAccess = true)
	isAdmin := false
	if actorRole == "SuperAdmin" {
		isAdmin = true
	} else if actorSchoolID != nil {
		roleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
		if err == nil && roleRec.IsAdminAccess {
			isAdmin = true
		}
	}

	if !isAdmin {
		return nil, errors.New("access denied: only administrative users can create organizations")
	}
	if strings.TrimSpace(req.OrganizationName) == "" {
		return nil, errors.New("organization name is required")
	}

	repoImpl, ok := s.repo.(*repository)
	if !ok {
		return nil, errors.New("repository conversion failed")
	}

	var parentOrgID *uuid.UUID
	var creatorRoleRec *models.Role

	if actorRole != "SuperAdmin" {
		if actorSchoolID == nil {
			return nil, errors.New("actor organization context required")
		}
		var actorOrg models.Organization
		if err := repoImpl.db.Where("tenant_id = ?", *actorSchoolID).First(&actorOrg).Error; err != nil {
			return nil, errors.New("actor organization not found")
		}
		parentOrgID = &actorOrg.ID

		roleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
		if err != nil {
			return nil, errors.New("actor role record not found")
		}
		creatorRoleRec = roleRec
	} else {
		parentOrgID = req.ParentOrgID
		roleRec, err := s.repo.GetRoleByName("SuperAdmin", nil)
		if err == nil {
			creatorRoleRec = roleRec
		}
	}

	if req.AdminEmail != "" {
		var existingUser models.User
		if err := repoImpl.db.First(&existingUser, "email = ?", strings.TrimSpace(strings.ToLower(req.AdminEmail))).Error; err == nil {
			return nil, errors.New("a user with this admin email already exists in the system")
		}
	}

	var org *models.Organization
	errTx := repoImpl.db.Transaction(func(tx *gorm.DB) error {
		var tenantID *uuid.UUID
		var pocID *uuid.UUID

		if req.AdminEmail != "" {
			newSchoolID := uuid.New()
			baseSlug := strings.ToLower(strings.ReplaceAll(req.OrganizationName, " ", "-"))
			slug := baseSlug + "-" + newSchoolID.String()[:8]
			newSchool := &models.School{
				ID:   newSchoolID,
				Name: req.OrganizationName,
				Slug: slug,
			}
			if err := tx.Create(newSchool).Error; err != nil {
				return errors.New("failed to create tenant school: " + err.Error())
			}
			tenantID = &newSchool.ID

			childRoleName := "Admin " + req.OrganizationName

			var parentRoleID *uuid.UUID
			var parentPath string
			if creatorRoleRec != nil {
				parentRoleID = &creatorRoleRec.ID
				parentPath = creatorRoleRec.Path
			}

			if parentOrgID != nil {
				var parentOrg models.Organization
				if err := tx.Where("id = ?", *parentOrgID).First(&parentOrg).Error; err == nil && parentOrg.TenantID != nil {
					var parentTenantRole models.Role
					if err := tx.Where("tenant_id = ? AND is_admin_access = true", *parentOrg.TenantID).First(&parentTenantRole).Error; err == nil {
						parentRoleID = &parentTenantRole.ID
						parentPath = parentTenantRole.Path
					}
				}
			}

			newRoleID := uuid.New()
			var path string
			if parentRoleID == nil {
				path = "/" + newRoleID.String() + "/"
			} else {
				path = parentPath + newRoleID.String() + "/"
			}
			newRole := &models.Role{
				ID:            newRoleID,
				RoleName:      childRoleName,
				IsAdminAccess: true,
				ParentRoleID:  parentRoleID,
				TenantID:      tenantID,
				CreatedBy:     actorRole,
				Path:          path,
			}
			if err := tx.Create(newRole).Error; err != nil {
				return errors.New("failed to create organization admin role: " + err.Error())
			}

			password := req.AdminPassword
			if password == "" {
				password = "password"
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}

			newUser := &models.User{
				ID:           uuid.New(),
				Name:         "Admin",
				Email:        strings.TrimSpace(strings.ToLower(req.AdminEmail)),
				PasswordHash: string(hash),
				Role:         childRoleName,
				SchoolID:     tenantID,
			}
			if err := tx.Create(newUser).Error; err != nil {
				return errors.New("failed to create point of contact admin user: " + err.Error())
			}
			pocID = &newUser.ID
		} else if req.PointOfContactID != nil {
			pocID = req.PointOfContactID
			tenantID = req.TenantID
		}

		org = &models.Organization{
			ID:               uuid.New(),
			OrganizationName: req.OrganizationName,
			Type:             req.Type,
			ParentOrgID:      parentOrgID,
			PointOfContactID: pocID,
			CreatedBy:        actorRole,
			TenantID:         tenantID,
		}

		if err := tx.Create(org).Error; err != nil {
			return err
		}
		return nil
	})

	if errTx != nil {
		return nil, errTx
	}

	return s.GetOrganizationByID(org.ID)
}

func (s *service) GetOrganizationByID(id uuid.UUID) (*OrganizationResponse, error) {
	org, err := s.repo.GetOrganizationByID(id)
	if err != nil {
		return nil, err
	}
	parentName := ""
	if org.ParentOrg != nil {
		parentName = org.ParentOrg.OrganizationName
	}
	pocName := ""
	if org.PointOfContact != nil {
		pocName = org.PointOfContact.Name
	}
	return &OrganizationResponse{
		ID:               org.ID,
		OrganizationName: org.OrganizationName,
		Type:             org.Type,
		ParentOrgID:      org.ParentOrgID,
		ParentOrgName:    parentName,
		PointOfContactID: org.PointOfContactID,
		PointOfContact:   pocName,
		CreatedBy:        org.CreatedBy,
		TenantID:         org.TenantID,
		CreatedAt:        org.CreatedAt,
		UpdatedAt:        org.UpdatedAt,
	}, nil
}

func (s *service) UpdateOrganization(id uuid.UUID, req UpdateOrganizationRequest, actorRole string, actorSchoolID *uuid.UUID) (*OrganizationResponse, error) {
	if actorRole != "SuperAdmin" && actorRole != "DHE" && actorRole != "Admin" {
		return nil, errors.New("access denied: only SuperAdmin and DHE Admins can manage organizations")
	}
	org, err := s.repo.GetOrganizationByID(id)
	if err != nil {
		return nil, err
	}

	repoImpl, ok := s.repo.(*repository)
	if !ok {
		return nil, errors.New("repository conversion failed")
	}

	if actorRole != "SuperAdmin" {
		if actorSchoolID == nil {
			return nil, errors.New("actor organization context required")
		}
		var actorOrg models.Organization
		if err := repoImpl.db.Where("tenant_id = ?", *actorSchoolID).First(&actorOrg).Error; err != nil {
			return nil, errors.New("actor organization not found")
		}

		if org.ID != actorOrg.ID && (org.ParentOrgID == nil || *org.ParentOrgID != actorOrg.ID) {
			return nil, errors.New("access denied: you can only update child organizations under your scope")
		}
	}

	if req.OrganizationName != "" {
		org.OrganizationName = req.OrganizationName
	}
	if req.Type != "" {
		org.Type = req.Type
	}
	if actorRole == "SuperAdmin" {
		org.ParentOrgID = req.ParentOrgID
	}
	org.PointOfContactID = req.PointOfContactID

	if err := repoImpl.db.Save(org).Error; err != nil {
		return nil, err
	}

	return s.GetOrganizationByID(org.ID)
}

func deleteOrgRecursive(tx *gorm.DB, orgID uuid.UUID) error {
	var childOrgs []models.Organization
	if err := tx.Where("parent_org_id = ?", orgID).Find(&childOrgs).Error; err == nil {
		for _, child := range childOrgs {
			if err := deleteOrgRecursive(tx, child.ID); err != nil {
				return err
			}
		}
	}

	var org models.Organization
	if err := tx.Where("id = ?", orgID).First(&org).Error; err != nil {
		return err
	}

	if org.TenantID != nil {
		if err := tx.Where("school_id = ?", *org.TenantID).Delete(&models.User{}).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ?", *org.TenantID).Delete(&models.Role{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", *org.TenantID).Delete(&models.School{}).Error; err != nil {
			return err
		}
	}

	return tx.Where("id = ?", orgID).Delete(&models.Organization{}).Error
}

func (s *service) DeleteOrganization(id uuid.UUID, actorRole string, actorSchoolID *uuid.UUID) error {
	repoImpl, ok := s.repo.(*repository)
	if !ok {
		return errors.New("repository conversion failed")
	}

	if actorRole == "SuperAdmin" {
		return repoImpl.db.Transaction(func(tx *gorm.DB) error {
			return deleteOrgRecursive(tx, id)
		})
	}

	// Verify the actor has administrative access
	isAdmin := false
	if actorSchoolID != nil {
		roleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
		if err == nil && roleRec.IsAdminAccess {
			isAdmin = true
		}
	}

	if !isAdmin {
		return errors.New("access denied: administrative access required to delete organizations")
	}

	org, err := s.repo.GetOrganizationByID(id)
	if err != nil {
		return err
	}

	if actorSchoolID == nil {
		return errors.New("actor organization context required")
	}

	var actorOrg models.Organization
	if err := repoImpl.db.Where("tenant_id = ?", *actorSchoolID).First(&actorOrg).Error; err != nil {
		return errors.New("actor organization not found")
	}

	// Traverse up to see if the deleted org is a descendant of the actor's org
	isChild := false
	currParentID := org.ParentOrgID
	for currParentID != nil {
		if *currParentID == actorOrg.ID {
			isChild = true
			break
		}
		var pOrg models.Organization
		if err := repoImpl.db.Where("id = ?", *currParentID).First(&pOrg).Error; err != nil {
			break
		}
		currParentID = pOrg.ParentOrgID
	}

	if !isChild {
		return errors.New("access denied: you can only delete child organizations under your scope")
	}

	return repoImpl.db.Transaction(func(tx *gorm.DB) error {
		return deleteOrgRecursive(tx, id)
	})
}

// ── Peer Connections (same-level role sharing) ───────────────────────────────

func (s *service) GetPeerConnections(actorRole string, actorSchoolID *uuid.UUID) ([]PeerConnectionResponse, error) {
	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	if err != nil {
		return nil, err
	}

	connections, err := s.repo.GetPeerConnectionsByRole(actorRoleRec.ID)
	if err != nil {
		return nil, err
	}

	resp := make([]PeerConnectionResponse, len(connections))
	for i, conn := range connections {
		resp[i] = PeerConnectionResponse{
			ID:             conn.ID,
			SenderRoleID:   conn.SenderRoleID,
			SenderRoleName: conn.SenderRole.RoleName,
			TargetRoleID:   conn.TargetRoleID,
			TargetRoleName: conn.TargetRole.RoleName,
			Status:         conn.Status,
			CreatedAt:      conn.CreatedAt,
			UpdatedAt:      conn.UpdatedAt,
		}
	}
	return resp, nil
}

func (s *service) RequestPeerConnection(req CreatePeerConnectionRequest, actorRole string, actorSchoolID *uuid.UUID) (*PeerConnectionResponse, error) {
	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	if err != nil {
		return nil, err
	}

	targetRole, err := s.repo.GetRoleByID(req.TargetRoleID)
	if err != nil {
		return nil, errors.New("target role not found")
	}

	// 1. Verify same hierarchy depth
	senderDepth := strings.Count(actorRoleRec.Path, "/")
	targetDepth := strings.Count(targetRole.Path, "/")
	if senderDepth != targetDepth {
		return nil, errors.New("peer connections can only be established between same-level roles in the hierarchy")
	}

	// 2. Same-level file attribute uniqueness constraint (overlap check)
	repoImpl, ok := s.repo.(*repository)
	if ok {
		var senderDocTypes, targetDocTypes []models.DocumentType
		if actorRoleRec.TenantID != nil {
			repoImpl.db.Where("school_id = ?", *actorRoleRec.TenantID).Find(&senderDocTypes)
		}
		if targetRole.TenantID != nil {
			repoImpl.db.Where("school_id = ?", *targetRole.TenantID).Find(&targetDocTypes)
		}

		for _, sDT := range senderDocTypes {
			for _, tDT := range targetDocTypes {
				if strings.EqualFold(sDT.Name, tDT.Name) || strings.EqualFold(sDT.Slug, tDT.Slug) {
					return nil, errors.New("peer connection rejected: identical or overlapping file type '" + sDT.Name + "' exists between branches")
				}
			}
		}
	}

	// 3. Create connection
	conn := &models.PeerConnection{
		ID:           uuid.New(),
		SenderRoleID: actorRoleRec.ID,
		TargetRoleID: targetRole.ID,
		Status:       "pending",
	}

	if ok {
		if err := repoImpl.db.Create(conn).Error; err != nil {
			return nil, err
		}
	}

	return &PeerConnectionResponse{
		ID:             conn.ID,
		SenderRoleID:   conn.SenderRoleID,
		SenderRoleName: actorRoleRec.RoleName,
		TargetRoleID:   conn.TargetRoleID,
		TargetRoleName: targetRole.RoleName,
		Status:         conn.Status,
		CreatedAt:      conn.CreatedAt,
		UpdatedAt:      conn.UpdatedAt,
	}, nil
}

func (s *service) AcceptPeerConnection(connectionID uuid.UUID, actorRole string, actorSchoolID *uuid.UUID) (*PeerConnectionResponse, error) {
	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	if err != nil {
		return nil, err
	}

	repoImpl, ok := s.repo.(*repository)
	if !ok {
		return nil, errors.New("repository conversion failed")
	}

	var conn models.PeerConnection
	if err := repoImpl.db.Preload("SenderRole").Preload("TargetRole").First(&conn, "id = ?", connectionID).Error; err != nil {
		return nil, errors.New("peer connection not found")
	}

	if conn.TargetRoleID != actorRoleRec.ID {
		return nil, errors.New("access denied: only the recipient role can accept this peer connection request")
	}

	// Re-verify same-level file type conflicts on acceptance
	var senderDocTypes, targetDocTypes []models.DocumentType
	if conn.SenderRole.TenantID != nil {
		repoImpl.db.Where("school_id = ?", *conn.SenderRole.TenantID).Find(&senderDocTypes)
	}
	if conn.TargetRole.TenantID != nil {
		repoImpl.db.Where("school_id = ?", *conn.TargetRole.TenantID).Find(&targetDocTypes)
	}

	for _, sDT := range senderDocTypes {
		for _, tDT := range targetDocTypes {
			if strings.EqualFold(sDT.Name, tDT.Name) || strings.EqualFold(sDT.Slug, tDT.Slug) {
				return nil, errors.New("peer connection rejected: identical or overlapping file type '" + sDT.Name + "' exists between branches")
			}
		}
	}

	conn.Status = "accepted"
	if err := repoImpl.db.Save(&conn).Error; err != nil {
		return nil, err
	}

	return &PeerConnectionResponse{
		ID:             conn.ID,
		SenderRoleID:   conn.SenderRoleID,
		SenderRoleName: conn.SenderRole.RoleName,
		TargetRoleID:   conn.TargetRoleID,
		TargetRoleName: conn.TargetRole.RoleName,
		Status:         conn.Status,
		CreatedAt:      conn.CreatedAt,
		UpdatedAt:      conn.UpdatedAt,
	}, nil
}

func (s *service) RejectPeerConnection(connectionID uuid.UUID, actorRole string, actorSchoolID *uuid.UUID) (*PeerConnectionResponse, error) {
	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	if err != nil {
		return nil, err
	}

	repoImpl, ok := s.repo.(*repository)
	if !ok {
		return nil, errors.New("repository conversion failed")
	}

	var conn models.PeerConnection
	if err := repoImpl.db.Preload("SenderRole").Preload("TargetRole").First(&conn, "id = ?", connectionID).Error; err != nil {
		return nil, errors.New("peer connection not found")
	}

	if conn.TargetRoleID != actorRoleRec.ID {
		return nil, errors.New("access denied: only the recipient role can reject this peer connection request")
	}

	conn.Status = "rejected"
	if err := repoImpl.db.Save(&conn).Error; err != nil {
		return nil, err
	}

	return &PeerConnectionResponse{
		ID:             conn.ID,
		SenderRoleID:   conn.SenderRoleID,
		SenderRoleName: conn.SenderRole.RoleName,
		TargetRoleID:   conn.TargetRoleID,
		TargetRoleName: conn.TargetRole.RoleName,
		Status:         conn.Status,
		CreatedAt:      conn.CreatedAt,
		UpdatedAt:      conn.UpdatedAt,
	}, nil
}

func (s *service) RevokePeerConnection(connectionID uuid.UUID, actorRole string, actorSchoolID *uuid.UUID) (*PeerConnectionResponse, error) {
	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	if err != nil {
		return nil, err
	}

	repoImpl, ok := s.repo.(*repository)
	if !ok {
		return nil, errors.New("repository conversion failed")
	}

	var conn models.PeerConnection
	if err := repoImpl.db.Preload("SenderRole").Preload("TargetRole").First(&conn, "id = ?", connectionID).Error; err != nil {
		return nil, errors.New("peer connection not found")
	}

	if conn.SenderRoleID != actorRoleRec.ID && conn.TargetRoleID != actorRoleRec.ID {
		return nil, errors.New("access denied: only connected peer roles can revoke this peer connection")
	}

	conn.Status = "revoked"
	if err := repoImpl.db.Save(&conn).Error; err != nil {
		return nil, err
	}

	return &PeerConnectionResponse{
		ID:             conn.ID,
		SenderRoleID:   conn.SenderRoleID,
		SenderRoleName: conn.SenderRole.RoleName,
		TargetRoleID:   conn.TargetRoleID,
		TargetRoleName: conn.TargetRole.RoleName,
		Status:         conn.Status,
		CreatedAt:      conn.CreatedAt,
		UpdatedAt:      conn.UpdatedAt,
	}, nil
}
