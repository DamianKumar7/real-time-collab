package config

import (
	"fmt"
	"log"
	"os"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DbConnection *gorm.DB

// Initialize the database connection
func InitDb() *gorm.DB {
    // Database connection details
    connectionDetails := fmt.Sprintf(
        "host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
        os.Getenv("DB_HOST"),    // Read from environment variable
        os.Getenv("DB_USER"),    // Read from environment variable
        os.Getenv("DB_PASSWORD"),// Read from environment variable
        os.Getenv("DB_NAME"),    // Read from environment variable
        os.Getenv("DB_PORT"),    // Read from environment variable
    )

    // Open the database connection using GORM
    db, err := gorm.Open(postgres.Open(connectionDetails), &gorm.Config{})
    if err != nil {
        log.Fatalf("Error connecting to the database: %v", err)
    }

    log.Println("Successfully connected to the database")
    return db
}
