package main

import (
	"log"

	"office-file-sharing/backend/internal/shared/config"
	"office-file-sharing/backend/internal/shared/db"
	"office-file-sharing/backend/internal/shared/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.Load()
	gormDB := db.Init(cfg.DatabaseURL)

	log.Println("Resetting and seeding database accounts...")

	// Clear existing data thoroughly across all tables
	tables := []string{"schools", "users", "document_types", "documents", "workflow_histories", "notifications", "attachments", "document_pending_approvers", "files", "notes"}
	for _, t := range tables {
		tx := gormDB.Exec("TRUNCATE TABLE " + t + " CASCADE;")
		if tx.Error != nil {
			log.Printf("Warning: failed to truncate table %s (might not exist yet): %v", t, tx.Error)
		}
	}

	// 1. Seed Schools
	school1 := models.School{ID: uuid.New(), Name: "Greenwood High School", Slug: "greenwood-high"}
	school2 := models.School{ID: uuid.New(), Name: "Delhi Public School", Slug: "dps"}
	school3 := models.School{ID: uuid.New(), Name: "Modern School", Slug: "modern-school"}
	if err := gormDB.Create(&school1).Error; err != nil {
		log.Fatalf("Failed to create school1: %v", err)
	}
	if err := gormDB.Create(&school2).Error; err != nil {
		log.Fatalf("Failed to create school2: %v", err)
	}
	if err := gormDB.Create(&school3).Error; err != nil {
		log.Fatalf("Failed to create school3: %v", err)
	}
	log.Println("Seeded schools: Greenwood High, DPS, Modern School")

	// 2. Hash default password
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash default password:", err)
	}

	// 3. Define users
	users := []models.User{
		{Name: "System Administrator", Email: "admin@school.edu", PasswordHash: string(hash), Role: "DHE", SchoolID: nil},
		
		// Greenwood High School
		{Name: "Rahul Gupta", Email: "rahul@school.edu", PasswordHash: string(hash), Role: "School Admin", SchoolID: &school1.ID},
		{Name: "Priya Patel", Email: "priya@school.edu", PasswordHash: string(hash), Role: "Teaching staff", SchoolID: &school1.ID, ClassSection: "Department A", Subject: "Science"},
		{Name: "Deepak Singh", Email: "deepak@school.edu", PasswordHash: string(hash), Role: "non-teaching", SchoolID: &school1.ID},
		
		// Delhi Public School
		{Name: "Gaurav Verma", Email: "gaurav@school.edu", PasswordHash: string(hash), Role: "School Admin", SchoolID: &school2.ID},
		{Name: "Neha Reddy", Email: "neha@school.edu", PasswordHash: string(hash), Role: "Teaching staff", SchoolID: &school2.ID, ClassSection: "Department B", Subject: "History"},
		
		// Modern School
		{Name: "Shalini Sen", Email: "shalini@school.edu", PasswordHash: string(hash), Role: "School Admin", SchoolID: &school3.ID},
		{Name: "Vikram Iyer", Email: "vikram@school.edu", PasswordHash: string(hash), Role: "Teaching staff", SchoolID: &school3.ID, ClassSection: "Department C", Subject: "Mathematics"},
		{Name: "Meera Menon", Email: "meera@school.edu", PasswordHash: string(hash), Role: "Teaching staff", SchoolID: &school3.ID, ClassSection: "Department D", Subject: "English"},
		{Name: "Aarav Sharma", Email: "aarav@school.edu", PasswordHash: string(hash), Role: "vocational", SchoolID: &school3.ID, ClassSection: "Department A"},
		{Name: "Ananya Iyer", Email: "ananya@school.edu", PasswordHash: string(hash), Role: "vocational", SchoolID: &school3.ID, ClassSection: "Department B"},
		{Name: "Rohan Das", Email: "rohan@school.edu", PasswordHash: string(hash), Role: "vocational", SchoolID: &school3.ID, ClassSection: "Department C"},
		{Name: "Kavya Menon", Email: "kavya@school.edu", PasswordHash: string(hash), Role: "vocational", SchoolID: &school3.ID, ClassSection: "Department D"},
	}

	for i := range users {
		users[i].ID = uuid.New()
		gormDB.Create(&users[i])
	}
	log.Println("Seeded users across multiple schools.")

	// 4. Ensure document types are seeded for all schools
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
		for i := range docTypes {
			var existing models.DocumentType
			if err := gormDB.Where("school_id = ? AND slug = ?", s.ID, docTypes[i].Slug).First(&existing).Error; err != nil {
				docTypes[i].ID = uuid.New()
				gormDB.Create(&docTypes[i])
				log.Printf("Seeded missing document type for school %s: %s", s.Name, docTypes[i].Name)
			}
		}
	}

	log.Println("Database seeding completed successfully.")
}
