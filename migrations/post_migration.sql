-- Post-migration script with Go templating (PostgreSQL compatible)
-- This script executes after all migrations are completed successfully

{{- if .CreatedObjects}}
-- Log all created objects
DO $$
BEGIN
    RAISE NOTICE 'Migration completed successfully!';
    RAISE NOTICE 'Created {{len .CreatedObjects}} database objects:';
    {{- range .CreatedObjects}}
    RAISE NOTICE '  - {{.Type}}: {{.Name}}';
    {{- end}}
END $$;

{{- if .DeletedObjects}}
-- Log all deleted objects
DO $$
BEGIN
    RAISE NOTICE 'Deleted {{len .DeletedObjects}} database objects:';
    {{- range .DeletedObjects}}
    RAISE NOTICE '  - {{.Type}}: {{.Name}}';
    {{- end}}
END $$;
{{- end}}

-- Create a summary table with migration information
CREATE TABLE IF NOT EXISTS migration_summary (
    id SERIAL PRIMARY KEY,
    migration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    database_type VARCHAR(50) NOT NULL,
    total_objects INTEGER NOT NULL,
    notes TEXT
);

-- Insert summary record
INSERT INTO migration_summary (database_type, total_objects, notes)
VALUES (
    '{{.DatabaseType}}',
    {{len .CreatedObjects}},
    'Migration completed. Created objects: {{range .CreatedObjects}}{{.Type}}:{{.Name}} {{end}}{{if .DeletedObjects}}Deleted objects: {{range .DeletedObjects}}{{.Type}}:{{.Name}} {{end}}{{end}}'
);

-- Create a table to track created objects
CREATE TABLE IF NOT EXISTS created_objects_log (
    id SERIAL PRIMARY KEY,
    migration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    object_type VARCHAR(50) NOT NULL,
    object_name VARCHAR(255) NOT NULL
);

-- Log each created object
{{- range .CreatedObjects}}
INSERT INTO created_objects_log (object_type, object_name)
VALUES ('{{.Type}}', '{{.Name}}');
{{- end}}

{{- else if .DeletedObjects}}
-- Only objects were deleted during migration
DO $$
BEGIN
    RAISE NOTICE 'Migration completed but objects were deleted';
    RAISE NOTICE 'Deleted {{len .DeletedObjects}} database objects:';
    {{- range .DeletedObjects}}
    RAISE NOTICE '  - {{.Type}}: {{.Name}}';
    {{- end}}
END $$;

INSERT INTO migration_summary (database_type, total_objects, notes)
VALUES (
    '{{.DatabaseType}}',
    0,
    'Migration completed. Deleted objects: {{range .DeletedObjects}}{{.Type}}:{{.Name}} {{end}}'
);

{{- else}}
-- No objects were created or deleted during migration
DO $$
BEGIN
    RAISE NOTICE 'Migration completed but no database objects were changed';
END $$;

INSERT INTO migration_summary (database_type, total_objects, notes)
VALUES (
    '{{.DatabaseType}}',
    0,
    'Migration completed but no database objects were changed'
);
{{- end}}

-- Example: Create documentation for created tables
{{- range .CreatedObjects}}
    {{- if eq .Type "table"}}
-- Documentation for table: {{.Name}}
COMMENT ON TABLE {{.Name}} IS 'Created by BloomDB migration process';
    {{- end}}
{{- end}}