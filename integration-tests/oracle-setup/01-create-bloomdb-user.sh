#!/bin/bash
# Oracle setup script for BloomDB tests
# This script creates the bloomdb user and grants necessary permissions

# Wait for Oracle to be fully ready
sleep 60

# Create bloomdb user
sqlplus -s sys/Oracle123456@XEPDB1 as sysdba << 'EOF'
-- Drop user if exists (ignore errors)
BEGIN
   EXECUTE IMMEDIATE 'DROP USER bloomdb CASCADE';
EXCEPTION
   WHEN OTHERS THEN
      IF SQLCODE != -1918 THEN
         RAISE;
      END IF;
END;
/

-- Create bloomdb user
CREATE USER bloomdb IDENTIFIED BY bloomdb
DEFAULT TABLESPACE users
TEMPORARY TABLESPACE temp;

-- Grant necessary permissions
GRANT CONNECT, RESOURCE, CREATE VIEW, CREATE SEQUENCE TO bloomdb;
GRANT UNLIMITED TABLESPACE TO bloomdb;

-- Exit
EXIT;
EOF

echo "Oracle setup completed - bloomdb user created"