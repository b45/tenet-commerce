package idempotency_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b45/tenet-commerce/backend/pkg/idempotency"
)

func TestComputeFingerprint_CanonicalJSONKeyOrder(t *testing.T) {
	// Two payloads with identical keys and values in different order and whitespace
	json1 := []byte(`{ "b": 2, "a": 1, "nested": { "z": 26, "y": 25 } }`)
	json2 := []byte(`{"nested":{"y":25,"z":26},"a":1,"b":2}`)

	hash1, err := idempotency.ComputeFingerprint("POST", "/api/v1/pos/checkout", json1)
	require.NoError(t, err)

	hash2, err := idempotency.ComputeFingerprint("POST", "/api/v1/pos/checkout", json2)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2, "identical content in different order must generate identical hash")
}

func TestComputeFingerprint_DifferentPayloadsDiffer(t *testing.T) {
	json1 := []byte(`{"amount": 10000}`)
	json2 := []byte(`{"amount": 20000}`)

	hash1, err := idempotency.ComputeFingerprint("POST", "/api/v1/pos/checkout", json1)
	require.NoError(t, err)

	hash2, err := idempotency.ComputeFingerprint("POST", "/api/v1/pos/checkout", json2)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "different amounts must produce different hashes")
}

func TestComputeFingerprint_DifferentRoutesDiffer(t *testing.T) {
	body := []byte(`{"amount": 10000}`)

	hash1, err := idempotency.ComputeFingerprint("POST", "/api/v1/pos/checkout", body)
	require.NoError(t, err)

	hash2, err := idempotency.ComputeFingerprint("POST", "/api/v1/pos/void", body)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "different routes with same payload must produce different hashes")
}

func TestComputeFingerprint_EmptyBody(t *testing.T) {
	hash1, err := idempotency.ComputeFingerprint("POST", "/api/v1/test", nil)
	require.NoError(t, err)

	hash2, err := idempotency.ComputeFingerprint("POST", "/api/v1/test", []byte("   "))
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2, "nil and whitespace body must produce identical hash")
}
