package cmd

type BaselineCommand struct{}

func (b *BaselineCommand) Run() {
	// Initialize printer first to ensure verbose output works
	InitPrinter()

	// Setup database connection first (needed for version resolution)
	setup := SetupDatabase()

	// Resolve baseline version with correct priority:
	// 1. Existing baseline in DB, 2. CLI flag, 3. Env var, 4. Default
	version := ResolveBaselineVersion(setup, baselineVersion)

	// Check if migration table exists
	tableExists, err := setup.Database.TableExists(setup.TableName)
	if err != nil {
		PrintError("Error checking table existence: " + err.Error())
		setup.Database.Close()
		return
	}

	if tableExists {
		// Table exists, check if baseline record already exists
		baselineExists, existingBaselineVersion, err := setup.CheckBaselineRecordExists()
		if err != nil {
			PrintError("Error checking baseline record: " + err.Error())
			setup.Database.Close()
			return
		}

		if baselineExists {
			// Baseline already exists - the resolved version IS the existing one
			// (ResolveBaselineVersion already returned the existing version)
			PrintSuccess("Baseline already exists with version " + existingBaselineVersion)
			return
		}

		PrintInfo("Migration table '" + setup.TableName + "' exists but no baseline record found")
	} else {
		// Table doesn't exist, create it
		PrintInfo("Migration table '" + setup.TableName + "' does not exist, creating it")
		err := setup.CreateMigrationTable()
		if err != nil {
			PrintError("Error creating migration table: " + err.Error())
			return
		}
	}

	// Insert baseline record
	err = setup.InsertBaselineRecord(version)
	if err != nil {
		PrintError("Error inserting baseline record: " + err.Error())
		return
	}

	PrintSuccess("Baseline completed successfully")
}
