package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"office-file-sharing/backend/internal/admin"
	"office-file-sharing/backend/internal/auth"
	"office-file-sharing/backend/internal/document"
	"office-file-sharing/backend/internal/shared/config"
	"office-file-sharing/backend/internal/shared/db"
	"office-file-sharing/backend/internal/shared/models"
	"office-file-sharing/backend/internal/user"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	database := db.Init(cfg.DatabaseURL)
	seedData(database)

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Rate limiter specifically for authentication endpoints to prevent brute force
	authRateLimiter := middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			return !strings.HasPrefix(c.Request().URL.Path, "/api/auth/")
		},
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(5.0 / 60.0), // 5 requests per minute
				Burst:     5,
				ExpiresIn: 3 * time.Minute,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			return ctx.RealIP(), nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusTooManyRequests, map[string]string{"error": "Too many requests. Please try again later."})
		},
	})

	e.Use(authRateLimiter)

	api := e.Group("/api")

	// Repositories
	authRepo := auth.NewRepository(database)
	userRepo := user.NewRepository(database)
	docRepo := document.NewRepository(database)
	adminRepo := admin.NewRepository(database)

	// Services
	authService := auth.NewService(authRepo, []byte(cfg.JWTSecret))
	userService := user.NewService(userRepo)
	docService := document.NewService(docRepo, "./uploads")
	adminService := admin.NewService(adminRepo)

	// Handlers
	authHandler := auth.NewHandler(authService)
	userHandler := user.NewHandler(userService)
	docHandler := document.NewHandler(docService)
	adminHandler := admin.NewHandler(adminService)

	// Register Modular Routes
	auth.RegisterRoutes(api, authHandler)
	user.RegisterRoutes(api, userHandler, []byte(cfg.JWTSecret))
	document.RegisterRoutes(api, docHandler, []byte(cfg.JWTSecret))
	admin.RegisterRoutes(api, adminHandler, []byte(cfg.JWTSecret), database)

	// Start background SLA auto-escalation job
	go startSLAScheduler(database)

	log.Println("Modular Academic Monolith starting on port :8080...")
	log.Fatal(e.Start(":8080"))
}

