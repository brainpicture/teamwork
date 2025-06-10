# Simple Telegram Bot

A simple Telegram bot powered by AI that provides intelligent responses, project management, and stores user information in a MySQL database.

## Project Structure

```
teamwork/
├── cmd/
│   └── bot/               # Application entry point
│       ├── main.go       # Main application
│       └── config.go     # Configuration loading
├── internal/              # Internal packages (not importable externally)
│   ├── config.go         # Configuration struct
│   ├── db.go             # Database operations
│   ├── projects.go       # Project management functionality
│   ├── reply.go          # Message handling and replies
│   ├── ai.go             # AI provider interface and implementations
│   └── prompts.go        # AI prompts and system context
├── init.sql              # Database schema initialization
├── migrate.sql           # Database migration script
├── migrate_simple.sql    # Simple migration (drops data)
├── Makefile             # Build and run commands
├── go.mod               # Go module definition
└── README.md            # This file
```

This structure follows Go best practices:
- `cmd/` - Contains main applications for this project
- `internal/` - Contains private application and library code (cannot be imported by external projects)

## Prerequisites

- Go 1.21 or higher
- Telegram Bot Token (obtain from BotFather)
- OpenAI API Key (for AI features)
- MySQL database
- Make (optional, for using Makefile commands)

## Quick Start

1. **Clone and initialize**:
   ```bash
   git clone <repository>
   cd teamwork
   make init  # Creates .env file
   ```

2. **Configure environment** (edit `.env` file):
   ```bash
   # Required tokens
   TELEGRAM_API_TOKEN=your_telegram_bot_token_here
   OPENAI_API_KEY=your_openai_api_key_here
   
   # Database settings
   DB_HOST=localhost
   DB_PORT=3306
   DB_USER=root
   DB_PASSWORD=your_password
   DB_NAME=teamwork
   ```

3. **Setup database**:
   ```bash
   # For new installation
   make db-init
   
   # Or for existing database
   make db-migrate
   
   # Check connection
   make db-check
   ```

4. **Run the bot**:
   ```bash
   make run
   ```

## Database Management

### Available Commands

| Command | Description | Use Case |
|---------|-------------|----------|
| `make db-init` | Initialize fresh database | New installations |
| `make db-migrate` | Run migration scripts | Updating existing database |
| `make db-reset` | Reset database (⚠️ deletes data) | Development/testing |
| `make db-check` | Test database connection | Troubleshooting |
| `make db-status` | Show database status | Monitoring |
| `make db-shell` | Open MySQL shell | Manual operations |

### Migration Scenarios

**🆕 New Installation:**
```bash
make db-init
```

**🔄 Updating Existing Database:**
```bash
make db-migrate
```

**🧹 Development Reset:**
```bash
make db-reset  # Will ask for confirmation
```

**🔍 Check Everything is Working:**
```bash
make db-check
make db-status
```

## Manual Database Setup

If you prefer manual setup or make commands don't work:

### For New Installations:
```bash
mysql -u root -p < init.sql
```

### For Existing Databases:
```bash
mysql -u root -p < migrate.sql
```

### For Development Reset:
```bash
mysql -u root -p < migrate_simple.sql
```

## Environment Configuration

Create a `.env` file with your settings:

```bash
# Telegram Bot Configuration
TELEGRAM_API_TOKEN=your_telegram_bot_token_here

# AI Configuration
OPENAI_API_KEY=your_openai_api_key_here
AI_ENABLED=true

# Bot Settings
DEBUG_MODE=true
UPDATE_TIMEOUT=60

# Database Configuration
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_database_password
DB_NAME=teamwork
```

Or set environment variables manually:

```bash
# Required
export TELEGRAM_API_TOKEN="your_telegram_bot_token_here"
export OPENAI_API_KEY="your_openai_api_key_here"

# Optional AI settings
export AI_ENABLED="true"  # Default: true if OPENAI_API_KEY is set

# Optional Telegram settings
export DEBUG_MODE="true"  # Default: true (accepts true, 1, yes)
export UPDATE_TIMEOUT="60"  # Default: 60 seconds

# Database settings
export DB_HOST="localhost"
export DB_PORT="3306"
export DB_USER="root"
export DB_PASSWORD="your_password"
export DB_NAME="teamwork"
```

## Running the Bot

### Development:
```bash
make run
```

### Production:
```bash
make build
./telegram-bot
```

