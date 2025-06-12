.PHONY: run build clean db-init db-migrate db-reset db-check db-status db-remove-fields db-add-messages help

# Default goal
.DEFAULT_GOAL := run

# Load environment variables from .env file
ifneq (,$(wildcard ./.env))
	include .env
	export
endif

# Database connection variables (with defaults)
DB_HOST ?= localhost
DB_PORT ?= 3306
DB_USER ?= root
DB_PASSWORD ?= 
DB_NAME ?= teamwork

# Run the application
run:
	@echo "Running Telegram bot..."
	go run ./cmd/bot/*.go

# Create .env from example if it doesn't exist
init:
	@if [ ! -f .env ]; then \
		echo "Creating .env file from example.env..."; \
		cp example.env .env; \
		echo "Please edit .env file and set your TELEGRAM_API_TOKEN and OPENAI_API_KEY"; \
	else \
		echo ".env file already exists"; \
	fi

# Build the application
build:
	@echo "Building Telegram bot..."
	go build -o telegram-bot ./cmd/bot/

# Build database utility
build-db:
	@echo "Building database utility..."
	go build -o db-tool ./cmd/db/

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f telegram-bot db-tool

# Initialize database with fresh schema (for new installations)
db-init:
	@echo "Initializing database schema..."
	go run ./cmd/db init
	@echo ""

# Run migration (for existing databases)
db-migrate:
	@echo "Running database migration..."
	go run ./cmd/db migrate
	@echo ""

# Remove priority and deadline fields from projects table
db-remove-fields:
	@echo "Removing priority and deadline fields from projects table..."
	go run ./cmd/db exec remove_priority_deadline.sql
	@echo ""

# Add messages table for conversation context
db-add-messages:
	@echo "Adding messages table for conversation context..."
	go run ./cmd/db exec add_messages_table.sql
	@echo ""

# Reset database (WARNING: This will delete all data!)
db-reset:
	@echo "Resetting database..."
	go run ./cmd/db reset
	@echo ""

# Check database connection
db-check:
	@echo "Checking database connection..."
	go run ./cmd/db check
	@echo ""

# Show database status
db-status:
	@echo "Showing database status..."
	go run ./cmd/db status
	@echo ""

# Open MySQL tunnel to production server
db-connect:
	ssh -L 3306:localhost:3306 root@prod

# Help command
help:
	@echo "Available commands:"
	@echo ""
	@echo "🚀 Application:"
	@echo "  make init      - Create .env file from example.env (if it doesn't exist)"
	@echo "  make run       - Run the bot using environment from .env"
	@echo "  make build     - Build the bot executable"
	@echo "  make build-db  - Build database utility executable"
	@echo "  make clean     - Clean build artifacts"
	@echo ""
	@echo "🗄️  Database:"
	@echo "  make db-init         - Initialize database schema (for new installations)"
	@echo "  make db-migrate      - Run database migration (for existing databases)"
	@echo "  make db-remove-fields - Remove priority and deadline fields from projects table"
	@echo "  make db-add-messages - Add messages table for conversation context"
	@echo "  make db-reset        - Reset database (⚠️  WARNING: deletes all data!)"
	@echo "  make db-check        - Check database connection"
	@echo "  make db-status       - Show database status and record counts"
	@echo ""
	@echo "🔧 Development:"
	@echo "  make db-connect  - Open MySQL tunnel to production server"
	@echo "  make help        - Show this help message"
	@echo ""
	@echo "Database connection uses environment variables:"
	@echo "  DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_NAME=$(DB_NAME)"
	@echo ""
	@echo "All database commands now use Go instead of mysql client" 