func startSLAScheduler(db *gorm.DB) {
	ticker := time.NewTicker(30 * time.Second)
	log.Println("Background SLA monitoring worker started successfully.")
	for range ticker.C {
		var breachedDocs []models.Document
		now := time.Now()

		// Fetch documents pending approval with breached deadlines
		err := db.Preload("School").
			Where("status = ? AND sla_deadline IS NOT NULL AND sla_deadline < ?", models.StatusPendingApproval, now).
			Find(&breachedDocs).Error
		if err != nil {
			log.Printf("SLA Worker Error: Failed to query breached documents: %v", err)
			continue
		}

		for _, doc := range breachedDocs {
			// Find the Principal of the School to escalate to
			var principal models.User
			errP := db.First(&principal, "school_id = ? AND role = 'Principal'", doc.SchoolID).Error
			if errP != nil {
				log.Printf("SLA Worker Warning: No Principal found for school %v to escalate document %s", doc.SchoolID, doc.UniqueNumber)
				continue
			}

			// Capture old owner for notification
			oldOwnerID := doc.CurrentOwnerID

			oldDeadlineStr := "Unknown"
			if doc.SlaDeadline != nil {
				oldDeadlineStr = doc.SlaDeadline.Format(time.RFC822)
			}

			// Update document state: escalate to Principal
			doc.CurrentOwnerID = principal.ID
			// Push SLA deadline out (e.g. Principal has 48 more hours to act)
			newDeadline := time.Now().Add(48 * time.Hour)
			doc.SlaDeadline = &newDeadline
			db.Save(&doc)

			// Log SLA breach event in WorkflowHistory audit timeline
			history := models.WorkflowHistory{
				ID:         uuid.New(),
				SchoolID:   doc.SchoolID,
				DocumentID: doc.ID,
				ActorID:    uuid.Nil, // System action
				TargetID:   &principal.ID,
				Action:     "Escalated",
				Remarks:    fmt.Sprintf("SLA breached (deadline was %s). Auto-escalated to Principal.", oldDeadlineStr),
				ActorRole:  "System",
				Stage:      doc.CurrentStage,
				Version:    doc.Version,
				EventType:  "sla_breach",
			}
			db.Create(&history)

			// Update stage pending approver status
			db.Model(&models.DocumentPendingApprover{}).
				Where("document_id = ? AND stage = ? AND status = 'Pending'", doc.ID, doc.CurrentStage).
				Updates(map[string]interface{}{"status": "Escalated"})

			// Create new pending approver record for Principal
			nextApprover := models.DocumentPendingApprover{
				ID:         uuid.New(),
				DocumentID: doc.ID,
				UserID:     principal.ID,
				Stage:      doc.CurrentStage,
				Status:     "Pending",
			}
			db.Create(&nextApprover)

			// Queue warning notifications in DB
			notifPayload := fmt.Sprintf(`{"document_title": "%s", "message": "SLA warning: Document %s has been escalated due to inaction."}`, doc.Title, doc.UniqueNumber)
			warningNotif := models.Notification{
				ID:          uuid.New(),
				SchoolID:    doc.SchoolID,
				RecipientID: oldOwnerID,
				DocumentID:  &doc.ID,
				Channel:     "email",
				Template:    "sla_warning",
				Payload:     notifPayload,
				Status:      "pending",
			}
			db.Create(&warningNotif)

			principalNotif := models.Notification{
				ID:          uuid.New(),
				SchoolID:    doc.SchoolID,
				RecipientID: principal.ID,
				DocumentID:  &doc.ID,
				Channel:     "email",
				Template:    "action_required",
				Payload:     fmt.Sprintf(`{"document_title": "%s", "message": "Escalated document %s requires your attention."}`, doc.Title, doc.UniqueNumber),
				Status:      "pending",
			}
			db.Create(&principalNotif)

			log.Printf("SLA Worker: Auto-escalated document %s (%s) to Principal %s due to SLA breach.", doc.UniqueNumber, doc.Title, principal.Name)
		}
	}
}

