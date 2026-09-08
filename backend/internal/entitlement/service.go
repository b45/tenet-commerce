package entitlement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/b45/tenet-commerce/backend/pkg/logger"
	pkgRedis "github.com/b45/tenet-commerce/backend/pkg/redis"
)

const (
	subscriptionCacheTTL = 5 * time.Minute
	cacheKeyPrefix       = "entitlement:sub:"
)

// SubscriptionRepository defines data access methods for tenant subscriptions.
type SubscriptionRepository interface {
	GetSubscriptionByTenantID(ctx context.Context, tenantID string) (*TenantSubscription, error)
}

// Service provides authoritative feature entitlement evaluation.
type Service struct {
	repo  SubscriptionRepository
	redis *pkgRedis.Client
}

// NewService constructs a new entitlement Service.
func NewService(repo SubscriptionRepository, redis *pkgRedis.Client) *Service {
	return &Service{
		repo:  repo,
		redis: redis,
	}
}

// Evaluate determines whether a tenant is entitled to use a specific feature.
// It strictly follows the deny-by-default rule: any unknown key, missing grant,
// or inactive subscription results in Allowed: false.
func (s *Service) Evaluate(ctx context.Context, tenantID string, featureKey string) (Decision, error) {
	if tenantID == "" || featureKey == "" {
		return Decision{
			Allowed: false,
			Reason:  ReasonFeatureNotEntitled,
		}, nil
	}

	sub, err := s.getSubscriptionCached(ctx, tenantID)
	if err != nil {
		if errors.Is(err, ErrNoSubscription) {
			return Decision{
				Allowed: false,
				Reason:  ReasonNoSubscription,
			}, nil
		}
		// Fail closed on infrastructure errors
		logger.Error("Entitlement evaluation query failed", "tenant_id", tenantID, "feature_key", featureKey, "error", err)
		return Decision{
			Allowed: false,
			Reason:  ReasonSubscriptionInactive,
		}, fmt.Errorf("entitlement check failed: %w", err)
	}

	// Verify subscription lifecycle status
	if sub.Status != StatusActive && sub.Status != StatusTrialing {
		return Decision{
			Allowed: false,
			Reason:  ReasonSubscriptionInactive,
		}, nil
	}

	if sub.Plan == nil || !sub.Plan.IsActive {
		return Decision{
			Allowed: false,
			Reason:  ReasonSubscriptionInactive,
		}, nil
	}

	// Check feature grant in plan
	feature, exists := sub.Plan.Features[featureKey]
	if !exists || !feature.IsEnabled {
		return Decision{
			Allowed: false,
			Reason:  ReasonFeatureNotEntitled,
		}, nil
	}

	return Decision{
		Allowed:    true,
		Reason:     ReasonEntitled,
		GrantType:  feature.GrantType,
		QuotaLimit: feature.QuotaLimit,
	}, nil
}

// GetEffectiveCapabilities returns all capabilities and relevant limits for the tenant.
func (s *Service) GetEffectiveCapabilities(ctx context.Context, tenantID string) (*CapabilitiesResponse, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id is required")
	}

	sub, err := s.getSubscriptionCached(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	resp := &CapabilitiesResponse{
		TenantID: tenantID,
		Plan: PlanSummary{
			Code:   sub.Plan.Code,
			Name:   sub.Plan.Name,
			Status: sub.Status,
		},
		Capabilities:  make(map[string]Decision),
		PolicyVersion: sub.Plan.Version,
	}

	// Map all plan features
	for key, feat := range sub.Plan.Features {
		decision := Decision{
			Allowed:    feat.IsEnabled && (sub.Status == StatusActive || sub.Status == StatusTrialing),
			GrantType:  feat.GrantType,
			QuotaLimit: feat.QuotaLimit,
		}
		if decision.Allowed {
			decision.Reason = ReasonEntitled
		} else if sub.Status != StatusActive && sub.Status != StatusTrialing {
			decision.Reason = ReasonSubscriptionInactive
		} else if !feat.IsEnabled {
			decision.Reason = ReasonFeatureDisabled
		} else {
			decision.Reason = ReasonFeatureNotEntitled
		}
		resp.Capabilities[key] = decision
	}

	return resp, nil
}

// InvalidateCache clears the cached subscription state for a tenant upon plan updates.
func (s *Service) InvalidateCache(ctx context.Context, tenantID string) error {
	if s.redis == nil || s.redis.RDB == nil {
		return nil
	}
	key := cacheKeyPrefix + tenantID
	return s.redis.RDB.Del(ctx, key).Err()
}

// getSubscriptionCached attempts to retrieve from Redis with fallback to database.
func (s *Service) getSubscriptionCached(ctx context.Context, tenantID string) (*TenantSubscription, error) {
	cacheKey := cacheKeyPrefix + tenantID

	if s.redis != nil && s.redis.RDB != nil {
		cachedData, err := s.redis.RDB.Get(ctx, cacheKey).Bytes()
		if err == nil && len(cachedData) > 0 {
			var sub TenantSubscription
			if jsonErr := json.Unmarshal(cachedData, &sub); jsonErr == nil {
				return &sub, nil
			}
		}
	}

	// Fallback to PostgreSQL
	sub, err := s.repo.GetSubscriptionByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Populate Redis cache asynchronously/non-blocking
	if s.redis != nil && s.redis.RDB != nil && sub != nil {
		if data, jsonErr := json.Marshal(sub); jsonErr == nil {
			_ = s.redis.RDB.Set(ctx, cacheKey, data, subscriptionCacheTTL).Err()
		}
	}

	return sub, nil
}
