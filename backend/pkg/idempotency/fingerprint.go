package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// ComputeFingerprint generates a deterministic SHA-256 hex digest for a request.
// It canonicalizes JSON bodies by normalizing whitespace and sorting object keys.
func ComputeFingerprint(method, path string, rawBody []byte) (string, error) {
	canonicalBody := ""
	trimmed := bytes.TrimSpace(rawBody)

	if len(trimmed) > 0 {
		var parsed any
		if err := json.Unmarshal(trimmed, &parsed); err == nil {
			// json.Marshal recursively sorts map keys deterministically
			canonicalJSON, err := json.Marshal(parsed)
			if err != nil {
				return "", fmt.Errorf("canonicalize json: %w", err)
			}
			canonicalBody = string(canonicalJSON)
		} else {
			// Non-JSON payload: use trimmed raw bytes
			canonicalBody = string(trimmed)
		}
	}

	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s:%s:%s", strings.ToUpper(strings.TrimSpace(method)), strings.TrimSpace(path), canonicalBody)
	return hex.EncodeToString(h.Sum(nil)), nil
}
