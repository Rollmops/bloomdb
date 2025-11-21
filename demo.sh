#!/bin/bash

# Check if we should run the demo or record it
if [ "$1" != "--run-demo" ]; then
    echo "Recording demo..."
    asciinema rec --cols 120 --rows 30 -c "$0 --run-demo" demo.cast --overwrite
    
    echo "Generating GIF..."
    if command -v agg >/dev/null 2>&1; then
        agg demo.cast demo.gif
        echo "Done: demo.gif generated"
    else
        echo "Error: 'agg' not found. Please install it to generate the GIF."
        echo "You can install it from: https://github.com/asciinema/agg"
    fi
    exit 0
fi

# Setup
set -e
DEMO_DIR="/tmp/bloomdb-demo"
rm -rf "$DEMO_DIR"
mkdir -p "$DEMO_DIR"
DB_PATH="$DEMO_DIR/demo.db"
MIGRATIONS_DIR="$DEMO_DIR/migrations"
BLOOMDB_BIN="./bloomdb"

# Ensure bloomdb is built
if [ ! -f "$BLOOMDB_BIN" ]; then
    echo "Building bloomdb..."
    go build -o bloomdb
fi

# Helper to simulate typing
type_cmd() {
    cmd="$1"
    printf "$ "
    for (( i=0; i<${#cmd}; i++ )); do
        char="${cmd:$i:1}"
        printf "$char"
        sleep 0.05
    done
    echo ""
    sleep 0.5
    eval "$cmd"
    sleep 1
    echo ""
}

# Helper to show file content
show_file() {
    file="$1"
    echo "$ cat $file"
    cat "$file"
    echo ""
    sleep 1
}

# Start Demo
clear
echo "# Welcome to BloomDB Demo"
sleep 1
echo "# Let's start by initializing a new SQLite database"
sleep 1

echo "# We are setting the baseline version to 0 to simulate a new database"
sleep 1
type_cmd "export BLOOMDB_BASELINE_VERSION=0"
echo ""

echo "# We are setting the path to our migration files"
sleep 1
type_cmd "export BLOOMDB_PATH=$MIGRATIONS_DIR"
echo ""


export BLOOMDB_CONNECT_STRING="sqlite:$DB_PATH"
type_cmd "./bloomdb baseline"

echo "# Now let's create some migrations"
mkdir -p "$MIGRATIONS_DIR"

# Create V1
cat > "$MIGRATIONS_DIR/V1__Create_users.sql" <<EOF
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    username TEXT NOT NULL,
    email TEXT NOT NULL
);
EOF

# Create V2
cat > "$MIGRATIONS_DIR/V2__Add_posts.sql" <<EOF
CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT,
    FOREIGN KEY(user_id) REFERENCES users(id)
);
EOF

# Create a repeatable migration
cat > "$MIGRATIONS_DIR/R__create_view.sql" <<EOF
CREATE VIEW user_posts AS
SELECT * FROM posts WHERE user_id IN (SELECT id FROM users);
EOF

echo "# We have created three migration files:"
type_cmd "ls -1 $MIGRATIONS_DIR"

echo "# Let's apply them"
type_cmd "./bloomdb migrate"

echo "# We can check the status of our migrations"
type_cmd "./bloomdb info"

echo "# BloomDB tracks everything for you!"
sleep 1

# Cleanup
rm -rf "$DEMO_DIR"
