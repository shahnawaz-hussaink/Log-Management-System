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
	"office-file-sharing/backend/internal/shared/email"
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
	userHandler := user.NewHandler(userService, database)
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
			if err := db.Create(&warningNotif).Error; err == nil {
				go email.SendNotificationEmail(db, warningNotif.ID)
			}
 
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
			if err := db.Create(&principalNotif).Error; err == nil {
				go email.SendNotificationEmail(db, principalNotif.ID)
			}

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
		SchoolSlug   string
	}

	seedUsers := []seedUser{
		// vocational (Modern School)
		{Name: "Aarav Sharma", Email: "aarav@school.edu", Role: "vocational", ClassSection: "Department A", SchoolSlug: "modern-school"},
		{Name: "Ananya Iyer", Email: "ananya@school.edu", Role: "vocational", ClassSection: "Department B", SchoolSlug: "modern-school"},
		{Name: "Rohan Das", Email: "rohan@school.edu", Role: "vocational", ClassSection: "Department C", SchoolSlug: "modern-school"},
		{Name: "Kavya Menon", Email: "kavya@school.edu", Role: "vocational", ClassSection: "Department D", SchoolSlug: "modern-school"},
		// Teaching staff
		{Name: "Priya Patel", Email: "priya@school.edu", Role: "Teaching staff", ClassSection: "Department A", Subject: "Science", SchoolSlug: "greenwood-high"},
		{Name: "Neha Reddy", Email: "neha@school.edu", Role: "Teaching staff", ClassSection: "Department B", Subject: "History", SchoolSlug: "dps"},
		{Name: "Vikram Iyer", Email: "vikram@school.edu", Role: "Teaching staff", ClassSection: "Department C", Subject: "Mathematics", SchoolSlug: "modern-school"},
		{Name: "Meera Menon", Email: "meera@school.edu", Role: "Teaching staff", ClassSection: "Department D", Subject: "English", SchoolSlug: "modern-school"},
		// School Admins
		{Name: "Rahul Gupta", Email: "rahul@school.edu", Role: "School Admin", SchoolSlug: "greenwood-high"},
		{Name: "Gaurav Verma", Email: "gaurav@school.edu", Role: "School Admin", SchoolSlug: "dps"},
		{Name: "Shalini Sen", Email: "shalini@school.edu", Role: "School Admin", SchoolSlug: "modern-school"},
		// DHE (No school)
		{Name: "System Administrator", Email: "admin@school.edu", Role: "DHE", SchoolSlug: ""},
		// Non-teaching
		{Name: "Deepak Singh", Email: "deepak@school.edu", Role: "non-teaching", SchoolSlug: "greenwood-high"},
	}

	for _, su := range seedUsers {
		var existing models.User
		result := gormDB.Where("email = ?", su.Email).First(&existing)
		if result.Error != nil {
			var schoolID *uuid.UUID
			if su.SchoolSlug != "" {
				var sch models.School
				if err := gormDB.Where("slug = ?", su.SchoolSlug).First(&sch).Error; err == nil {
					schoolID = &sch.ID
				}
			}
			newUser := models.User{
				ID:           uuid.New(),
				Name:         su.Name,
				Email:        su.Email,
				PasswordHash: string(hash),
				Role:         su.Role,
				SchoolID:     schoolID,
				ClassSection: su.ClassSection,
				Subject:      su.Subject,
			}
			gormDB.Create(&newUser)
			log.Printf("Seeded user: %s (%s)", su.Name, su.Role)
		}
	}



	// 3. Seed Document Types
	docTypes := []models.DocumentType{
		{
			SchoolID:          school.ID,
			Name:              "Staff Grievance",
			Slug:              "staff-grievance",
			WorkflowStages:    `[{"stage": 1, "role": "Teaching staff", "label": "Department Head", "optional": false}]`,
			RequiredFields:    `[]`,
			SlaHours:          72,
			
		},
		{
			SchoolID:          school.ID,
			Name:              "Infrastructure Issue",
			Slug:              "infrastructure-issue",
			WorkflowStages:    `[{"stage": 1, "role": "School Admin", "label": "School Admin Final approval", "optional": false}]`,
			RequiredFields:    `["reason", "urgency"]`,
			SlaHours:          120,
			
		},
		{
			SchoolID:          school.ID,
			Name:              "Disciplinary Issue",
			Slug:              "disciplinary-issue",
			WorkflowStages:    `[{"stage": 1, "role": "Teaching staff", "label": "Department Head", "optional": false}]`,
			RequiredFields:    `["event_name", "event_date"]`,
			SlaHours:          24,
			
		},
		{
			SchoolID:          school.ID,
			Name:              "Audit Report",
			Slug:              "audit-report",
			WorkflowStages:    `[{"stage": 1, "role": "School Admin", "label": "School Admin Approval", "optional": false}]`,
			RequiredFields:    `["audit_reason", "percentage"]`,
			SlaHours:          96,
		},
		{
			SchoolID:          school.ID,
			Name:              "Official Circular",
			Slug:              "official-circular",
			WorkflowStages:    `[]`,
			RequiredFields:    `[]`,
			SlaHours:          0,
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