### Help:
```bash
make help  # Shows all available commands
```

## Features

### 🤖 AI-Powered Responses
- **ChatGPT Integration**: Uses OpenAI GPT-4o for intelligent responses
- **Context-Aware**: Maintains context about the development team and project
- **Fallback Support**: Gracefully falls back to static responses if AI is unavailable
- **Personalized Welcome**: AI-generated welcome messages for new users
- **Typing Indicator**: Shows "typing..." while AI generates responses for better UX

### 📋 Project Management
- **Create Projects**: Add new projects with title, description, priority, and deadline
- **Project Status**: Track project status (planning, active, paused, completed, cancelled)
- **Priority Levels**: Set priority levels (low, medium, high, urgent)
- **User Ownership**: Each user manages their own projects
- **Project Listing**: View all projects or filter by status
- **CRUD Operations**: Full create, read, update, delete functionality

### 👥 User Management
- **Automatic Registration**: New users are automatically added to the database
- **User Tracking**: Stores Telegram ID and display name
- **Welcome Messages**: Personalized greetings for new users and `/start` command

### 🗄️ Database Integration
- **MySQL Storage**: Persistent user data and project storage
- **Migration Support**: Easy database schema updates
- **User Profiles**: Tracks user information and interaction history
- **Relational Data**: Foreign key relationships between users and projects

### 🔧 Developer Features
- **Modular Architecture**: Clean separation of concerns
- **Provider Interface**: Easy to add new AI providers (Claude, Gemini, etc.)
- **Configurable**: Environment-based configuration
- **Logging**: Comprehensive logging for debugging
- **Timeout Handling**: Smart timeout management for AI requests

## Project Management Commands

The bot supports various commands for project management:

- `/projects` - Show all your projects
- `/project_add` - Add a new project (guided process)
- `/project_status` - Change project status
- `/project_delete` - Delete a project
- `/help` - Show available commands

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    tg_id BIGINT NOT NULL UNIQUE,
    tg_name VARCHAR(255),
    email VARCHAR(255),
    name VARCHAR(255),
    ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Projects Table
```sql
CREATE TABLE projects (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status ENUM('planning', 'active', 'paused', 'completed', 'cancelled'),
    priority ENUM('low', 'medium', 'high', 'urgent'),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deadline DATE NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

## AI Provider Architecture

The bot uses a provider interface that makes it easy to add new AI services:

```go
type AIProvider interface {
    GenerateResponse(ctx context.Context, prompt string) (string, error)
    GenerateWelcomeMessage(ctx context.Context, userName, status, timestamp string) (string, error)
    GenerateErrorMessage(ctx context.Context, errorContext string) (string, error)
}
```

Currently supported:
- **OpenAI GPT-4o** - Primary AI provider

Future providers can be easily added by implementing the `AIProvider` interface.

## Usage

- **Regular Messages**: Send any message to get an AI-powered response with typing indicator
- **New Users**: Automatically receive a personalized AI-generated welcome message
- **Start Command**: Send `/start` to get a welcome message anytime
- **Project Commands**: Use `/projects`, `/project_add`, etc. for project management
- **Fallback Mode**: If AI is disabled, the bot provides friendly static responses
- **Visual Feedback**: Typing indicator shows while AI is thinking (up to 30 seconds for regular messages, 15 seconds for welcome messages)

## Configuration Options

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `TELEGRAM_API_TOKEN` | Telegram Bot API token | - | ✅ |
| `OPENAI_API_KEY` | OpenAI API key for GPT-4o | - | For AI features |
| `AI_ENABLED` | Enable/disable AI features | `true` | No |
| `DEBUG_MODE` | Enable debug logging | `true` | No |
| `UPDATE_TIMEOUT` | Telegram update timeout | `60` | No |
| `DB_HOST` | Database host | `localhost` | No |
| `DB_PORT` | Database port | `3306` | No |
| `DB_USER` | Database username | `root` | No |
| `DB_PASSWORD` | Database password | `password` | No |
| `DB_NAME` | Database name | `teamwork` | No |

## Troubleshooting

### Database Connection Issues
```bash
make db-check  # Test connection
make db-status # Check database state
```

### Missing Environment Variables
```bash
make init      # Create .env file
# Edit .env with your actual values
```

### Build Issues
```bash
go mod tidy    # Update dependencies
make clean     # Clean build artifacts
make build     # Rebuild
``` 