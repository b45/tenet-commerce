package supplychain

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ComplianceDecision is a historical decision, not a promise of future validity.
// Calendar dates are inclusive in UTC; the clock is sampled after acquiring locks.
type ComplianceDecision struct {
	Policy      string                 `json:"policy"`
	EvaluatedAt time.Time              `json:"evaluated_at"`
	Strict      bool                   `json:"strict"`
	SupplierID  uuid.UUID              `json:"supplier_id"`
	Certificate *ComplianceCertificate `json:"certificate,omitempty"`
	Required    []string               `json:"required_types"`
	Outcome     string                 `json:"outcome"`
	Reason      string                 `json:"reason,omitempty"`
}

func certificateStatus(cert *ComplianceCertificate, now time.Time) string {
	day := now.UTC().Format(time.DateOnly)
	switch {
	case cert.RevokedAt != nil:
		return "REVOKED"
	case cert.ValidFrom.Format(time.DateOnly) > day:
		return "NOT_YET_VALID"
	case cert.ExpiryDate.Format(time.DateOnly) < day:
		return "EXPIRED"
	case cert.ExpiryDate.Format(time.DateOnly) <= now.UTC().AddDate(0, 0, 30).Format(time.DateOnly):
		return "EXPIRING_SOON"
	default:
		return "VALID"
	}
}

func evaluateCompliance(supplierID uuid.UUID, config map[string]interface{}, cert *ComplianceCertificate, now time.Time) (*ComplianceDecision, error) {
	strict, _ := config["strict_compliance_mode"].(bool)
	d := &ComplianceDecision{Policy: "UTC_DATE_INCLUSIVE_V1", EvaluatedAt: now.UTC(), Strict: strict, SupplierID: supplierID, Certificate: cert, Required: []string{}, Outcome: "ACCEPTED"}
	if values, ok := config["required_compliance"].([]interface{}); ok {
		for _, value := range values {
			if name, ok := value.(string); ok {
				d.Required = append(d.Required, name)
			}
		}
	}
	var failure error
	if cert == nil {
		failure = ErrComplianceCertRequired
	} else {
		cert.ComputedStatus = certificateStatus(cert, now)
		allowed := false
		for _, name := range d.Required {
			allowed = allowed || name == cert.CertType
		}
		switch {
		case cert.SupplierID != supplierID, strings.TrimSpace(cert.Scope) == "", !allowed:
			failure = ErrComplianceCertInvalid
		case cert.ComputedStatus == "REVOKED":
			failure = ErrComplianceCertInvalid
		case cert.ComputedStatus == "NOT_YET_VALID", cert.ComputedStatus == "EXPIRED":
			failure = ErrComplianceCertExpired
		}
	}
	if failure != nil {
		d.Reason = failure.Error()
		if strict {
			d.Outcome = "BLOCKED"
			return d, failure
		}
		d.Outcome = "WARNING"
	}
	return d, nil
}

func (s *Service) checkCompliance(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID, certID *uuid.UUID) (*ComplianceDecision, error) {
	// SHARE also protects an absent configuration row against concurrent INSERT.
	// Order: PO (receipts only), config table, supplier, certificate, inventory.
	if _, err := tx.Exec(ctx, `LOCK TABLE tenant_config IN SHARE MODE`); err != nil {
		return nil, err
	}
	config, err := s.repo.GetTenantConfig(ctx, tx, "compliance")
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	var active bool
	if err := tx.QueryRow(ctx, `SELECT is_active FROM suppliers WHERE id = $1 FOR UPDATE`, supplierID).Scan(&active); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrComplianceCertInvalid
		}
		return nil, err
	}
	if !active {
		return nil, ErrSupplierInactive
	}
	var cert *ComplianceCertificate
	if certID != nil {
		cert, err = s.repo.GetComplianceCertificateByID(ctx, tx, *certID)
		if errors.Is(err, ErrNotFound) {
			return nil, ErrComplianceCertInvalid
		}
		if err != nil {
			return nil, err
		}
		// Ownership must also hold in warning mode: do not record another supplier's certificate.
		if cert.SupplierID != supplierID {
			return nil, ErrComplianceCertInvalid
		}
	}
	return evaluateCompliance(supplierID, config, cert, s.now())
}

func (r *Repository) recordComplianceDecision(ctx context.Context, tx pgx.Tx, kind string, id uuid.UUID, decision *ComplianceDecision) error {
	_, err := tx.Exec(ctx, `INSERT INTO compliance_decisions (document_type, document_id, decision) VALUES ($1, $2, $3)`, kind, id, decision)
	return err
}

func (r *Repository) getComplianceDecision(ctx context.Context, db queryRower, kind string, id uuid.UUID) (*ComplianceDecision, error) {
	var decision *ComplianceDecision
	err := db.QueryRow(ctx, `SELECT decision FROM compliance_decisions WHERE document_type = $1 AND document_id = $2`, kind, id).Scan(&decision)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // Historical documents predate recorded evaluations.
	}
	return decision, err
}
