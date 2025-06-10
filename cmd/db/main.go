package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"telegram-bot/internal"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "init":
		initDatabase()
	case "migrate":
		migrateDatabase()
	case "reset":
		resetDatabase()
	case "check":
		checkConnection()
	case "status":
		showStatus()
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Database management utility")
	fmt.Println()
	fmt.Println("Usage: go run ./cmd/db <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  init     - Initialize database schema (for new installations)")
	fmt.Println("  migrate  - Run database migration (for existing databases)")
	fmt.Println("  reset    - Reset database (WARNING: deletes all data!)")
	fmt.Println("  check    - Check database connection")
	fmt.Println("  status   - Show database status and record counts")
	fmt.Println("  help     - Show this help message")
}

func initDatabase() {
	fmt.Println("Initializing database schema...")

	config := internal.LoadConfigForDB()
	if err := executeSQLFile(config, "init.sql"); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	fmt.Println("✅ Database initialized successfully")
}

func migrateDatabase() {
	fmt.Println("Running database migration...")

	config := internal.LoadConfigForDB()
	if err := executeSQLFile(config, "migrate.sql"); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	fmt.Println("✅ Migration completed successfully")
}

func resetDatabase() {
	fmt.Println("⚠️  WARNING: This will delete ALL data in the database!")
	fmt.Print("Are you sure? Type 'yes' to continue: ")

	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "yes" {
		fmt.Println("Operation cancelled")
		return
	}

	fmt.Println("Resetting database...")

	config := internal.LoadConfigForDB()
	if err := executeSQLFile(config, "migrate_simple.sql"); err != nil {
		log.Fatalf("Failed to reset database: %v", err)
	}

	fmt.Println("✅ Database reset completed")
}

func checkConnection() {
	fmt.Println("Checking database connection...")

	config := internal.LoadConfigForDB()
	db, err := internal.ConnectDB(config)
	if err != nil {
		fmt.Printf("❌ Database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("✅ Database connection successful")
}

func showStatus() {
	fmt.Println("Database Status:")

	config := internal.LoadConfigForDB()
	fmt.Printf("Host: %s:%d\n", config.DBHost, config.DBPort)
	fmt.Printf("User: %s\n", config.DBUser)
	fmt.Printf("Database: %s\n", config.DBName)
	fmt.Println()

	db, err := internal.ConnectDB(config)
	if err != nil {
		fmt.Printf("❌ Cannot connect to database: %v\n", err)
		return
	}
	defer db.Close()

	// Get table counts
	tables := []string{"users", "projects"}
	for _, table := range tables {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			fmt.Printf("❌ Error counting %s: %v\n", table, err)
		} else {
			fmt.Printf("📊 %s: %d records\n", table, count)
		}
	}
}

func executeSQLFile(config *internal.Config, filename string) error {
	// Read SQL file
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read SQL file %s: %v", filename, err)
	}

	// Connect to database
	db, err := internal.ConnectDB(config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Split SQL content by semicolons and execute each statement
	sqlContent := string(content)
	statements := strings.Split(sqlContent, ";")

	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		_, err := db.Exec(statement)
		if err != nil {
			return fmt.Errorf("failed to execute SQL statement: %v\nStatement: %s", err, statement)
		}
	}

	return nil
}
