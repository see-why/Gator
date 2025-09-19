# Gator

A CLI tool and HTTP API for managing RSS feeds and users with PostgreSQL database integration.

## Overview

Gator is a command-line application and HTTP API server built in Go that provides user management and RSS feed functionality. It features both a robust CLI interface and a comprehensive REST API with database persistence using PostgreSQL and SQLC for type-safe database operations.

## Features

- **RSS Feed Management**: Add, follow, and unfollow RSS feeds
- **Post Aggregation**: Automatically fetch and store posts from followed feeds
- **Pagination**: Browse posts with user-friendly page-based navigation (5 posts per page)
- **Search Functionality**: Fuzzy search through post titles and descriptions
- **Bookmarking**: Save and manage favorite posts for later reading
- **Liking**: Like posts to show appreciation (separate from bookmarks)
- **User Management**: Register, login, and manage multiple users
- **PostgreSQL Integration**: Persistent data storage with migrations
- **CLI Interface**: Easy-to-use command-line interface with comprehensive commands
- **HTTP API**: RESTful API with authentication for remote access
- **API Authentication**: Secure API key-based authentication system
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

This gives you access to both the CLI interface and the HTTP API server functionality.

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

**Start HTTP API server:**

```bash
gator serve [port]
```

Starts the HTTP API server for remote access. See the [HTTP API](#http-api) section for detailed usage.

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

#### Search Posts

**Search posts by fuzzy match (title or description):**

```bash
gator search <term> [page]
```

- `gator search rust` - Searches titles and descriptions for "rust" and shows page 1 of results.
- `gator search "openai" 2` - Shows page 2 of search results for "openai".

Notes:

- Search is case-insensitive and performs a fuzzy substring match using ILIKE on both title and description.
- Pagination matches the `browse` command: 5 results per page and navigation hints when more results exist.
- Use quotes for multi-word search terms, e.g. `gator search "machine learning"`.

#### Bookmark Management

**Bookmark a post for later reading:**

```bash
gator bookmark <post_id>
```

**Remove a bookmark:**

```bash
gator unbookmark <post_id>
```

**View all bookmarked posts:**

```bash
gator bookmarks [page]
```

- `gator bookmarks` - Shows page 1 of your bookmarked posts
- `gator bookmarks 2` - Shows page 2 of bookmarked posts

Notes:

- Post IDs are displayed when browsing or searching posts
- Bookmarks are sorted by bookmark creation date (newest first)
- Each bookmark shows when it was bookmarked and the original publication date
- Pagination works the same as browse and search commands (5 posts per page)

**Like a post to show appreciation:**

```bash
gator like <post_id>
```

**Remove a like from a post:**

```bash
gator unlike <post_id>
```

**View all your liked posts:**

```bash
gator likes [page]
```

- `gator likes` - Shows page 1 of your liked posts
- `gator likes 2` - Shows page 2 of liked posts

Notes:

- Likes are separate from bookmarks - you can like without bookmarking and vice versa
- Liked posts are sorted by like creation date (newest first)
- Each liked post shows when you liked it and the original publication date
- Pagination works the same as other commands (5 posts per page)

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

#### Bookmark Workflow

```bash
# Browse posts to find something interesting
gator browse
# Output:
# Posts (page 1, showing 5 posts):
# 
# 1. Interesting AI Article
#    Post ID: 550e8400-e29b-41d4-a716-446655440000
#    Feed: Tech News
#    Published: 2025-08-22 10:30:00
#    URL: https://example.com/ai-article
#
# 2. Another Great Article
#    Post ID: 6ba7b810-9dad-11d1-80b4-00c04fd430c8
#    Feed: Science Daily
#    Published: 2025-08-22 09:15:00
#    URL: https://example.com/science-article

# Bookmark the AI article for later reading
gator bookmark 550e8400-e29b-41d4-a716-446655440000
# Output:
# Successfully bookmarked post 550e8400-e29b-41d4-a716-446655440000
# Bookmark ID: 7ca7b820-8ead-22f2-90c5-11d04fd430d9
# Bookmarked at: 2025-08-22 14:30:00

# Search for more articles about AI
gator search "artificial intelligence"
# Bookmark another interesting post
gator bookmark 6ba7b810-9dad-11d1-80b4-00c04fd430c8

# View all your bookmarked posts
gator bookmarks
# Output:
# Bookmarked posts (page 1, showing 2 posts):
#
# 1. Another Great Article
#    Post ID: 6ba7b810-9dad-11d1-80b4-00c04fd430c8
#    Feed: Science Daily
#    Published: 2025-08-22 09:15:00
#    Bookmarked: 2025-08-22 14:31:00
#    URL: https://example.com/science-article
#
# 2. Interesting AI Article
#    Post ID: 550e8400-e29b-41d4-a716-446655440000
#    Feed: Tech News
#    Published: 2025-08-22 10:30:00
#    Bookmarked: 2025-08-22 14:30:00
#    URL: https://example.com/ai-article

# Remove a bookmark when done reading
gator unbookmark 550e8400-e29b-41d4-a716-446655440000
# Output:
# Successfully removed bookmark for post 550e8400-e29b-41d4-a716-446655440000
```

## HTTP API

Gator includes a comprehensive HTTP API that provides remote access to all RSS feed functionality through REST endpoints.

### Starting the API Server

**Start the HTTP server:**

```bash
gator serve [port]
```

- `gator serve` - Starts server on port 8080 (default)
- `gator serve 3000` - Starts server on port 3000
- Server also respects the `PORT` environment variable

The server provides:
- Health check endpoint: `http://localhost:8080/health`
- Interactive API documentation: `http://localhost:8080/api/docs`

### Authentication

The API uses API key-based authentication. All endpoints (except health check and documentation) require authentication.

**Authentication Header Format:**
```
Authorization: ApiKey <your_api_key_here>
```

### API Endpoints

#### Authentication Endpoints

**Register a new user and get API key:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name": "username"}'
```

**Login with existing user:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"name": "username"}'
```

Both endpoints return:
```json
{
  "user": {"id": "uuid", "name": "username"},
  "api_key": "64-character-hex-string"
}
```

#### User Management

**Get all users:**
```bash
curl -H "Authorization: ApiKey <api_key>" \
  http://localhost:8080/api/users
```

#### Feed Management

**Get all feeds:**
```bash
curl -H "Authorization: ApiKey <api_key>" \
  http://localhost:8080/api/feeds
```

**Create a new feed:**
```bash
curl -X POST http://localhost:8080/api/feeds \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey <api_key>" \
  -d '{"name": "Feed Name", "url": "https://example.com/feed.xml"}'
```

#### Feed Following

**Get user's followed feeds:**
```bash
curl -H "Authorization: ApiKey <api_key>" \
  http://localhost:8080/api/feed-follows
```

**Follow a feed:**
```bash
curl -X POST http://localhost:8080/api/feed-follows \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey <api_key>" \
  -d '{"feed_url": "https://example.com/feed.xml"}'
```

**Unfollow a feed:**
```bash
curl -X DELETE http://localhost:8080/api/feed-follows \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey <api_key>" \
  -d '{"feed_url": "https://example.com/feed.xml"}'
```

#### Posts

**Get posts from followed feeds (with pagination):**
```bash
curl -H "Authorization: ApiKey <api_key>" \
  "http://localhost:8080/api/posts?page=1&limit=10"
```

**Search posts:**
```bash
curl -H "Authorization: ApiKey <api_key>" \
  "http://localhost:8080/api/posts/search?q=search_term&page=1&limit=10"
```

#### Bookmarks

**Get user's bookmarks:**
```bash
curl -H "Authorization: ApiKey <api_key>" \
  "http://localhost:8080/api/bookmarks?page=1&limit=10"
```

**Bookmark a post:**
```bash
curl -X POST http://localhost:8080/api/bookmarks \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey <api_key>" \
  -d '{"post_id": "post-uuid-here"}'
```

**Remove bookmark:**
```bash
curl -X DELETE http://localhost:8080/api/bookmarks/{post_id} \
  -H "Authorization: ApiKey <api_key>"
```

#### Likes

**Get user's liked posts:**
```bash
curl -H "Authorization: ApiKey <api_key>" \
  "http://localhost:8080/api/likes?page=1&limit=10"
```

**Like a post:**
```bash
curl -X POST http://localhost:8080/api/likes \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey <api_key>" \
  -d '{"post_id": "post-uuid-here"}'
```

**Remove like:**
```bash
curl -X DELETE http://localhost:8080/api/likes/{post_id} \
  -H "Authorization: ApiKey <api_key>"
```

### API Response Format

All API responses use JSON format. Successful responses return the requested data, while errors return:

```json
{
  "error": "Error message describing what went wrong"
}
```

### Pagination

List endpoints support pagination with query parameters:
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 10, max: 100)

