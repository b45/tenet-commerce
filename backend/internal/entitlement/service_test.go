package entitlement

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockRepo implements SubscriptionRepository for unit testing without DB dependency
type mockRepo struct {
	sub *TenantSubscription
	err error
}

func (m *mockRepo) GetSubscriptionByTenantID(ctx context.Context, tenantID string) (*TenantSubscription, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.sub, nil
}

func TestEntitlementService_Evaluate(t *testing.T) {
	quotaLimit := int64(100)
	starterPlan := &Plan{
		ID:       "plan-1",
		Code:     "starter",
		Name:     "Starter Tier",
		Version:  1,
		IsActive: true,
		Features: map[string]PlanFeature{
			"pos.checkout": {
				FeatureKey: "pos.checkout",
				GrantType:  GrantTypeBoolean,
				IsEnabled:  true,
			},
			"pos.daily_summary": {
				FeatureKey: "pos.daily_summary",
				GrantType:  GrantTypeBoolean,
				IsEnabled:  false, // explicitly disabled in starter
			},
			"catalog.max_products": {
				FeatureKey: "catalog.max_products",
				GrantType:  GrantTypeQuota,
				QuotaLimit: &quotaLimit,
				IsEnabled:  true,
			},
		},
	}

	activeSub := &TenantSubscription{
		ID:                 "sub-1",
		TenantID:           "tenant-alpha",
		PlanID:             "plan-1",
		Status:             StatusActive,
		CurrentPeriodStart: time.Now().Add(-24 * time.Hour),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		Plan:               starterPlan,
	}

	tests := []struct {
		name          string
		tenantID      string
		featureKey    string
		sub           *TenantSubscription
		subErr        error
		wantAllowed   bool
		wantReason    DecisionReason
		wantGrantType GrantType
		wantErr       bool
	}{
		{
			name:          "Active plan with enabled boolean feature is allowed",
			tenantID:      "tenant-alpha",
			featureKey:    "pos.checkout",
			sub:           activeSub,
			wantAllowed:   true,
			wantReason:    ReasonEntitled,
			wantGrantType: GrantTypeBoolean,
		},
		{
			name:        "Active plan with disabled feature is denied",
			tenantID:    "tenant-alpha",
			featureKey:  "pos.daily_summary",
			sub:         activeSub,
			wantAllowed: false,
			wantReason:  ReasonFeatureNotEntitled,
		},
		{
			name:        "Unknown feature key denies by default",
			tenantID:    "tenant-alpha",
			featureKey:  "unknown.feature.flag",
			sub:         activeSub,
			wantAllowed: false,
			wantReason:  ReasonFeatureNotEntitled,
		},
		{
			name:          "Quota-based feature returns grant type and quota limit",
			tenantID:      "tenant-alpha",
			featureKey:    "catalog.max_products",
			sub:           activeSub,
			wantAllowed:   true,
			wantReason:    ReasonEntitled,
			wantGrantType: GrantTypeQuota,
		},
		{
			name:        "Missing subscription denies access",
			tenantID:    "tenant-unknown",
			featureKey:  "pos.checkout",
			subErr:      ErrNoSubscription,
			wantAllowed: false,
			wantReason:  ReasonNoSubscription,
		},
		{
			name:       "DB query failure fails closed and returns error",
			tenantID:   "tenant-err",
			featureKey: "pos.checkout",
			subErr:     errors.New("db connection timeout"),
			wantErr:    true,
		},
		{
			name:       "Canceled subscription denies access even for basic features",
			tenantID:   "tenant-canceled",
			featureKey: "pos.checkout",
			sub: &TenantSubscription{
				ID:       "sub-canceled",
				TenantID: "tenant-canceled",
				Status:   StatusCanceled,
				Plan:     starterPlan,
			},
			wantAllowed: false,
			wantReason:  ReasonSubscriptionInactive,
		},
		{
			name:       "Past-due subscription denies access",
			tenantID:   "tenant-pastdue",
			featureKey: "pos.checkout",
			sub: &TenantSubscription{
				ID:       "sub-pastdue",
				TenantID: "tenant-pastdue",
				Status:   StatusPastDue,
				Plan:     starterPlan,
			},
			wantAllowed: false,
			wantReason:  ReasonSubscriptionInactive,
		},
		{
			name:        "Empty tenant ID denies by default",
			tenantID:    "",
			featureKey:  "pos.checkout",
			sub:         activeSub,
			wantAllowed: false,
			wantReason:  ReasonFeatureNotEntitled,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{sub: tc.sub, err: tc.subErr}
			svc := NewService(repo, nil)

			decision, err := svc.Evaluate(context.Background(), tc.tenantID, tc.featureKey)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if decision.Allowed {
					t.Errorf("expected Allowed=false on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if decision.Allowed != tc.wantAllowed {
				t.Errorf("expected Allowed=%v, got %v", tc.wantAllowed, decision.Allowed)
			}
			if decision.Reason != tc.wantReason {
				t.Errorf("expected Reason=%v, got %v", tc.wantReason, decision.Reason)
			}
			if tc.wantGrantType != "" && decision.GrantType != tc.wantGrantType {
				t.Errorf("expected GrantType=%v, got %v", tc.wantGrantType, decision.GrantType)
			}
			if tc.wantGrantType == GrantTypeQuota && (decision.QuotaLimit == nil || *decision.QuotaLimit != quotaLimit) {
				t.Errorf("expected QuotaLimit=%d, got %v", quotaLimit, decision.QuotaLimit)
			}
		})
	}
}

func TestEntitlementService_GetEffectiveCapabilities(t *testing.T) {
	plan := &Plan{
		ID:       "plan-1",
		Code:     "growth",
		Name:     "Growth Tier",
		Version:  2,
		IsActive: true,
		Features: map[string]PlanFeature{
			"pos.checkout": {
				FeatureKey: "pos.checkout",
				GrantType:  GrantTypeBoolean,
				IsEnabled:  true,
			},
			"pos.daily_summary": {
				FeatureKey: "pos.daily_summary",
				GrantType:  GrantTypeBoolean,
				IsEnabled:  true,
			},
		},
	}

	sub := &TenantSubscription{
		ID:       "sub-1",
		TenantID: "tenant-beta",
		Status:   StatusActive,
		Plan:     plan,
	}

	repo := &mockRepo{sub: sub}
	svc := NewService(repo, nil)

	resp, err := svc.GetEffectiveCapabilities(context.Background(), "tenant-beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TenantID != "tenant-beta" {
		t.Errorf("expected TenantID=tenant-beta, got %s", resp.TenantID)
	}
	if resp.Plan.Code != "growth" {
		t.Errorf("expected Plan.Code=growth, got %s", resp.Plan.Code)
	}
	if resp.PolicyVersion != 2 {
		t.Errorf("expected PolicyVersion=2, got %d", resp.PolicyVersion)
	}
	if !resp.Capabilities["pos.daily_summary"].Allowed {
		t.Errorf("expected pos.daily_summary to be allowed")
	}
	if resp.Capabilities["pos.daily_summary"].Reason != ReasonEntitled {
		t.Errorf("expected Reason=ENTITLED, got %s", resp.Capabilities["pos.daily_summary"].Reason)
	}
}
