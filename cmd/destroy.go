package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type DestroyCommand struct{}

func (d *DestroyCommand) Run() {
	PrintWarning("Starting destroy command - this is a destructive operation")

	// Setup database connection
	setup := SetupDatabase()

	PrintWarning("This will destroy ALL database objects in " + string(setup.DBType) + " database!")
	PrintWarning("This includes tables, views, indexes, triggers, and all data.")
	PrintWarning("This operation cannot be undone.")
	PrintInfo("")

	// Get confirmation from user
	if !getConfirmation() {
		PrintInfo("Destroy operation cancelled.")
		return
	}

	PrintInfo("User confirmed destroy operation - proceeding with destruction")
	PrintInfo("Destroying all database objects...")

	// Drop all objects based on database type
	err := setup.Database.DestroyAllObjects()
	if err != nil {
		PrintError("Error destroying database objects: " + err.Error())
		return
	}

	PrintSuccess("Successfully destroyed all database objects.")
}

// getConfirmation gets user confirmation before proceeding with destroy
func getConfirmation() bool {
	fmt.Print("Type 'DESTROY' to confirm: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		PrintError("Error reading input: " + err.Error())
		return false
	}

	// Trim whitespace and convert to uppercase
	input = strings.TrimSpace(strings.ToUpper(input))

	return input == "DESTROY"
}
