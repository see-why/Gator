package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gator/internal/config"
	"gator/internal/database"
	"gator/internal/rss"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// state holds application state
type state struct {
	db  *database.Queries
	cfg *config.Config
}

// command represents a CLI command and its arguments
type command struct {
	name string
	args []string
}

// commands holds all registered CLI commands
// The map is from command name to handler function
// Handler signature: func(*state, command) error
type commands struct {
	handlers map[string]func(*state, command) error
}

// middlewareLoggedIn is a higher-order function that wraps handlers requiring authentication
// It takes a handler that expects a user and returns a handler that can be registered
func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		// Get the current user from the database
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return fmt.Errorf("couldn't get current user: %w", err)
		}

		// Call the original handler with the user
		return handler(s, cmd, user)
	}
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.handlers[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	handler, ok := c.handlers[cmd.name]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}
	return handler(s, cmd)
}

// handlerLogin sets the current user in the config file
func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("login requires a username argument")
	}
	username := cmd.args[0]

	// Check if user exists in database
	_, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user '%s' not found", username)
		}
		return fmt.Errorf("database error while looking up user '%s': %w", username, err)
	}

	err = s.cfg.SetUser(username)
	if err != nil {
		return err
	}
	fmt.Printf("User set to '%s'\n", username)
	return nil
}

// handlerRegister creates a new user in the database
func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("register requires a username argument")
	}
	username := cmd.args[0]

	// Create new user in database
	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      username,
	})
	if err != nil {
		return fmt.Errorf("couldn't create user: %w", err)
	}

	// Set the current user in config
	err = s.cfg.SetUser(username)
	if err != nil {
		return err
	}

	fmt.Printf("User created successfully!\n")
	fmt.Printf("ID: %s\n", user.ID)
	fmt.Printf("Name: %s\n", user.Name)
	fmt.Printf("Created: %s\n", user.CreatedAt)
	return nil
}

// handlerReset deletes all users from the database
func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't reset users: %w", err)
	}

	fmt.Println("Database has been reset successfully!")
	return nil
}

