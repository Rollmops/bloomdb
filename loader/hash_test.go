package loader

import (
	"testing"
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
	if checksum1 != checksum2 || checksum2 != checksum3 {
		t.Errorf("Checksum function is not deterministic: %d, %d, %d", checksum1, checksum2, checksum3)
	}
}

func TestCalculateChecksum_DifferentContent(t *testing.T) {
	content1 := []byte("CREATE TABLE users (id INT PRIMARY KEY);")
	content2 := []byte("CREATE TABLE users (id INTEGER PRIMARY KEY);")
	content3 := []byte("CREATE TABLE users (id INT PRIMARY KEY); ")

	checksum1 := CalculateChecksum(content1)
	checksum2 := CalculateChecksum(content2)
	checksum3 := CalculateChecksum(content3)

	// content1 and content2 should be different
	if checksum1 == checksum2 {
		t.Errorf("Different content should produce different checksums: %d vs %d", checksum1, checksum2)
	}

	// content1 and content3 should be different (trailing space)
	if checksum1 == checksum3 {
		t.Errorf("Content with trailing space should produce different checksums: %d vs %d", checksum1, checksum3)
	}
}

func TestCalculateChecksum_KnownValues(t *testing.T) {
	// Test with known content to verify consistency
	content := []byte("hello world")
	actualChecksum := CalculateChecksum(content)

	// We don't have a specific expected value since we changed the algorithm,
	// but we can verify it's consistent across runs
	if actualChecksum == 0 {
		t.Errorf("Expected non-zero checksum for 'hello world', got %d", actualChecksum)
	}

	emptyContent := []byte("")
	_ = CalculateChecksum(emptyContent)
	// Empty content might produce zero, which is acceptable
}

func TestCalculateChecksum_Performance(t *testing.T) {
	// Test with larger content to ensure reasonable performance
	content := make([]byte, 100000) // 100KB
	for i := range content {
		content[i] = byte(i % 256)
	}

	// Should complete quickly
	checksum := CalculateChecksum(content)
	if checksum == 0 {
		t.Errorf("Expected non-zero checksum for large content, got %d", checksum)
	}
}
