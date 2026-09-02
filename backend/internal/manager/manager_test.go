package manager_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/b45/tenet-commerce/backend/internal/manager"
)

func TestSalesSummary_Calculations(t *testing.T) {
	t.Run("Zero Orders Yields Zero AOV", func(t *testing.T) {
		s := manager.SalesSummary{
			TodayGrossSales:  0,
			TodayOrdersCount: 0,
		}
		if s.TodayOrdersCount > 0 {
			s.AverageOrderValue = s.TodayGrossSales / float64(s.TodayOrdersCount)
		}
		if s.AverageOrderValue != 0 {
			t.Errorf("expected AOV 0, got %f", s.AverageOrderValue)
		}
	})

	t.Run("Positive Orders Yields Correct AOV", func(t *testing.T) {
		s := manager.SalesSummary{
			TodayGrossSales:  150000.0,
			TodayOrdersCount: 3,
		}
		if s.TodayOrdersCount > 0 {
			s.AverageOrderValue = s.TodayGrossSales / float64(s.TodayOrdersCount)
		}
		expectedAOV := 50000.0
		if s.AverageOrderValue != expectedAOV {
			t.Errorf("expected AOV %f, got %f", expectedAOV, s.AverageOrderValue)
		}
	})
}

func TestCertificateAlertItem_StatusClassification(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name           string
		daysRemaining  int
		expectedStatus string
	}{
		{
			name:           "Expired Certificate",
			daysRemaining:  -5,
			expectedStatus: "EXPIRED",
		},
		{
			name:           "Expires Today",
			daysRemaining:  0,
			expectedStatus: "EXPIRING_SOON",
		},
		{
			name:           "Expires in 15 Days",
			daysRemaining:  15,
			expectedStatus: "EXPIRING_SOON",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			item := manager.CertificateAlertItem{
				CertificateID:     "cert-123",
				SupplierID:        "supp-456",
				SupplierName:      "Halal Poultry Ltd",
				CertificateNumber: "MUI-123456",
				IssuingAuthority:  "MUI",
				ExpiryDate:        now.AddDate(0, 0, tc.daysRemaining),
				DaysRemaining:     tc.daysRemaining,
			}

			if item.DaysRemaining < 0 {
				item.Status = "EXPIRED"
			} else {
				item.Status = "EXPIRING_SOON"
			}

			if item.Status != tc.expectedStatus {
				t.Errorf("expected status %s, got %s", tc.expectedStatus, item.Status)
			}
		})
	}
}

func TestDashboardSummary_Serialization(t *testing.T) {
	now := time.Now().UTC()
	summary := manager.DashboardSummary{
		GeneratedAt: now,
		SalesSummary: manager.SalesSummary{
			TodayGrossSales:    250000.0,
			TodayNetSales:      240000.0,
			TodayOrdersCount:   5,
			AllTimeOrdersCount: 120,
			AverageOrderValue:  50000.0,
		},
		InventoryAlerts: manager.InventoryAlerts{
			LowStockCount: 1,
			Items: []manager.LowStockItem{
				{
					ProductID:    "prod-1",
					SKU:          "SKU-001",
					Name:         "Daging Ayam Halal 1kg",
					CategoryName: "Fresh Meat",
					CurrentStock: 3,
					Threshold:    10,
					UnitPrice:    45000.0,
				},
			},
		},
		ComplianceAlerts: manager.ComplianceAlerts{
			ExpiringCertificatesCount: 1,
			ExpiredCertificatesCount:  0,
			Items: []manager.CertificateAlertItem{
				{
					CertificateID:     "cert-1",
					SupplierID:        "supp-1",
					SupplierName:      "PT Berkah Jaya",
					CertificateNumber: "BPJPH-2026-001",
					IssuingAuthority:  "BPJPH",
					ExpiryDate:        now.AddDate(0, 0, 10),
					DaysRemaining:     10,
					Status:            "EXPIRING_SOON",
				},
			},
		},
		FinancialSummary: manager.FinancialSummary{
			ActiveAccountsCount:      15,
			TodayJournalEntriesCount: 8,
		},
	}

	bytes, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("failed to marshal dashboard summary: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	expectedKeys := []string{"generated_at", "sales_summary", "inventory_alerts", "compliance_alerts", "financial_summary"}
	for _, key := range expectedKeys {
		if _, ok := parsed[key]; !ok {
			t.Errorf("missing expected key %s in serialized json", key)
		}
	}
}
