package main

import (
	"log"
	"net/http"
	"strings"
	"time"

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

	// Services
	authService := auth.NewService(authRepo, []byte(cfg.JWTSecret))
	userService := user.NewService(userRepo)
	docService := document.NewService(docRepo, "./uploads")

	// Handlers
	authHandler := auth.NewHandler(authService)
	userHandler := user.NewHandler(userService)
	docHandler := document.NewHandler(docService)

	// Register Modular Routes
	auth.RegisterRoutes(api, authHandler)
	user.RegisterRoutes(api, userHandler)
	document.RegisterRoutes(api, docHandler, []byte(cfg.JWTSecret))

	log.Println("Modular Academic Monolith starting on port :8080...")
	log.Fatal(e.Start(":8080"))
}

func seedData(gormDB *gorm.DB) {
	// GORM DB instance is required for seed queries
	var count int64
	gormDB.Model(&models.User{}).Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("Failed to hash default password:", err)
		}
		users := []models.User{
			{Name: "Alice Smith", Email: "alice@office.com", PasswordHash: string(hash), Role: "Student"},
			{Name: "Bob Jones", Email: "bob@office.com", PasswordHash: string(hash), Role: "Faculty"},
			{Name: "Charlie Brown", Email: "charlie@office.com", PasswordHash: string(hash), Role: "Administrator"},
		}
		for _, u := range users {
			u.ID = uuid.New()
			gormDB.Create(&u)
		}
		log.Println("Database seeded with test users and academic roles.")
	} else {
		// Update legacy dummy password hashes to bcrypt hashes
		var dummyUsers []models.User
		gormDB.Where("password_hash = ?", "dummy").Find(&dummyUsers)
		if len(dummyUsers) > 0 {
			hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
			gormDB.Model(&models.User{}).Where("password_hash = ?", "dummy").Update("password_hash", string(hash))
			log.Println("Updated legacy test users with hashed passwords.")
		}
	}
}
