package idempotency

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusProcessing = "PROCESSING"
	StatusCompleted  = "COMPLETED"
	StatusFailed     = "FAILED"

	DefaultTTL        = 24 * time.Hour
	InProgressLockTTL = 30 * time.Second
)

// Record represents a persisted idempotency request record in the tenant schema.
type Record struct {
	ID                 uuid.UUID `json:"id"`
	IdempotencyKey     string    `json:"idempotency_key"`
	TargetRoute        string    `json:"target_route"`
	RequestHash        string    `json:"request_hash"`
	Status             string    `json:"status"`
	ResponseStatusCode *int      `json:"response_status_code,omitempty"`
	ResponseHeaders    []byte    `json:"response_headers,omitempty"`
	ResponseBody       []byte    `json:"response_body,omitempty"`
	LockedAt           time.Time `json:"locked_at"`
	ExpiresAt          time.Time `json:"expires_at"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
