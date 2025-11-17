#!/bin/bash
# generate-docs.sh - Generate HTML documentation from AsciiDoc files

set -e

# Check if asciidoctor is available
if ! command -v asciidoctor &> /dev/null; then
    echo "Error: asciidoctor is required to generate HTML documentation"
    echo "Install with: gem install asciidoctor"
    exit 1
fi

# Create output directory
mkdir -p docs/html

# Generate HTML for each documentation file
echo "Generating HTML documentation..."

# Generate index.html
asciidoctor -D docs/html docs/index.adoc

# Generate individual pages
asciidoctor -D docs/html docs/getting-started.adoc
asciidoctor -D docs/html docs/migration-files.adoc
asciidoctor -D docs/html docs/commands.adoc
asciidoctor -D docs/html docs/configuration.adoc
asciidoctor -D docs/html docs/advanced-usage.adoc
asciidoctor -D docs/html docs/troubleshooting.adoc

echo "HTML documentation generated in docs/html/"
echo "Open docs/html/index.html to view documentation"

# Create a simple viewer script
cat > docs/view-docs.sh << 'EOF'
#!/bin/bash
# view-docs.sh - Open documentation in browser

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HTML_DIR="$SCRIPT_DIR/html"

if [ ! -f "$HTML_DIR/index.html" ]; then
    echo "HTML documentation not found. Run ./generate-docs.sh first."
    exit 1
fi

# Try to open in default browser
if command -v xdg-open &> /dev/null; then
    xdg-open "$HTML_DIR/index.html"
elif command -v open &> /dev/null; then
    open "$HTML_DIR/index.html"
elif command -v python3 &> /dev/null; then
    python3 -m webbrowser "$HTML_DIR/index.html"
else
    echo "Could not open browser automatically."
    echo "Open $HTML_DIR/index.html manually in your browser."
fi
EOF

chmod +x docs/view-docs.sh

echo "Created docs/view-docs.sh - run this script to view documentation in browser"