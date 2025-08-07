package main

import (
	"fmt"
	"gator/internal/config"
	"log"
)

func main() {
	// Read the config file
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
	}

	fmt.Println("Initial config:")
	fmt.Printf("DB URL: %s\n", cfg.DbURL)
	fmt.Printf("Current User: %s\n", cfg.CurrentUserName)
	fmt.Println()

	// Set the current user to "andi" and update the config file on disk
	err = cfg.SetUser("andi")
	if err != nil {
		log.Fatalf("Error setting user: %v", err)
	}

	fmt.Println("Updated user to 'andi'")
	fmt.Println()

	// Read the config file again and print the contents
	cfg, err = config.Read()
	if err != nil {
		log.Fatalf("Error reading config after update: %v", err)
	}

	fmt.Println("Final config:")
	fmt.Printf("DB URL: %s\n", cfg.DbURL)
	fmt.Printf("Current User: %s\n", cfg.CurrentUserName)
}