// handlerUsers lists all users from the database
func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't retrieve users: %w", err)
	}

	for _, user := range users {
		if user.Name == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

// handlerAgg fetches a single feed and prints the entire struct to the console
func handlerAgg(s *state, cmd command) error {
	// If user asks to aggregate all feeds: `agg all [workers]`
	if len(cmd.args) >= 1 && cmd.args[0] == "all" {
		// Determine worker concurrency (optional second argument)
		workers := 5 // default
		if len(cmd.args) >= 2 {
			if w, err := strconv.Atoi(cmd.args[1]); err == nil && w > 0 {
				workers = w
			}
		}

		// Fetch all feeds from database
		feeds, err := s.db.GetFeedsWithUsers(context.Background())
		if err != nil {
			return fmt.Errorf("couldn't retrieve feeds: %w", err)
		}

		if len(feeds) == 0 {
			fmt.Println("No feeds found to aggregate.")
			return nil
		}

		// Create context with timeout for the aggregation operation
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		client := rss.NewHTTPClient()

		config := AggregationConfig{
			Workers: workers,
			Client:  client,
			DB:      s.db,
		}

		result := aggregateFeeds(ctx, feeds, config)
		fmt.Printf("Finished aggregating %d feeds. Processed ~%d posts.\n", len(feeds), result.TotalPosts)
		if result.FetchErrors > 0 || result.SaveErrors > 0 {
			fmt.Printf("Errors: %d fetch failures, %d save failures\n", result.FetchErrors, result.SaveErrors)
		}
		return nil
	}

	// Otherwise, fetch a single feed. Prefer explicit URL arg, then FEED_URL env.
	feedURL := ""
	if len(cmd.args) >= 1 {
		feedURL = cmd.args[0]
	}
	if feedURL == "" {
		feedURL = os.Getenv("FEED_URL")
	}
	if feedURL == "" {
		return fmt.Errorf("FEED_URL environment variable is not set and no URL argument provided")
	}

	// Create a reusable HTTP client with timeout configuration
	client := rss.NewHTTPClient()

	// Create context with timeout for the single feed fetch
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	feed, err := rss.FetchFeed(ctx, client, feedURL)
	if err != nil {
		return fmt.Errorf("couldn't fetch feed: %w", err)
	}

	// Print the entire struct to the console
	fmt.Printf("%+v\n", feed)
	return nil
}

// handlerAddFeed creates a new feed for the current user
func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("addfeed requires name and url arguments")
	}
	name := cmd.args[0]
	url := cmd.args[1]

	// Create new feed in database
	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})
	if err != nil {
		return fmt.Errorf("couldn't create feed: %w", err)
	}

	// Automatically create a feed follow record for the current user
	feedFollow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("couldn't create feed follow: %w", err)
	}

	// Print the fields of the new feed record
	fmt.Printf("Feed created successfully!\n")
	fmt.Printf("ID: %s\n", feed.ID)
	fmt.Printf("Name: %s\n", feed.Name)
	fmt.Printf("URL: %s\n", feed.Url)
	fmt.Printf("User ID: %s\n", feed.UserID)
	fmt.Printf("Created: %s\n", feed.CreatedAt)
	fmt.Printf("Updated: %s\n", feed.UpdatedAt)
	fmt.Printf("Now following %s as %s\n", feedFollow.FeedName, feedFollow.UserName)

	// Fetch and save posts from the feed
	fmt.Printf("Fetching posts from %s...\n", feed.Name)
	client := rss.NewHTTPClient()

	// Create context with timeout for feed fetching
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rssFeed, err := rss.FetchFeed(ctx, client, url)
	if err != nil {
		return fmt.Errorf("couldn't fetch RSS feed: %w", err)
	}

	err = rss.SavePostsToDatabase(ctx, s.db, rssFeed, feed.ID)
	if err != nil {
		return fmt.Errorf("couldn't save posts to database: %w", err)
	}

	fmt.Printf("Successfully processed %d posts from %s\n", len(rssFeed.Channel.Items), feed.Name)
	return nil
}

// handlerFeeds lists all feeds in the database with their associated user names
func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeedsWithUsers(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't retrieve feeds: %w", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found in the database.")
		return nil
	}

	for _, feed := range feeds {
		fmt.Printf("* %s (%s) - %s\n", feed.Name, feed.UserName, feed.Url)
	}

	return nil
}

// handlerFollow creates a new feed follow record for the current user
func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("follow requires a url argument")
	}
	url := cmd.args[0]

	// Look up the feed by URL
	feed, err := s.db.GetFeedByURL(context.Background(), url)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("feed not found with URL: %s", url)
		}
		return fmt.Errorf("database error while looking up feed with URL %s: %w", url, err)
	}

	// Create new feed follow record
	feedFollow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("couldn't create feed follow: %w", err)
	}

	// Print the feed name and current user
	fmt.Printf("Now following %s as %s\n", feedFollow.FeedName, feedFollow.UserName)
	return nil
}

// handlerFollowing lists all feeds the current user is following
func handlerFollowing(s *state, cmd command, user database.User) error {
	// Get all feed follows for the user
	feedFollows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("couldn't retrieve feed follows: %w", err)
	}

	if len(feedFollows) == 0 {
		fmt.Printf("You're not following any feeds yet.\n")
		return nil
	}

	fmt.Printf("You're following %d feeds:\n", len(feedFollows))
	for _, follow := range feedFollows {
		fmt.Printf("* %s\n", follow.FeedName)
	}

	return nil
}

// handlerUnfollow removes a feed follow record for the current user
func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("unfollow requires a url argument")
	}
	url := cmd.args[0]

	// Delete the feed follow record
	rowsAffected, err := s.db.DeleteFeedFollowByUserAndFeedURL(context.Background(), database.DeleteFeedFollowByUserAndFeedURLParams{
		UserID: user.ID,
		Url:    url,
	})
	if err != nil {
		return fmt.Errorf("couldn't unfollow feed: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("you're not following a feed with URL: %s", url)
	}

	fmt.Printf("Successfully unfollowed feed: %s\n", url)
	return nil
}

