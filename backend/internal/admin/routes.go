package admin

import (
	"net/http"
	"office-file-sharing/backend/internal/shared/middleware"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"office-file-sharing/backend/internal/shared/models"
)

// FindRole finds a role by name, prioritizing the given tenant/school ID.
func FindRole(db *gorm.DB, roleName string, schoolID *uuid.UUID) (*models.Role, error) {
	var role models.Role
	var err error
	if schoolID != nil {
		err = db.First(&role, "role_name = ? AND tenant_id = ?", roleName, *schoolID).Error
	}
	if schoolID == nil || err != nil {
		err = db.First(&role, "role_name = ? AND tenant_id IS NULL", roleName).Error
	}
	if err != nil {
		err = db.First(&role, "role_name = ?", roleName).Error
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// HasRole checks if the roleName inherits from targetRole in the tree hierarchy.
func HasRole(db *gorm.DB, roleName string, targetRole string, schoolID *uuid.UUID) bool {
	if roleName == targetRole {
		return true
	}

	uRole, err := FindRole(db, roleName, schoolID)
	if err != nil {
		return false
	}

	tRole, err := FindRole(db, targetRole, schoolID)
	if err != nil {
		return false
	}

	return strings.HasPrefix(uRole.Path, tRole.Path)
}

// HasAdminAccess helper recursively checks if a role has administrative access.
func HasAdminAccess(db *gorm.DB, roleName string, schoolID *uuid.UUID) bool {
	if roleName == "SuperAdmin" || roleName == "Admin" || roleName == "DHE" || roleName == "School Admin" {
		return true
	}

	role, err := FindRole(db, roleName, schoolID)
	if err != nil {
		return false
	}

	return role.IsAdminAccess
}

// adminAccessMiddleware ensures the user has the "Admin" role.
func adminAccessMiddleware(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userIDStr, ok := c.Get("user_id").(string)
			if !ok || userIDStr == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			}

			var user models.User
			if err := db.First(&user, "id = ?", userIDStr).Error; err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found"})
			}

			if !HasAdminAccess(db, user.Role, user.SchoolID) {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Access denied: Admin role required"})
			}

			// Store role and school for downstream scoping
			c.Set("actor_role", user.Role)
			if user.SchoolID != nil {
				c.Set("actor_school_id", user.SchoolID.String())
			}

			return next(c)
		}
	}
}

// superAdminAccessMiddleware ensures the user has the "SuperAdmin" role.
func superAdminAccessMiddleware(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userIDStr, ok := c.Get("user_id").(string)
			if !ok || userIDStr == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			}

			var user models.User
			if err := db.First(&user, "id = ?", userIDStr).Error; err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found"})
			}

			if user.Role != "SuperAdmin" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Access denied: SuperAdmin role required"})
			}

			return next(c)
		}
	}
}

// RegisterRoutes registers all admin API routes under /api/admin
func RegisterRoutes(g *echo.Group, handler *Handler, jwtSecret []byte, db *gorm.DB) {
	admin := g.Group("/admin")

	// JWT auth first, then Admin/SuperAdmin role check
	admin.Use(middleware.AuthMiddleware(jwtSecret))
	admin.Use(adminAccessMiddleware(db))

	// Stats
	admin.GET("/stats", handler.GetStats)

	// User management
	admin.GET("/users", handler.GetUsers)
	admin.POST("/users", handler.CreateUser)
	admin.PUT("/users/:id", handler.UpdateUser)
	admin.DELETE("/users/:id", handler.DeleteUser)

	// Document type management
	admin.GET("/document-types", handler.GetDocumentTypes)
	admin.POST("/document-types", handler.CreateDocumentType)
	admin.PUT("/document-types/:id", handler.UpdateDocumentType)
	admin.DELETE("/document-types/:id", handler.DeleteDocumentType)

	// School management (Admin only)
	admin.GET("/schools", handler.GetSchools)
	admin.PUT("/schools/:id", handler.UpdateSchool)

	// Role management (Subtree scoping validated in service layer)
	roles := admin.Group("/roles")
	roles.GET("", handler.GetRoles)
	roles.POST("", handler.CreateRole)
	roles.PUT("/:id", handler.UpdateRole)
	roles.DELETE("/:id", handler.DeleteRole)

	// Organization management (SuperAdmin and DHE Admins, scoped in service layer)
	orgs := admin.Group("/organizations")
	orgs.GET("", handler.GetOrganizations)
	orgs.POST("", handler.CreateOrganization)
	orgs.PUT("/:id", handler.UpdateOrganization)
	orgs.DELETE("/:id", handler.DeleteOrganization)

	// Peer connections (same-level role sharing)
	peers := admin.Group("/peer-connections")
	peers.GET("", handler.GetPeerConnections)
	peers.POST("", handler.RequestPeerConnection)
	peers.PUT("/:id/accept", handler.AcceptPeerConnection)
	peers.PUT("/:id/reject", handler.RejectPeerConnection)
	peers.PUT("/:id/revoke", handler.RevokePeerConnection)
}

