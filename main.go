package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gator/internal/config"
	"gator/internal/database"
	"gator/internal/rss"
	"os"
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
	feedURL := os.Getenv("FEED_URL")
	if feedURL == "" {
		return fmt.Errorf("FEED_URL environment variable is not set")
	}

	// Create a reusable HTTP client with timeout configuration
	client := rss.NewHTTPClient()

	feed, err := rss.FetchFeed(context.Background(), client, feedURL)
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
	rssFeed, err := rss.FetchFeed(context.Background(), client, url)
	if err != nil {
		return fmt.Errorf("couldn't fetch RSS feed: %w", err)
	}

	err = rss.SavePostsToDatabase(context.Background(), s.db, rssFeed, feed.ID)
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

// handlerBrowse displays posts for the current user
func handlerBrowse(s *state, cmd command, user database.User) error {
	limit := int32(2) // default limit

	if len(cmd.args) >= 1 {
		// Try to parse the limit argument
		if parsedLimit, err := fmt.Sscanf(cmd.args[0], "%d", &limit); err != nil || parsedLimit != 1 {
			return fmt.Errorf("limit must be a number, got: %s", cmd.args[0])
		}
	}

	// Get posts for the user
	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  limit,
	})
	if err != nil {
		return fmt.Errorf("couldn't retrieve posts: %w", err)
	}

	if len(posts) == 0 {
		fmt.Printf("No posts found. Try following some feeds first!\n")
		return nil
	}

	fmt.Printf("Latest posts (showing %d):\n\n", len(posts))
	for i, post := range posts {
		fmt.Printf("%d. %s\n", i+1, post.Title)
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

	return nil
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
