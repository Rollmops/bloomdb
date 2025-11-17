# BloomDB Documentation

This directory contains comprehensive documentation for BloomDB, organized into multiple focused pages.

## Documentation Structure

- **index.adoc** - Main documentation index and overview
- **getting-started.adoc** - Installation and basic usage guide
- **migration-files.adoc** - Migration file structure and naming conventions
- **commands.adoc** - Complete command reference
- **configuration.adoc** - Environment variables and configuration options
- **advanced-usage.adoc** - Advanced features like post-migration scripts
- **troubleshooting.adoc** - Common issues and solutions

## Viewing Documentation

### Online

The documentation is automatically published to GitHub Pages at:
https://rollmops.github.io/bloomdb/

### Local

To generate and view documentation locally:

```bash
# Generate HTML documentation
cd docs
./generate-docs.sh

# View in browser
./view-docs.sh
```

### Prerequisites for Local Viewing

- Ruby 3.0+
- Asciidoctor gem

Install with:
```bash
gem install asciidoctor
gem install asciidoctor-html5s
```

## Contributing to Documentation

When contributing to documentation:

1. Edit the appropriate `.adoc` file in this directory
2. Test your changes locally with `./generate-docs.sh`
3. Submit a pull request

### AsciiDoc Guidelines

- Use standard AsciiDoc syntax
- Keep lines under 100 characters when possible
- Use semantic section titles (==, ===, ====)
- Include code examples with [source,bash] or [source,sql]
- Use tables for structured information
- Cross-reference other documents with link:page.adoc[text]

## Documentation Features

The documentation includes:

- **Comprehensive coverage** - All BloomDB features documented
- **Practical examples** - Real-world usage scenarios
- **Cross-references** - Easy navigation between topics
- **Multiple formats** - AsciiDoc source and HTML output
- **Searchable** - Well-structured for easy searching
- **Printable** - Clean formatting for printing

## Support

If you find issues with the documentation:

1. Check the [troubleshooting guide](troubleshooting.adoc)
2. [Open an issue](https://github.com/rollmops/bloomdb/issues)
3. [Start a discussion](https://github.com/rollmops/bloomdb/discussions)

## License

Documentation follows the same license as the BloomDB project.