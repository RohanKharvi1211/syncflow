package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"syncflow-backend/internal/config"

	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Build connection string
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.Port,
	)

	// Connect to database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("Connected to database successfully")
	fmt.Println("Running migrations...")

	// Get migration files
	migrationsDir := "migrations/postgres"
	files, err := ioutil.ReadDir(migrationsDir)
	if err != nil {
		log.Fatalf("Failed to read migrations directory: %v", err)
	}

	// Filter and sort migration files
	var migrationFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".up.sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}

	sort.Strings(migrationFiles)

	// Run migrations
	for _, fileName := range migrationFiles {
		filePath := filepath.Join(migrationsDir, fileName)
		fmt.Printf("Running migration: %s\n", fileName)

		// Read migration file
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", fileName, err)
		}

		// Execute migration
		_, err = db.Exec(string(content))
		if err != nil {
			// Check if error is because table already exists (for idempotency)
			if strings.Contains(err.Error(), "already exists") {
				fmt.Printf("  ⚠ Migration already applied (skipping): %s\n", fileName)
				continue
			}
			log.Fatalf("Failed to execute migration %s: %v", fileName, err)
		}

		fmt.Printf("  ✓ Migration completed: %s\n", fileName)
	}

	fmt.Println("\nAll migrations completed successfully!")
}

