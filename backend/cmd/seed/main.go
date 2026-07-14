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

	// Clear existing users and cascade child references
	tx := gormDB.Exec("TRUNCATE TABLE users CASCADE;")
	if tx.Error != nil {
		log.Fatalf("Failed to truncate users table: %v", tx.Error)
	}

	// 1. Resolve or seed Greenwood High School
	var school models.School
	err := gormDB.First(&school, "slug = ?", "greenwood-high").Error
	if err != nil {
		school = models.School{
			ID:   uuid.New(),
			Name: "Greenwood High School",
			Slug: "greenwood-high",
		}
		gormDB.Create(&school)
		log.Println("Seeded school: Greenwood High School")
	}

	// 2. Hash default password
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash default password:", err)
	}

	// 3. Define users
	users := []models.User{
		{Name: "Alice Smith", Email: "alice@school.edu", PasswordHash: string(hash), Role: "Student", SchoolID: &school.ID, ClassSection: "10-A"},
		{Name: "Bob Johnson", Email: "bob@school.edu", PasswordHash: string(hash), Role: "Teacher", SchoolID: &school.ID, ClassSection: "10-A", Subject: "Science"},
		{Name: "Charlie Brown", Email: "charlie@school.edu", PasswordHash: string(hash), Role: "Principal", SchoolID: &school.ID},
		{Name: "David Smith", Email: "david@school.edu", PasswordHash: string(hash), Role: "Parent", SchoolID: &school.ID},
		{Name: "Diana Prince", Email: "diana@school.edu", PasswordHash: string(hash), Role: "Teacher", SchoolID: &school.ID, ClassSection: "10-B", Subject: "History"},
		{Name: "Evan Wright", Email: "evan@school.edu", PasswordHash: string(hash), Role: "Teacher", SchoolID: &school.ID, ClassSection: "10-C", Subject: "Mathematics"},
		{Name: "Fiona Gallagher", Email: "fiona@school.edu", PasswordHash: string(hash), Role: "Teacher", SchoolID: &school.ID, ClassSection: "10-D", Subject: "English"},
		{Name: "George Vance", Email: "george@school.edu", PasswordHash: string(hash), Role: "Principal", SchoolID: &school.ID},
		{Name: "System Administrator", Email: "admin@school.edu", PasswordHash: string(hash), Role: "Admin", SchoolID: &school.ID},
	}

	for i := range users {
		users[i].ID = uuid.New()
		gormDB.Create(&users[i])
	}
	log.Println("Seeded school-scoped users.")

	// Establish Parent-Child link (David is Alice's parent)
	var alice, david models.User
	gormDB.First(&alice, "email = ?", "alice@school.edu")
	gormDB.First(&david, "email = ?", "david@school.edu")
	if alice.ID != uuid.Nil && david.ID != uuid.Nil {
		pc := models.ParentChild{
			ParentID: david.ID,
			ChildID:  alice.ID,
		}
		gormDB.Create(&pc)
		log.Println("Established Parent-Child relationship: David -> Alice")
	}

	// 4. Ensure document types are seeded
	var docTypeCount int64
	gormDB.Model(&models.DocumentType{}).Count(&docTypeCount)
	if docTypeCount == 0 {
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
				Name:              "Permission Slip",
				Slug:              "permission-slip",
				WorkflowStages:    `[{"stage": 1, "role": "Teacher", "label": "Class Teacher", "optional": false}]`,
				RequiredFields:    `["event_name", "event_date"]`,
				SlaHours:          24,
				NeedsParentCosign: false,
			},
			{
				SchoolID:          school.ID,
				Name:              "Transfer Certificate Request",
				Slug:              "transfer-certificate-request",
				WorkflowStages:    `[{"stage": 1, "role": "Teacher", "label": "Class Teacher", "optional": false}, {"stage": 2, "role": "Principal", "label": "Principal Final approval", "optional": false}]`,
				RequiredFields:    `["reason", "last_date"]`,
				SlaHours:          120,
				NeedsParentCosign: false,
			},
			{
				SchoolID:          school.ID,
				Name:              "Fee Concession Form",
				Slug:              "fee-concession-form",
				WorkflowStages:    `[{"stage": 1, "role": "Principal", "label": "Principal Approval", "optional": false}]`,
				RequiredFields:    `["concession_reason", "concession_percentage"]`,
				SlaHours:          96,
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
			docTypes[i].ID = uuid.New()
			gormDB.Create(&docTypes[i])
		}
		log.Println("Database seeded with document types.")
	}

	log.Println("Database seeding completed successfully.")
}
