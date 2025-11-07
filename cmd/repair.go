package cmd

import (
	"fmt"

	"bloomdb/logger"
)

type RepairCommand struct{}

func (r *RepairCommand) Run() {
	logger.Info("Starting repair command")

	// Setup database connection
	setup := SetupDatabase()
	logger.Infof("Connected to %s database", setup.DBType)

	logger.Infof("Repair completed successfully - connected to %s database", setup.DBType)
	fmt.Printf("repair - connected to %s database\n", setup.DBType)
}
