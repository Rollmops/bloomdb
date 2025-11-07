package loader

import (
	"crypto/sha256"
)

// CalculateChecksum creates a numeric checksum from the SHA256 hash of the given content
func CalculateChecksum(content []byte) int64 {
	hash := sha256.Sum256(content)

	// Convert first 8 bytes of hash to int64
	var result int64
	for i := 0; i < 8; i++ {
		result = (result << 8) | int64(hash[i])
	}

	return result
}
