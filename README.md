# Gator

A CLI tool for managing RSS feeds and users with PostgreSQL database integration.

## Overview

Gator is a command-line application built in Go that provides user management and RSS feed functionality. It features a robust CLI interface with database persistence using PostgreSQL and SQLC for type-safe database operations.

## Features

- **User Management**: Register, login, and manage users
- **PostgreSQL Integration**: Persistent data storage with migrations
- **CLI Interface**: Easy-to-use command-line interface
- **Type-safe Database Operations**: Uses SQLC for generated Go code
- **Configuration Management**: JSON-based configuration system

## Prerequisites

Before using Gator, you'll need to have the following installed:

### Go

- Go 1.19 or later
- Install from [https://golang.org/dl/](https://golang.org/dl/)

### PostgreSQL

- PostgreSQL 15 or later

**macOS (using Homebrew):**

```bash
brew install postgresql@15
brew services start postgresql@15
```

**Linux/WSL (Debian/Ubuntu):**

```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo service postgresql start
```

### Database Setup

1. Connect to PostgreSQL:

   ```bash
   # macOS
   psql postgres
   
   # Linux
   sudo -u postgres psql
   ```

2. Create the gator database:

   ```sql
   CREATE DATABASE gator;
   \c gator
   ```

3. Exit psql:

   ```sql
   \q
   ```

## Installation

Install the Gator CLI using Go:

```bash
go install github.com/see-why/Gator@latest
```

## Configuration

Create a configuration file at `~/.gatorconfig.json`:

```json
{
  "db_url": "postgres://username:@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}
```

**Note:** Replace `username` with your system username. On Linux, you might need:

```json
{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}
```

## Database Migrations

Run the database migrations to set up the required tables:

```bash
cd sql/schema
goose postgres "your_connection_string_here" up
```

## Usage

### Available Commands

#### User Management

**Register a new user:**

```bash
gator register <username>
```

**Login as an existing user:**

```bash
gator login <username>
```

**List all users:**

```bash
gator users
```

Shows all registered users with the current user marked as `(current)`.

**Reset all users:**

```bash
gator reset
```

Deletes all users from the database.

### Examples

```bash
# Register a new user
gator register alice

# Login as alice
gator login alice

# List all users
gator users
# Output:
# * alice (current)
# * bob
# * charlie

# Reset the database
gator reset
```

## Project Structure

```text
Gator/
├── main.go                    # Main CLI application
├── go.mod                     # Go module definition
├── sqlc.yaml                  # SQLC configuration
├── internal/
│   ├── config/
│   │   └── config.go          # Configuration management
│   └── database/              # Generated database code (SQLC)
│       ├── db.go
│       ├── models.go
│       └── users.sql.go
└── sql/
    ├── queries/
    │   └── users.sql          # SQL queries
    └── schema/
        └── 001_users.sql      # Database migrations
```

## Development

### Building from Source

```bash
git clone https://github.com/see-why/Gator.git
cd Gator
go build -o gator main.go
```

### Running Tests

```bash
go test ./...
```

### Database Operations

**Generate Go code from SQL queries:**

```bash
sqlc generate
```

**Run database migrations:**

```bash
cd sql/schema
goose postgres "your_connection_string" up
```

**Rollback migrations:**

```bash
cd sql/schema
goose postgres "your_connection_string" down
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests
5. Submit a pull request

## License

This project is open source and available under the [MIT License](LICENSE).
