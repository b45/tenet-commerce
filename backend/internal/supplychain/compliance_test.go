package supplychain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCompliancePolicyBoundaries(t *testing.T) {
	supplier := uuid.New()
	day := time.Date(2030, 4, 12, 0, 0, 0, 0, time.UTC)
	config := map[string]interface{}{"strict_compliance_mode": true, "required_compliance": []interface{}{"HALAL_MUI"}}
	for _, tc := range []struct {
		name   string
		now    time.Time
		change func(*ComplianceCertificate)
		want   error
		status string
	}{
		{"start inclusive", day, nil, nil, "EXPIRING_SOON"},
		{"expiry end inclusive", day.Add(24*time.Hour - time.Nanosecond), nil, nil, "EXPIRING_SOON"},
		{"next day expired", day.Add(24 * time.Hour), nil, ErrComplianceCertExpired, "EXPIRED"},
		{"future", day.Add(-time.Nanosecond), nil, ErrComplianceCertExpired, "NOT_YET_VALID"},
		{"timezone is UTC", day.In(time.FixedZone("UTC+7", 7*3600)), nil, nil, "EXPIRING_SOON"},
		{"revoked", day, func(c *ComplianceCertificate) { c.RevokedAt = &day }, ErrComplianceCertInvalid, "REVOKED"},
		{"wrong supplier", day, func(c *ComplianceCertificate) { c.SupplierID = uuid.New() }, ErrComplianceCertInvalid, "EXPIRING_SOON"},
		{"wrong type expiring soon", day, func(c *ComplianceCertificate) { c.CertType = "ISO_9001" }, ErrComplianceCertInvalid, "EXPIRING_SOON"},
		{"empty scope", day, func(c *ComplianceCertificate) { c.Scope = " " }, ErrComplianceCertInvalid, "EXPIRING_SOON"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cert := &ComplianceCertificate{SupplierID: supplier, CertType: "HALAL_MUI", Scope: "Food", ValidFrom: day, ExpiryDate: day}
			if tc.change != nil {
				tc.change(cert)
			}
			d, err := evaluateCompliance(supplier, config, cert, tc.now)
			require.ErrorIs(t, err, tc.want)
			require.Equal(t, tc.status, d.Certificate.ComputedStatus)
			require.Equal(t, tc.now.UTC(), d.EvaluatedAt)
			if tc.want != nil {
				require.Equal(t, "BLOCKED", d.Outcome)
			}
		})
	}
}

func TestComplianceWarningAndMissingPolicy(t *testing.T) {
	now := time.Date(2030, 4, 12, 0, 0, 0, 0, time.UTC)
	supplier := uuid.New()
	d, err := evaluateCompliance(supplier, nil, nil, now)
	require.NoError(t, err)
	require.Equal(t, "WARNING", d.Outcome)
	d, err = evaluateCompliance(supplier, map[string]interface{}{"strict_compliance_mode": true}, nil, now)
	require.ErrorIs(t, err, ErrComplianceCertRequired)
	require.Equal(t, "BLOCKED", d.Outcome)
	cert := &ComplianceCertificate{SupplierID: supplier, CertType: "HALAL_MUI", Scope: "Food", ValidFrom: now, ExpiryDate: now}
	_, err = evaluateCompliance(supplier, map[string]interface{}{"strict_compliance_mode": true}, cert, now)
	require.ErrorIs(t, err, ErrComplianceCertInvalid, "strict mode must fail closed without required types")
}