// handlerBrowse displays posts for the current user with pagination
func handlerBrowse(s *state, cmd command, user database.User) error {
	const postsPerPage = 5 // Number of posts to show per page
	page := int32(1)       // Default to page 1
	if len(cmd.args) >= 1 {
		var err error
		page, err = parsePageArg(cmd.args[0])
		if err != nil {
			return err
		}
	}

	// Calculate offset based on page number
	offset := (page - 1) * postsPerPage

	// Get posts for the user with pagination (query for one extra to check if more pages exist)
	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  postsPerPage + 1, // Query for one extra post
		Offset: offset,
	})
	if err != nil {
		return fmt.Errorf("couldn't retrieve posts: %w", err)
	}

	// Check if there are more pages available
	hasMorePages := len(posts) > int(postsPerPage)

	// If we got more than postsPerPage, trim the extra post
	if hasMorePages {
		posts = posts[:postsPerPage]
	}

	if len(posts) == 0 {
		if page == 1 {
			fmt.Printf("No posts found. Try following some feeds first!\n")
		} else {
			fmt.Printf("No posts found on page %d. Try a lower page number.\n", page)
		}
		return nil
	}

	fmt.Printf("Posts (page %d, showing %d posts):\n\n", page, len(posts))
	for i, post := range posts {
		// Calculate the overall post number based on page and position
		postNumber := offset + int32(i) + 1
		fmt.Printf("%d. %s\n", postNumber, post.Title)
		fmt.Printf("   Post ID: %s\n", post.ID)
		fmt.Printf("   Feed: %s\n", post.FeedName)
		if post.Description.Valid && post.Description.String != "" {
			// Truncate description if it's too long
			desc := post.Description.String
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			fmt.Printf("   %s\n", desc)
		}
		if post.PublishedAt.Valid {
			fmt.Printf("   Published: %s\n", post.PublishedAt.Time.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("   URL: %s\n", post.Url)
		fmt.Println()
	}

	// Show pagination info
	if hasMorePages {
		fmt.Printf("To see more posts, run: gator browse %d\n", page+1)
	}
	if page > 1 {
		fmt.Printf("To see previous posts, run: gator browse %d\n", page-1)
	}

	return nil
}

// handlerSearch searches posts for the current user by a fuzzy term (title/description)
func handlerSearch(s *state, cmd command, user database.User) error {
	const postsPerPage = 5
	if len(cmd.args) < 1 {
		return fmt.Errorf("search requires a search term")
	}

	query := cmd.args[0]
	page := int32(1)
	if len(cmd.args) >= 2 {
		var err error
		page, err = parsePageArg(cmd.args[1])
		if err != nil {
			return err
		}
	}

	offset := (page - 1) * postsPerPage

	// Create context with timeout for the search operation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Query for one extra to determine if more pages exist
	posts, err := s.db.SearchPostsForUser(ctx, database.SearchPostsForUserParams{
		UserID:  user.ID,
		Column2: sql.NullString{String: query, Valid: true},
		Limit:   postsPerPage + 1,
		Offset:  offset,
	})
	if err != nil {
		return fmt.Errorf("couldn't search posts: %w", err)
	}

	hasMorePages := len(posts) > int(postsPerPage)
	if hasMorePages {
		posts = posts[:postsPerPage]
	}

	if len(posts) == 0 {
		if page == 1 {
			fmt.Printf("No matching posts found for '%s'.\n", query)
		} else {
			fmt.Printf("No matching posts found on page %d for '%s'. Try a lower page number.\n", page, query)
		}
		return nil
	}

	fmt.Printf("Search results for '%s' (page %d, showing %d posts):\n\n", query, page, len(posts))
	for i, post := range posts {
		postNumber := offset + int32(i) + 1
		fmt.Printf("%d. %s\n", postNumber, post.Title)
		fmt.Printf("   Post ID: %s\n", post.ID)
		fmt.Printf("   Feed: %s\n", post.FeedName)
		if post.Description.Valid && post.Description.String != "" {
			desc := post.Description.String
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			fmt.Printf("   %s\n", desc)
		}
		if post.PublishedAt.Valid {
			fmt.Printf("   Published: %s\n", post.PublishedAt.Time.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("   URL: %s\n", post.Url)
		fmt.Println()
	}

	if hasMorePages {
		fmt.Printf("To see more results, run: gator search %s %d\n", query, page+1)
	}
	if page > 1 {
		fmt.Printf("To see previous results, run: gator search %s %d\n", query, page-1)
	}

	return nil
}

// handlerBookmark adds a post to the user's bookmarks
func handlerBookmark(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("bookmark requires a post ID argument")
	}

	// Parse the post ID
	postIDStr := cmd.args[0]
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		return fmt.Errorf("invalid post ID format: %s", postIDStr)
	}

	// Check if the post exists
	_, err = s.db.GetPostByID(context.Background(), postID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("post not found with ID: %s", postIDStr)
		}
		return fmt.Errorf("database error while looking up post: %w", err)
	}

	// Create the bookmark
	bookmark, err := s.db.CreateBookmark(context.Background(), database.CreateBookmarkParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		PostID:    postID,
	})
	if err != nil {
		// Check if the bookmark already exists
		count, checkErr := s.db.CheckBookmarkExists(context.Background(), database.CheckBookmarkExistsParams{
			UserID: user.ID,
			PostID: postID,
		})
		if checkErr == nil && count > 0 {
			return fmt.Errorf("post is already bookmarked")
		}
		return fmt.Errorf("couldn't create bookmark: %w", err)
	}

	fmt.Printf("Successfully bookmarked post %s\n", postID)
	fmt.Printf("Bookmark ID: %s\n", bookmark.ID)
	fmt.Printf("Bookmarked at: %s\n", bookmark.CreatedAt.Format("2006-01-02 15:04:05"))
	return nil
}

