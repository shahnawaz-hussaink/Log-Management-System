package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=localhost user=postgres password=110085 dbname=office_files port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Delete from tables in order to respect foreign keys, or just use CASCADE
	tables := []string{
		"file_documents",
		"workflow_histories",
		"noting_sheets",
		"notifications",
		"attachments",
		"files",
		"documents",
	}

	for _, table := range tables {
		err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE;", table)).Error
		if err != nil {
			fmt.Printf("Error truncating %s: %v\n", table, err)
		} else {
			fmt.Printf("Truncated %s\n", table)
		}
	}
}
