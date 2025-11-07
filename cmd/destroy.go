package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"bloomdb/logger"
)

type DestroyCommand struct{}

func (d *DestroyCommand) Run() {
	logger.Warn("Starting destroy command - this is a destructive operation")

	// Setup database connection
	setup := SetupDatabase()
	logger.Infof("Connected to %s database", setup.DBType)

	logger.Warnf("This will destroy ALL database objects in %s database!", setup.DBType)
	logger.Warn("This includes tables, views, indexes, triggers, and all data.")
	logger.Warn("This operation cannot be undone.")

	fmt.Printf("WARNING: This will destroy ALL database objects in %s database!\n", setup.DBType)
	fmt.Printf("This includes tables, views, indexes, triggers, and all data.\n")
	fmt.Printf("This operation cannot be undone.\n")
	fmt.Println()

	// Get confirmation from user
	if !getConfirmation() {
		logger.Info("Destroy operation cancelled by user")
		fmt.Println("Destroy operation cancelled.")
		return
	}

	logger.Info("User confirmed destroy operation - proceeding with destruction")
	fmt.Println("Destroying all database objects...")

	// Drop all objects based on database type
	err := setup.Database.DestroyAllObjects()
	if err != nil {
		logger.Errorf("Error destroying database objects: %v", err)
		fmt.Printf("Error destroying database objects: %v\n", err)
		return
	}

	logger.Warn("Successfully destroyed all database objects")
	fmt.Println("Successfully destroyed all database objects.")
}

// getConfirmation gets user confirmation before proceeding with destroy
func getConfirmation() bool {
	fmt.Printf("Type 'DESTROY' to confirm: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		logger.Errorf("Error reading user input: %v", err)
		fmt.Printf("Error reading input: %v\n", err)
		return false
	}

	// Trim whitespace and convert to uppercase
	input = strings.TrimSpace(strings.ToUpper(input))
	logger.Debugf("User confirmation input: %s", input)

	return input == "DESTROY"
}
