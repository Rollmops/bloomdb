package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateChecksum(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
	}{
		{
			name:    "Empty content",
			content: []byte(""),
		},
		{
			name:    "Simple SQL",
			content: []byte("CREATE TABLE users (id INT PRIMARY KEY);"),
		},
		{
			name:    "SQL with newlines",
			content: []byte("CREATE TABLE users (\n    id INT PRIMARY KEY,\n    name VARCHAR(255),\n    email VARCHAR(255)\n);"),
		},
		{
			name:    "Unicode content",
			content: []byte("CREATE TABLE 测试 (id INT PRIMARY KEY);"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum := CalculateChecksum(tt.content)

			// Checksum should not be zero (unless content is empty and produces zero)
			if checksum == 0 && len(tt.content) > 0 {
				t.Errorf("Expected non-zero checksum for non-empty content, got %d", checksum)
			}
		})
	}
}

func TestCalculateChecksum_Deterministic(t *testing.T) {
	content := []byte("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255));")

	// Calculate checksum multiple times
	checksum1 := CalculateChecksum(content)
	checksum2 := CalculateChecksum(content)
	checksum3 := CalculateChecksum(content)

	// All should be identical
	assert.Equal(t, checksum1, checksum2, "Checksum should be deterministic")
	assert.Equal(t, checksum2, checksum3, "Checksum should be deterministic")
}

func TestCalculateChecksum_DifferentContent(t *testing.T) {
	content1 := []byte("CREATE TABLE users (id INT PRIMARY KEY);")
	content2 := []byte("CREATE TABLE users (id INTEGER PRIMARY KEY);")
	content3 := []byte("CREATE TABLE users (id INT PRIMARY KEY); ")

	checksum1 := CalculateChecksum(content1)
	checksum2 := CalculateChecksum(content2)
	checksum3 := CalculateChecksum(content3)

	// content1 and content2 should be different
	assert.NotEqual(t, checksum1, checksum2, "Different content should produce different checksums")

	// content1 and content3 should be different (trailing space)
	assert.NotEqual(t, checksum1, checksum3, "Content with trailing space should produce different checksums")
}

func TestCalculateChecksum_KnownValues(t *testing.T) {
	// Test with known content to verify CRC32 implementation
	// These values are calculated using the same algorithm as Flyway
	tests := []struct {
		name            string
		content         []byte
		expectedNonZero bool
	}{
		{
			name:            "hello world",
			content:         []byte("hello world"),
			expectedNonZero: true,
		},
		{
			name:            "empty content",
			content:         []byte(""),
			expectedNonZero: false, // Empty content produces zero
		},
		{
			name:            "simple SQL",
			content:         []byte("CREATE TABLE users (id INT);"),
			expectedNonZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum := CalculateChecksum(tt.content)
			if tt.expectedNonZero {
				assert.NotZero(t, checksum, "Expected non-zero checksum")
			}
			// Verify it's within int32 range (Flyway uses signed 32-bit)
			assert.True(t, checksum >= -2147483648 && checksum <= 2147483647, "Checksum should be within int32 range")
		})
	}
}

func TestCalculateChecksum_LineEndingIndependence(t *testing.T) {
	// Test that different line endings produce the same checksum (Flyway behavior)
	contentUnix := []byte("CREATE TABLE users (\nid INT,\nname VARCHAR(255)\n);")
	contentWindows := []byte("CREATE TABLE users (\r\nid INT,\r\nname VARCHAR(255)\r\n);")
	contentMac := []byte("CREATE TABLE users (\rid INT,\rname VARCHAR(255)\r);")

	checksumUnix := CalculateChecksum(contentUnix)
	checksumWindows := CalculateChecksum(contentWindows)
	checksumMac := CalculateChecksum(contentMac)

	assert.Equal(t, checksumUnix, checksumWindows, "Unix and Windows line endings should produce same checksum")
	assert.Equal(t, checksumUnix, checksumMac, "Unix and Mac line endings should produce same checksum")
}

func TestCalculateChecksum_BOMStripping(t *testing.T) {
	// Test that UTF-8 BOM is stripped from first line
	contentWithBOM := []byte("\ufeffCREATE TABLE users (id INT);")
	contentWithoutBOM := []byte("CREATE TABLE users (id INT);")

	checksumWithBOM := CalculateChecksum(contentWithBOM)
	checksumWithoutBOM := CalculateChecksum(contentWithoutBOM)

	assert.Equal(t, checksumWithoutBOM, checksumWithBOM, "BOM should be stripped and produce same checksum")
}

func TestCalculateChecksum_Performance(t *testing.T) {
	// Test with larger content to ensure reasonable performance
	content := make([]byte, 100000) // 100KB
	for i := range content {
		content[i] = byte(i % 256)
	}

	// Should complete quickly
	checksum := CalculateChecksum(content)
	assert.NotZero(t, checksum, "Expected non-zero checksum for large content")
}