Example: `?page=2&limit=20`

### Interactive Documentation

Visit `http://localhost:8080/api/docs` when the server is running to see interactive API documentation with example requests and responses.

## Project Structure

```text
Gator/
├── main.go                    # Main CLI application
├── go.mod                     # Go module definition
├── sqlc.yaml                  # SQLC configuration
├── .env.example               # Environment variables template
├── internal/
│   ├── api/
│   │   ├── auth.go            # Authentication middleware and handlers
│   │   ├── docs.go            # Interactive API documentation
│   │   ├── handlers.go        # All REST endpoint handlers
│   │   └── server.go          # HTTP server setup and routing
│   ├── config/
│   │   └── config.go          # Configuration management
│   ├── database/              # Generated database code (SQLC)
│   │   ├── db.go
│   │   ├── models.go
│   │   ├── users.sql.go
│   │   ├── feeds.sql.go
│   │   └── bookmarks.sql.go
│   └── rss/
│       └── rss.go             # RSS feed fetching and parsing
└── sql/
    ├── queries/
    │   ├── users.sql          # User-related SQL queries
    │   ├── feeds.sql          # Feed and post-related SQL queries
    │   └── bookmarks.sql      # Bookmark-related SQL queries
    └── schema/
        ├── 001_users.sql      # User table migration
        ├── 002_feeds.sql      # Feed table migration
        ├── 003_feed_follows.sql  # Feed follows table migration
        ├── 004_posts.sql      # Posts table migration
        ├── 005_bookmarks.sql  # Bookmarks table migration
        └── 006_api_keys.sql   # API keys migration
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
