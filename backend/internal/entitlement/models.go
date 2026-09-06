package entitlement

import (
	"time"
)

// GrantType represents whether a feature is a boolean capability or quota-limited.
type GrantType string

const (
	GrantTypeBoolean GrantType = "BOOLEAN"
	GrantTypeQuota   GrantType = "QUOTA"
)

// SubscriptionStatus represents the lifecycle state of a tenant subscription.
type SubscriptionStatus string

const (
	StatusActive   SubscriptionStatus = "ACTIVE"
	StatusTrialing SubscriptionStatus = "TRIALING"
	StatusPastDue  SubscriptionStatus = "PAST_DUE"
	StatusCanceled SubscriptionStatus = "CANCELED"
)

// DecisionReason describes the rationale behind an access evaluation decision.
type DecisionReason string

const (
	ReasonEntitled             DecisionReason = "ENTITLED"
	ReasonFeatureNotEntitled   DecisionReason = "FEATURE_NOT_ENTITLED"
	ReasonNoSubscription       DecisionReason = "NO_ACTIVE_SUBSCRIPTION"
	ReasonSubscriptionInactive DecisionReason = "SUBSCRIPTION_INACTIVE"
	ReasonFeatureDisabled      DecisionReason = "FEATURE_DISABLED"
	ReasonQuotaExceeded        DecisionReason = "QUOTA_EXCEEDED"
)

// PlanFeature defines a specific capability grant within a plan.
type PlanFeature struct {
	ID         string     `json:"id"`
	PlanID     string     `json:"plan_id"`
	FeatureKey string     `json:"feature_key"`
	GrantType  GrantType  `json:"grant_type"`
	QuotaLimit *int64     `json:"quota_limit,omitempty"`
	IsEnabled  bool       `json:"is_enabled"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Plan represents a commercial or operational packaging tier.
type Plan struct {
	ID        string                 `json:"id"`
	Code      string                 `json:"code"`
	Name      string                 `json:"name"`
	Version   int                    `json:"version"`
	IsActive  bool                   `json:"is_active"`
	Features  map[string]PlanFeature `json:"features,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// TenantSubscription maps a tenant to an active plan with lifecycle status.
type TenantSubscription struct {
	ID                 string             `json:"id"`
	TenantID           string             `json:"tenant_id"`
	PlanID             string             `json:"plan_id"`
	Status             SubscriptionStatus `json:"status"`
	CurrentPeriodStart time.Time          `json:"current_period_start"`
	CurrentPeriodEnd   time.Time          `json:"current_period_end"`
	CanceledAt         *time.Time         `json:"canceled_at,omitempty"`
	Plan               *Plan              `json:"plan,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// Decision represents the result of a feature capability evaluation.
type Decision struct {
	Allowed    bool           `json:"allowed"`
	Reason     DecisionReason `json:"reason"`
	GrantType  GrantType      `json:"grant_type,omitempty"`
	QuotaLimit *int64         `json:"quota_limit,omitempty"`
}

// PlanSummary provides a concise overview of the active plan for client introspection.
type PlanSummary struct {
	Code   string             `json:"code"`
	Name   string             `json:"name"`
	Status SubscriptionStatus `json:"status"`
}

// CapabilitiesResponse represents the payload returned by GET /api/v1/me/capabilities.
type CapabilitiesResponse struct {
	TenantID      string              `json:"tenant_id"`
	Plan          PlanSummary         `json:"plan"`
	Capabilities  map[string]Decision `json:"capabilities"`
	PolicyVersion int                 `json:"policy_version"`
}