// handlerUnbookmark removes a post from the user's bookmarks
func handlerUnbookmark(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("unbookmark requires a post ID argument")
	}

	// Parse the post ID
	postIDStr := cmd.args[0]
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		return fmt.Errorf("invalid post ID format: %s", postIDStr)
	}

	// Delete the bookmark
	rowsAffected, err := s.db.DeleteBookmark(context.Background(), database.DeleteBookmarkParams{
		UserID: user.ID,
		PostID: postID,
	})
	if err != nil {
		return fmt.Errorf("couldn't remove bookmark: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post %s is not bookmarked", postIDStr)
	}

	fmt.Printf("Successfully removed bookmark for post %s\n", postID)
	return nil
}

// handlerBookmarks displays all bookmarked posts for the current user with pagination
func handlerBookmarks(s *state, cmd command, user database.User) error {
	const postsPerPage = 5 // Number of bookmarks to show per page
	page := int32(1)       // Default to page 1
	if len(cmd.args) >= 1 {
		var err error
		page, err = parsePageArg(cmd.args[0])
		if err != nil {
			return err
		}
	}

	// Calculate offset based on page number
	offset := (page - 1) * postsPerPage

	// Get bookmarks for the user with pagination (query for one extra to check if more pages exist)
	bookmarks, err := s.db.GetBookmarksForUser(context.Background(), database.GetBookmarksForUserParams{
		UserID: user.ID,
		Limit:  postsPerPage + 1, // Query for one extra bookmark
		Offset: offset,
	})
	if err != nil {
		return fmt.Errorf("couldn't retrieve bookmarks: %w", err)
	}

	// Check if there are more pages available
	hasMorePages := len(bookmarks) > int(postsPerPage)

	// If we got more than postsPerPage, trim the extra bookmark
	if hasMorePages {
		bookmarks = bookmarks[:postsPerPage]
	}

	if len(bookmarks) == 0 {
		if page == 1 {
			fmt.Printf("No bookmarked posts found. Try bookmarking some posts first!\n")
		} else {
			fmt.Printf("No bookmarked posts found on page %d. Try a lower page number.\n", page)
		}
		return nil
	}

	fmt.Printf("Bookmarked posts (page %d, showing %d posts):\n\n", page, len(bookmarks))
	for i, bookmark := range bookmarks {
		// Calculate the overall bookmark number based on page and position
		bookmarkNumber := offset + int32(i) + 1
		fmt.Printf("%d. %s\n", bookmarkNumber, bookmark.Title)
		fmt.Printf("   Post ID: %s\n", bookmark.ID)
		fmt.Printf("   Feed: %s\n", bookmark.FeedName)
		if bookmark.Description.Valid && bookmark.Description.String != "" {
			// Truncate description if it's too long
			desc := bookmark.Description.String
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			fmt.Printf("   %s\n", desc)
		}
		if bookmark.PublishedAt.Valid {
			fmt.Printf("   Published: %s\n", bookmark.PublishedAt.Time.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("   Bookmarked: %s\n", bookmark.BookmarkedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("   URL: %s\n", bookmark.Url)
		fmt.Println()
	}

	// Show pagination info
	if hasMorePages {
		fmt.Printf("To see more bookmarks, run: gator bookmarks %d\n", page+1)
	}
	if page > 1 {
		fmt.Printf("To see previous bookmarks, run: gator bookmarks %d\n", page-1)
	}

	return nil
}

// parsePageArg parses a page argument string and returns a validated int32 page number.
func parsePageArg(s string) (int32, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("page must be a number, got: %s", s)
	}
	if i < 1 {
		return 0, fmt.Errorf("page must be 1 or greater, got: %d", i)
	}
	return int32(i), nil
}

// PostView is a lightweight view of a post used by rendering functions and tests.
type PostView struct {
	ID          uuid.UUID
	Title       string
	Url         string
	FeedName    string
	Description sql.NullString
	PublishedAt sql.NullTime
}

// convert functions
func toPostViewsFromGet(rows []database.GetPostsForUserRow) []PostView {
	out := make([]PostView, 0, len(rows))
	for _, r := range rows {
		out = append(out, PostView{
			ID:          r.ID,
			Title:       r.Title,
			Url:         r.Url,
			FeedName:    r.FeedName,
			Description: r.Description,
			PublishedAt: r.PublishedAt,
		})
	}
	return out
}

func toPostViewsFromSearch(rows []database.SearchPostsForUserRow) []PostView {
	out := make([]PostView, 0, len(rows))
	for _, r := range rows {
		out = append(out, PostView{
			ID:          r.ID,
			Title:       r.Title,
			Url:         r.Url,
			FeedName:    r.FeedName,
			Description: r.Description,
			PublishedAt: r.PublishedAt,
		})
	}
	return out
}

// renderPosts renders a page of posts into a string, returning the output.
func renderPosts(views []PostView, page int32, postsPerPage int) string {
	offset := (page - 1) * int32(postsPerPage)
	hasMore := false
	if len(views) > postsPerPage {
		hasMore = true
		views = views[:postsPerPage]
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Posts (page %d, showing %d posts):\n\n", page, len(views)))
	for i, post := range views {
		postNumber := offset + int32(i) + 1
		b.WriteString(fmt.Sprintf("%d. %s\n", postNumber, post.Title))
		b.WriteString(fmt.Sprintf("   Post ID: %s\n", post.ID))
		b.WriteString(fmt.Sprintf("   Feed: %s\n", post.FeedName))
		if post.Description.Valid && post.Description.String != "" {
			desc := post.Description.String
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			b.WriteString(fmt.Sprintf("   %s\n", desc))
		}
		if post.PublishedAt.Valid {
			b.WriteString(fmt.Sprintf("   Published: %s\n", post.PublishedAt.Time.Format("2006-01-02 15:04:05")))
		}
		b.WriteString(fmt.Sprintf("   URL: %s\n\n", post.Url))
	}
	if hasMore {
		b.WriteString(fmt.Sprintf("To see more posts, run: gator browse %d\n", page+1))
	}
	if page > 1 {
		b.WriteString(fmt.Sprintf("To see previous posts, run: gator browse %d\n", page-1))
	}
	return b.String()
}

// AggregationConfig holds configuration for concurrent feed aggregation
type AggregationConfig struct {
	Workers int
	Fetch   func(ctx context.Context, client *http.Client, url string) (*rss.RSSFeed, error)
	Save    func(ctx context.Context, db *database.Queries, feed *rss.RSSFeed, feedID uuid.UUID) error
	Client  *http.Client
	DB      *database.Queries
}

// AggregationResult holds the results of feed aggregation
type AggregationResult struct {
	FeedsProcessed int
	TotalPosts     int
	FetchErrors    int
	SaveErrors     int
}

// validateConfig ensures the aggregation config has valid settings
func validateConfig(config *AggregationConfig) {
	if config.Workers <= 0 {
		config.Workers = 1
	}
	if config.Fetch == nil {
		config.Fetch = rss.FetchFeed
	}
	if config.Save == nil {
		config.Save = rss.SavePostsToDatabase
	}
}

// processFeed processes a single feed and updates shared counters
func processFeed(ctx context.Context, feedURL string, feedID uuid.UUID, config *AggregationConfig, mu *sync.Mutex, result *AggregationResult) {
	rssFeed, err := config.Fetch(ctx, config.Client, feedURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching feed %s: %v\n", feedURL, err)
		mu.Lock()
		result.FetchErrors++
		mu.Unlock()
		return
	}

	// attempt to save and track errors
	if err := config.Save(ctx, config.DB, rssFeed, feedID); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving posts from feed %s: %v\n", feedURL, err)
		mu.Lock()
		result.SaveErrors++
		mu.Unlock()
		return
	}

	mu.Lock()
	result.FeedsProcessed++
	result.TotalPosts += len(rssFeed.Channel.Items)
	mu.Unlock()
}

// aggregateFeeds concurrently fetches and saves posts for the provided feeds.
// Returns the number of feeds processed and total posts processed.
func aggregateFeeds(ctx context.Context, feeds []database.GetFeedsWithUsersRow, config AggregationConfig) AggregationResult {
	validateConfig(&config)

	sem := make(chan struct{}, config.Workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	result := AggregationResult{}

	for _, f := range feeds {
		wg.Add(1)
		sem <- struct{}{}

		go func(feed database.GetFeedsWithUsersRow) {
			defer wg.Done()
			defer func() { <-sem }()

			processFeed(ctx, feed.Url, feed.ID, &config, &mu, &result)
		}(f)
	}

	wg.Wait()
	return result
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		// Don't exit if .env file doesn't exist, just log a warning
		fmt.Fprintf(os.Stderr, "Warning: Could not load .env file: %v\n", err)
	}

	// Read the config file
	cfg, err := config.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		os.Exit(1)
	}

	// Open database connection
	db, err := sql.Open("postgres", cfg.DbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	dbQueries := database.New(db)
	appState := &state{
		db:  dbQueries,
		cfg: &cfg,
	}

	cmds := &commands{handlers: make(map[string]func(*state, command) error)}
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmds.register("browse", middlewareLoggedIn(handlerBrowse))
	cmds.register("search", middlewareLoggedIn(handlerSearch))
	cmds.register("bookmark", middlewareLoggedIn(handlerBookmark))
	cmds.register("unbookmark", middlewareLoggedIn(handlerUnbookmark))
	cmds.register("bookmarks", middlewareLoggedIn(handlerBookmarks))

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: not enough arguments. Usage: gator <command> [args...]")
		os.Exit(1)
	}

	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]
	cmd := command{name: cmdName, args: cmdArgs}

	err = cmds.run(appState, cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
