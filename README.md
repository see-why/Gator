# Gator

A CLI tool for managing RSS feeds and users with PostgreSQL database integration.

## Overview

Gator is a command-line application built in Go that provides user management and RSS feed functionality. It features a robust CLI interface with database persistence using PostgreSQL and SQLC for type-safe database operations.

## Features

- **RSS Feed Management**: Add, follow, and unfollow RSS feeds
- **Post Aggregation**: Automatically fetch and store posts from followed feeds
- **Pagination**: Browse posts with user-friendly page-based navigation (5 posts per page)
- **User Management**: Register, login, and manage multiple users
- **PostgreSQL Integration**: Persistent data storage with migrations
- **CLI Interface**: Easy-to-use command-line interface with comprehensive commands
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

#### Feed Management

**Add a new RSS feed:**

```bash
gator addfeed <name> <url>
```

Creates a new feed and automatically follows it. Also fetches and saves recent posts.

**List all feeds:**

```bash
gator feeds
```

Shows all feeds in the database with their creators and URLs.

**Follow an existing feed:**

```bash
gator follow <url>
```

Start following an RSS feed that already exists in the database.

**List followed feeds:**

```bash
gator following
```

Shows all feeds you're currently following.

**Unfollow a feed:**

```bash
gator unfollow <url>
```

Stop following an RSS feed.

#### Post Browsing

**Browse posts with pagination:**

```bash
gator browse [page]
```

View posts from all followed feeds with pagination support. Shows 5 posts per page by default.

- `gator browse` - Shows page 1 (most recent posts)
- `gator browse 2` - Shows page 2 (older posts)
- `gator browse 3` - Shows page 3 (even older posts)

Posts are sorted by publication date (newest first) and numbered sequentially across pages. Navigation hints are provided to help you move between pages.

### Examples

#### Basic User Setup

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
```

#### Feed Management Workflow

```bash
# Add some RSS feeds
gator addfeed "Hacker News" "https://feeds.feedburner.com/hacker-news-feed-50"
gator addfeed "Ars Technica" "https://feeds.arstechnica.com/arstechnica/index"

# List all feeds in the system
gator feeds
# Output:
# * Hacker News (alice) - https://feeds.feedburner.com/hacker-news-feed-50
# * Ars Technica (alice) - https://feeds.arstechnica.com/arstechnica/index

# Follow an existing feed (if someone else added it)
gator follow "https://feeds.arstechnica.com/arstechnica/index"

# See what feeds you're following
gator following
# Output:
# You're following 2 feeds:
# * Hacker News
# * Ars Technica
```

#### Browsing Posts with Pagination

```bash
# Browse recent posts (page 1)
gator browse
# Output:
# Posts (page 1, showing 5 posts):
# 
# 1. Latest Tech News Article
#    Feed: Hacker News
#    Published: 2025-08-17 10:30:00
#    URL: https://example.com/article1
#
# 2. Another Interesting Article
#    Feed: Ars Technica
#    Published: 2025-08-17 09:15:00
#    URL: https://example.com/article2
#
# [... 3 more posts ...]
#
# To see more posts, run: gator browse 2

# Browse older posts (page 2)
gator browse 2
# Output:
# Posts (page 2, showing 5 posts):
#
# 6. Older Article Title
#    Feed: Hacker News
#    Published: 2025-08-16 18:45:00
#    URL: https://example.com/article6
#
# [... 4 more posts ...]
#
# To see more posts, run: gator browse 3
# To see previous posts, run: gator browse 1

# Navigate to a specific page
gator browse 5

# Reset the database if needed
gator reset
```

## Project Structure

```text
Gator/
├── main.go                    # Main CLI application
├── go.mod                     # Go module definition
├── sqlc.yaml                  # SQLC configuration
├── .env.example               # Environment variables template
├── internal/
│   ├── config/
│   │   └── config.go          # Configuration management
│   ├── database/              # Generated database code (SQLC)
│   │   ├── db.go
│   │   ├── models.go
│   │   ├── users.sql.go
│   │   └── feeds.sql.go
│   └── rss/
│       └── rss.go             # RSS feed fetching and parsing
└── sql/
    ├── queries/
    │   ├── users.sql          # User-related SQL queries
    │   └── feeds.sql          # Feed and post-related SQL queries
    └── schema/
        ├── 001_users.sql      # User table migration
        ├── 002_feeds.sql      # Feed table migration
        ├── 003_feed_follows.sql  # Feed follows table migration
        └── 004_posts.sql      # Posts table migration
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
