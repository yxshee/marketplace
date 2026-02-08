package identifier

import (
	"crypto/rand"
	"encoding/hex"
)

// New creates a short entropy-backed identifier with a stable prefix.
func New(prefix string) string {
	buf := make([]byte, 12)
	_, _ = rand.Read(buf)
	return prefix + "_" + hex.EncodeToString(buf)
}
