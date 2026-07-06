package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func UserIDFromFingerprint(fingerprint string) string {
	normalized := strings.TrimSpace(fingerprint)
	if normalized == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