func seedData(gormDB *gorm.DB) {
	// 1. Seed School
	var schoolCount int64
	gormDB.Model(&models.School{}).Count(&schoolCount)
	var school models.School
	if schoolCount == 0 {
		school = models.School{
			ID:   uuid.New(),
			Name: "Greenwood High School",
			Slug: "greenwood-high",
		}
		gormDB.Create(&school)
		log.Println("Seeded school: Greenwood High School")
	} else {
		gormDB.First(&school)
	}

	// 2. Seed Users (idempotent — checks each email individually)
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash default password:", err)
	}

	type seedUser struct {
		Name         string
		Email        string
		Role         string
		ClassSection string
		Subject      string
	}

	seedUsers := []seedUser{
		// 4 Students
		{Name: "Alice Smith", Email: "alice@school.edu", Role: "Student", ClassSection: "10-A"},
		{Name: "Brian Lee", Email: "brian@school.edu", Role: "Student", ClassSection: "10-B"},
		{Name: "Chloe Davis", Email: "chloe@school.edu", Role: "Student", ClassSection: "10-C"},
		{Name: "Daniel Roy", Email: "daniel@school.edu", Role: "Student", ClassSection: "10-D"},
		// 4 Teachers
		{Name: "Bob Johnson", Email: "bob@school.edu", Role: "Teacher", ClassSection: "10-A", Subject: "Science"},
		{Name: "Diana Prince", Email: "diana@school.edu", Role: "Teacher", ClassSection: "10-B", Subject: "Mathematics"},
		{Name: "Evan Wright", Email: "evan@school.edu", Role: "Teacher", ClassSection: "10-C", Subject: "Mathematics"},
		{Name: "Fiona Gallagher", Email: "fiona@school.edu", Role: "Teacher", ClassSection: "10-D", Subject: "English"},
		// Principals
		{Name: "Charlie Brown", Email: "charlie@school.edu", Role: "Principal"},
		{Name: "George Vance", Email: "george@school.edu", Role: "Principal"},
		// 1 Admin (school-level admin)
		{Name: "System Administrator", Email: "admin@school.edu", Role: "Admin"},
		// 1 Parent (kept for parent-child relationship)
		{Name: "David Smith", Email: "david@school.edu", Role: "Parent"},
	}

	for _, su := range seedUsers {
		var existing models.User
		result := gormDB.Where("email = ?", su.Email).First(&existing)
		if result.Error != nil {
			// Not found — create
			newUser := models.User{
				ID:           uuid.New(),
				Name:         su.Name,
				Email:        su.Email,
				PasswordHash: string(hash),
				Role:         su.Role,
				SchoolID:     &school.ID,
				ClassSection: su.ClassSection,
				Subject:      su.Subject,
			}
			gormDB.Create(&newUser)
			log.Printf("Seeded user: %s (%s)", su.Name, su.Role)
		}
	}

	// Establish Parent-Child link (David → Alice)
	var alice, david models.User
	gormDB.First(&alice, "email = ?", "alice@school.edu")
	gormDB.First(&david, "email = ?", "david@school.edu")
	if alice.ID != uuid.Nil && david.ID != uuid.Nil {
		var pcCount int64
		gormDB.Model(&models.ParentChild{}).Where("parent_id = ? AND child_id = ?", david.ID, alice.ID).Count(&pcCount)
		if pcCount == 0 {
			pc := models.ParentChild{ParentID: david.ID, ChildID: alice.ID}
			gormDB.Create(&pc)
			log.Println("Established Parent-Child relationship: David → Alice")
		}
	}

	// 3. Seed Document Types
	docTypes := []models.DocumentType{
		{
			SchoolID:          school.ID,
			Name:              "Assignment",
			Slug:              "assignment",
			WorkflowStages:    `[{"stage": 1, "role": "Teacher", "label": "Subject Teacher", "optional": false}]`,
			RequiredFields:    `[]`,
			SlaHours:          72,
			NeedsParentCosign: false,
		},
		{
			SchoolID:          school.ID,
			Name:              "Leave Application",
			Slug:              "leave-application",
			WorkflowStages:    `[{"stage": 1, "role": "Teacher", "label": "Class Teacher", "optional": false}, {"stage": 2, "role": "Principal", "label": "Principal Approval", "optional": true, "condition": "leave_days > 3"}]`,
			RequiredFields:    `["from_date", "to_date", "reason", "leave_days"]`,
			SlaHours:          48,
			NeedsParentCosign: false,
		},
		{
			SchoolID:          school.ID,
			Name:              "Report",
			Slug:              "report",
			WorkflowStages:    `[{"stage": 1, "role": "Teacher", "label": "Class Teacher", "optional": false}]`,
			RequiredFields:    `[]`,
			SlaHours:          72,
			NeedsParentCosign: false,
		},
		{
			SchoolID:          school.ID,
			Name:              "Permission Slip",
			Slug:              "permission-slip",
			WorkflowStages:    `[{"stage": 1, "role": "Teacher", "label": "Class Teacher", "optional": false}]`,
			RequiredFields:    `["event_name", "event_date"]`,
			SlaHours:          24,
			NeedsParentCosign: false,
		},
		{
			SchoolID:          school.ID,
			Name:              "General Request Letter",
			Slug:              "general-request-letter",
			WorkflowStages:    `[{"stage": 1, "role": "Teacher", "label": "Teacher Acknowledgement", "optional": false}]`,
			RequiredFields:    `[]`,
			SlaHours:          120,
			NeedsParentCosign: false,
		},
		{
			SchoolID:          school.ID,
			Name:              "Circular",
			Slug:              "circular",
			WorkflowStages:    `[]`,
			RequiredFields:    `[]`,
			SlaHours:          0,
			NeedsParentCosign: false,
		},
	}
	for i := range docTypes {
		var existing models.DocumentType
		if err := gormDB.Where("slug = ?", docTypes[i].Slug).First(&existing).Error; err != nil {
			docTypes[i].ID = uuid.New()
			gormDB.Create(&docTypes[i])
			log.Printf("Seeded missing document type: %s", docTypes[i].Name)
		}
	}
}
