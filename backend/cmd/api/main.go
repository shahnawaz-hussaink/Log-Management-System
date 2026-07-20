package main

import (
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
	userHandler := user.NewHandler(userService, database)
	docHandler := document.NewHandler(docService)
	adminHandler := admin.NewHandler(adminService)

	// Register Modular Routes
	auth.RegisterRoutes(api, authHandler)
	user.RegisterRoutes(api, userHandler, []byte(cfg.JWTSecret))
	document.RegisterRoutes(api, docHandler, []byte(cfg.JWTSecret))
	admin.RegisterRoutes(api, adminHandler, []byte(cfg.JWTSecret), database)



	log.Println("Modular Academic Monolith starting on port :8080...")
	log.Fatal(e.Start(":8080"))
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



	// 3. Seed Document/Receipt Types for all schools
	var schools []models.School
	gormDB.Find(&schools)

	for _, s := range schools {
		docTypes := []models.DocumentType{
			{
				SchoolID:       s.ID,
				Name:           "Staff Grievance",
				Slug:           "staff-grievance",
				WorkflowStages: `[{"stage": 1, "role": "Teaching staff", "label": "Department Head", "optional": false}]`,
				RequiredFields: `[]`,
			},
			{
				SchoolID:       s.ID,
				Name:           "Infrastructure Issue",
				Slug:           "infrastructure-issue",
				WorkflowStages: `[{"stage": 1, "role": "School Admin", "label": "School Admin Final approval", "optional": false}]`,
				RequiredFields: `["reason", "urgency"]`,
			},
			{
				SchoolID:       s.ID,
				Name:           "Disciplinary Issue",
				Slug:           "disciplinary-issue",
				WorkflowStages: `[{"stage": 1, "role": "Teaching staff", "label": "Department Head", "optional": false}]`,
				RequiredFields: `["event_name", "event_date"]`,
			},
			{
				SchoolID:       s.ID,
				Name:           "Audit Report",
				Slug:           "audit-report",
				WorkflowStages: `[{"stage": 1, "role": "School Admin", "label": "School Admin Approval", "optional": false}]`,
				RequiredFields: `["audit_reason", "percentage"]`,
			},
		}

		// Ensure Official Circular category is deleted from database
		gormDB.Where("slug = ?", "official-circular").Delete(&models.DocumentType{})

		for i := range docTypes {
			var existing models.DocumentType
			if err := gormDB.Where("school_id = ? AND slug = ?", s.ID, docTypes[i].Slug).First(&existing).Error; err != nil {
				docTypes[i].ID = uuid.New()
				gormDB.Create(&docTypes[i])
				log.Printf("Seeded missing document type for school %s: %s", s.Name, docTypes[i].Name)
			}
		}
	}
}
