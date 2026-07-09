package main

import (
	"log"

	"office-file-sharing/backend/internal/db"
	"office-file-sharing/backend/internal/handlers"
	"office-file-sharing/backend/internal/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	db.InitDB()

	seedData()

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	api := e.Group("/api")

	// Public routes
	api.POST("/auth/login", handlers.Login)

	// Protected routes
	r := api.Group("")
	r.Use(handlers.AuthMiddleware)

	r.GET("/users", handlers.GetUsers)
	r.POST("/documents", handlers.UploadDocument)
	r.GET("/documents", handlers.GetDocuments)
	r.GET("/documents/:id", handlers.GetDocumentDetails)
	r.GET("/documents/:id/download", handlers.DownloadDocument)
	r.PUT("/documents/:id/replace", handlers.ReplaceDocument)
	r.POST("/documents/:id/action", handlers.DocumentAction)

	log.Fatal(e.Start(":8080"))
}

func seedData() {
	var count int64
	db.DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("Failed to hash default password:", err)
		}
		users := []models.User{
			{Name: "Alice Smith", Email: "alice@office.com", PasswordHash: string(hash)},
			{Name: "Bob Jones", Email: "bob@office.com", PasswordHash: string(hash)},
			{Name: "Charlie Brown", Email: "charlie@office.com", PasswordHash: string(hash)},
		}
		for _, u := range users {
			u.ID = uuid.New()
			db.DB.Create(&u)
		}
		log.Println("Database seeded with test users.")
	} else {
		// Update legacy dummy password hashes to bcrypt hashes
		var dummyUsers []models.User
		db.DB.Where("password_hash = ?", "dummy").Find(&dummyUsers)
		if len(dummyUsers) > 0 {
			hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
			db.DB.Model(&models.User{}).Where("password_hash = ?", "dummy").Update("password_hash", string(hash))
			log.Println("Updated legacy test users with hashed passwords.")
		}
	}
}
