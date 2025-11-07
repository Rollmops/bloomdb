package loader

import (
	"hash/crc32"
	"strings"
)

// CalculateChecksum creates a CRC32 checksum from the given content,
// compatible with Flyway's checksum calculation algorithm.
// This implementation:
// - Uses CRC32 algorithm (same as Flyway)
// - Normalizes line endings (makes it cross-platform compatible)
// - Strips UTF-8 BOM from the first line if present
// - Returns a signed 32-bit integer
func CalculateChecksum(content []byte) int64 {
	crc := crc32.NewIEEE()

	// Split content into lines, handling all line ending types
	lines := splitLines(content)

	for i, line := range lines {
		// Strip BOM from first line if present
		if i == 0 {
			line = stripBOM(line)
		}

		// Update CRC32 with the line content (UTF-8 bytes)
		crc.Write([]byte(line))
	}

	// Cast to int32 to match Flyway's behavior (signed 32-bit integer)
	// then convert to int64 for storage
	return int64(int32(crc.Sum32()))
}

// splitLines splits content into lines, handling \n, \r\n, and \r line endings
// This ensures cross-platform compatibility with Flyway's behavior
func splitLines(content []byte) []string {
	if len(content) == 0 {
		return []string{}
	}

	var lines []string
	var currentLine strings.Builder

	for i := 0; i < len(content); i++ {
		b := content[i]

		if b == '\r' {
			// Check if next byte is \n (Windows line ending)
			if i+1 < len(content) && content[i+1] == '\n' {
				// Windows line ending \r\n
				lines = append(lines, currentLine.String())
				currentLine.Reset()
				i++ // Skip the \n
			} else {
				// Mac classic line ending \r
				lines = append(lines, currentLine.String())
				currentLine.Reset()
			}
		} else if b == '\n' {
			// Unix line ending \n
			lines = append(lines, currentLine.String())
			currentLine.Reset()
		} else {
			// Regular character
			currentLine.WriteByte(b)
		}
	}

	// Add the last line if there's any remaining content
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// stripBOM removes the UTF-8 Byte Order Mark (BOM) character from the beginning of a string
func stripBOM(s string) string {
	if strings.HasPrefix(s, "\ufeff") {
		return s[3:] // UTF-8 BOM is 3 bytes: 0xEF, 0xBB, 0xBF
	}
	return s
}
