package main

import (
	"context"
	"database/sql"
	"fmt"
	"gator/internal/config"
	"gator/internal/database"
	"gator/internal/rss"
	"os"
	"time"

	"github.com/google/uuid"
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
		return fmt.Errorf("user '%s' not found", username)
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
	feedURL := "https://www.wagslane.dev/index.xml"

	feed, err := rss.FetchFeed(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("couldn't fetch feed: %w", err)
	}

	// Print the entire struct to the console
	fmt.Printf("%+v\n", feed)
	return nil
}

func main() {
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
