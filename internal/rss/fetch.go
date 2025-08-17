package rss

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"gator/internal/database"

	"github.com/google/uuid"
)

// NewHTTPClient creates a new HTTP client with proper timeout configuration
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

// FetchFeed fetches a feed from the given URL and returns a filled-out RSSFeed struct
func FetchFeed(ctx context.Context, client *http.Client, feedURL string) (*RSSFeed, error) {
	// Validate that the client has a reasonable timeout
	if client.Timeout == 0 {
		return nil, fmt.Errorf("HTTP client must have a timeout configured")
	}
	// Create a new request with context
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}

	// Set User-Agent header to identify our program
	req.Header.Set("User-Agent", "gator")

	// Make the request using the provided client
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal the XML into RSSFeed struct
	var feed RSSFeed
	err = xml.Unmarshal(body, &feed)
	if err != nil {
		return nil, err
	}

	// Decode HTML entities in channel title and description
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	// Decode HTML entities in item titles and descriptions
	for i := range feed.Channel.Items {
		feed.Channel.Items[i].Title = html.UnescapeString(feed.Channel.Items[i].Title)
		feed.Channel.Items[i].Description = html.UnescapeString(feed.Channel.Items[i].Description)
	}

	return &feed, nil
}

// SavePostsToDatabase saves the posts from an RSS feed to the database
func SavePostsToDatabase(ctx context.Context, db *database.Queries, feed *RSSFeed, feedID uuid.UUID) error {
	for _, item := range feed.Channel.Items {
		// Parse the published date
		publishedAt, err := parsePubDate(item.PubDate)
		if err != nil {
			log.Printf("Warning: Could not parse published date for post '%s': %v", item.Title, err)
			// Continue with other posts even if one has a bad date
		}

		// Create the post
		_, err = db.CreatePost(ctx, database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
			Title:       item.Title,
			Url:         item.Link,
			Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
			PublishedAt: sql.NullTime{Time: publishedAt, Valid: !publishedAt.IsZero()},
			FeedID:      feedID,
		})

		if err != nil {
			// Check if it's a unique constraint violation (post already exists)
			if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
				// Ignore duplicate posts - this is expected
				continue
			}
			// Log other errors but continue processing other posts
			log.Printf("Error saving post '%s': %v", item.Title, err)
			continue
		}
	}

	return nil
}

// parsePubDate attempts to parse various date formats commonly found in RSS feeds
func parsePubDate(pubDate string) (time.Time, error) {
	if pubDate == "" {
		return time.Time{}, fmt.Errorf("empty published date")
	}

	// Common RSS date formats
	formats := []string{
		time.RFC1123Z,         // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC1123,          // "Mon, 02 Jan 2006 15:04:05 MST"
		time.RFC3339,          // "2006-01-02T15:04:05Z07:00"
		"2006-01-02 15:04:05", // "2006-01-02 15:04:05"
		"2006-01-02T15:04:05", // "2006-01-02T15:04:05"
		"2006-01-02",          // "2006-01-02"
	}

	for _, format := range formats {
		if t, err := time.Parse(format, strings.TrimSpace(pubDate)); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", pubDate)
}
