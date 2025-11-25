package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/naozine/project_crud_with_auth_tmpl/db"
	"github.com/naozine/project_crud_with_auth_tmpl/internal/database"
)

func main() {
	// Flags
	adminEmail := flag.String("email", "", "Email for the initial admin user")
	adminName := flag.String("name", "Admin", "Name for the initial admin user")
	flag.Parse()

	if *adminEmail == "" {
		log.Fatal("Usage: go run cmd/setup/main.go -email <email> [-name <name>]")
	}

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or could not be loaded, using OS environment variables.")
	}

	// Database Setup
	conn, err := sql.Open("sqlite3", "file:app.db?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		log.Fatal("Failed to connect to app.db:", err)
	}
	defer conn.Close()

	// Apply Schema
	if err := applySchema(conn); err != nil {
		log.Fatal("Failed to apply schema:", err)
	}
	fmt.Println("Schema applied successfully.")

	// Create Admin User
	q := database.New(conn)
	_, err = q.CreateUser(context.Background(), database.CreateUserParams{
		Email:    *adminEmail,
		Name:     *adminName,
		Role:     "admin",
		IsActive: true,
	})
	if err != nil {
		log.Printf("Failed to create admin user (maybe already exists?): %v", err)
	} else {
		fmt.Printf("Admin user %s (%s) created successfully.\n", *adminName, *adminEmail)
	}
}

func applySchema(conn *sql.DB) error {
	schema, err := db.SchemaFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	_, err = conn.Exec(string(schema))
	return err
}
