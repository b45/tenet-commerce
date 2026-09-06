package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/b45/tenet-commerce/backend/internal/manager"
	"github.com/b45/tenet-commerce/backend/internal/supplychain"
	"github.com/b45/tenet-commerce/backend/pkg/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var complianceDay = time.Date(2030, 4, 12, 0, 0, 0, 0, time.UTC)

func complianceConn(t *testing.T, db *database.PostgresDB) *pgxpool.Conn {
	t.Helper()
	conn, err := db.Pool.Acquire(context.Background())
	require.NoError(t, err)
	t.Cleanup(conn.Release)
	_, err = conn.Exec(context.Background(), "SET search_path TO tenant_al_barakah_mart")
	require.NoError(t, err)
	return conn
}

func complianceService(now func() time.Time) *supplychain.Service {
	return supplychain.NewServiceWithClock(supplychain.NewRepository(), ledger.NewService(ledger.NewRepository()), now)
}

func complianceFixture(t *testing.T, svc *supplychain.Service, conn *pgxpool.Conn) (*supplychain.Supplier, *supplychain.CreatePurchaseOrderRequest) {
	t.Helper()
	supplier, err := svc.CreateSupplier(context.Background(), conn, &supplychain.CreateSupplierRequest{
		Code: "CERT-" + uuid.NewString()[:8], CompanyName: "Compliance test supplier",
		ComplianceCertificate: &supplychain.CreateComplianceCertRequest{CertType: "HALAL_MUI", CertificateNumber: uuid.NewString(), IssuingAuthority: "BPJPH", Scope: "Food", ValidFrom: "2030-04-12", ExpiryDate: "2030-04-12"},
	})
	require.NoError(t, err)
	id := supplier.ComplianceCertificate.ID.String()
	return supplier, &supplychain.CreatePurchaseOrderRequest{SupplierID: supplier.ID.String(), ComplianceCertID: &id, Items: []supplychain.CreatePOItemRequest{{ProductID: "10000000-0000-0000-0000-000000000001", Quantity: 2, UnitCost: 10000}}}
}

func complianceReceipt(poID uuid.UUID) *supplychain.CreateGoodsReceiptRequest {
	return &supplychain.CreateGoodsReceiptRequest{PurchaseOrderID: poID.String(), Items: []supplychain.CreateGRItemRequest{{ProductID: "10000000-0000-0000-0000-000000000001", ReceivedQuantity: 1}}}
}

func TestCompliance_ExpiryAtReceiptAndHTTPRejections(t *testing.T) {
	db := newTestDatabase(t)
	conn := complianceConn(t, db)
	var clock atomic.Int64
	clock.Store(complianceDay.UnixNano())
	svc := complianceService(func() time.Time { return time.Unix(0, clock.Load()).UTC() })
	supplier, req := complianceFixture(t, svc, conn)
	po, err := svc.CreatePurchaseOrder(context.Background(), conn, req)
	require.NoError(t, err)
	router := setupSupplyChainTestRouter(t, db, svc)
	var beforeStock, beforeJournals int
	require.NoError(t, conn.QueryRow(context.Background(), "SELECT stock_quantity FROM inventory WHERE product_id = $1", req.Items[0].ProductID).Scan(&beforeStock))
	require.NoError(t, conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM ledger_entries").Scan(&beforeJournals))
	clock.Store(complianceDay.Add(24 * time.Hour).UnixNano())
	for _, path := range []string{"purchase-orders", "goods-receipts"} {
		var payload any = req
		if path == "goods-receipts" {
			payload = complianceReceipt(po.ID)
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)
		httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/"+path, bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httpReq)
		require.Equal(t, 422, w.Code, w.Body.String())
		var envelope apiResponseEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
		require.Equal(t, "COMPLIANCE_ERROR", envelope.Error.Code)
	}
	var stock, journals, receipts int
	require.NoError(t, conn.QueryRow(context.Background(), "SELECT stock_quantity FROM inventory WHERE product_id = $1", req.Items[0].ProductID).Scan(&stock))
	require.NoError(t, conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM ledger_entries").Scan(&journals))
	require.NoError(t, conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM goods_receipts WHERE purchase_order_id = $1", po.ID).Scan(&receipts))
	require.Equal(t, beforeStock, stock)
	require.Equal(t, beforeJournals, journals)
	require.Zero(t, receipts)
	var historic supplychain.ComplianceDecision
	require.NoError(t, conn.QueryRow(context.Background(), "SELECT decision FROM compliance_decisions WHERE document_type = 'PO' AND document_id = $1", po.ID).Scan(&historic))
	require.Equal(t, "ACCEPTED", historic.Outcome)
	require.Equal(t, complianceDay, historic.EvaluatedAt)
	require.Equal(t, supplier.ComplianceCertificate.ID, historic.Certificate.ID)
	_, err = conn.Exec(context.Background(), "UPDATE compliance_decisions SET decision = '{}' WHERE document_id = $1", po.ID)
	require.ErrorContains(t, err, "Compliance decisions are immutable")
}

func TestCompliance_StrictInvalidCertificateMatrix(t *testing.T) {
	db := newTestDatabase(t)
	conn := complianceConn(t, db)
	svc := complianceService(func() time.Time { return complianceDay })
	ctx := context.Background()
	for _, tc := range []struct {
		name, sql string
		want      error
	}{
		{"future", "UPDATE compliance_certificates SET valid_from = '2030-04-13' WHERE id = $1", supplychain.ErrComplianceCertExpired},
		{"revoked", "UPDATE compliance_certificates SET revoked_at = clock_timestamp() WHERE id = $1", supplychain.ErrComplianceCertInvalid},
		{"wrong type near expiry", "UPDATE compliance_certificates SET cert_type = 'ISO_9001' WHERE id = $1", supplychain.ErrComplianceCertInvalid},
		{"empty scope", "UPDATE compliance_certificates SET scope = '' WHERE id = $1", supplychain.ErrComplianceCertInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			supplier, req := complianceFixture(t, svc, conn)
			po, err := svc.CreatePurchaseOrder(ctx, conn, req)
			require.NoError(t, err)
			_, err = conn.Exec(ctx, tc.sql, supplier.ComplianceCertificate.ID)
			require.NoError(t, err)
			_, err = svc.CreatePurchaseOrder(ctx, conn, req)
			require.ErrorIs(t, err, tc.want)
			_, err = svc.CreateGoodsReceipt(ctx, conn, uuid.New(), uuid.NewString(), complianceReceipt(po.ID))
			require.ErrorIs(t, err, tc.want)
		})
	}
	supplier, req := complianceFixture(t, svc, conn)
	other, _ := complianceFixture(t, svc, conn)
	req.SupplierID = other.ID.String()
	_, err := svc.CreatePurchaseOrder(ctx, conn, req)
	require.ErrorIs(t, err, supplychain.ErrComplianceCertInvalid)
	req.SupplierID = supplier.ID.String()
	po, err := svc.CreatePurchaseOrder(ctx, conn, req)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, "UPDATE suppliers SET is_active = false WHERE id = $1", supplier.ID)
	require.NoError(t, err)
	_, err = svc.CreatePurchaseOrder(ctx, conn, req)
	require.ErrorIs(t, err, supplychain.ErrSupplierInactive)
	_, err = svc.CreateGoodsReceipt(ctx, conn, uuid.New(), uuid.NewString(), complianceReceipt(po.ID))
	require.ErrorIs(t, err, supplychain.ErrSupplierInactive)
}

// Wait for PostgreSQL's actual lock graph, not elapsed wall time, before releasing a blocker.
func waitComplianceLock(t *testing.T, conn *pgxpool.Conn, waiting, blocker uint32) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		var blocked bool
		err := conn.QueryRow(ctx, "SELECT $2::int = ANY(pg_blocking_pids($1::int))", waiting, blocker).Scan(&blocked)
		require.NoError(t, err, "expected database lock was not observed")
		if blocked {
			return
		}
	}
}

func TestCompliance_ConcurrentRevokeReceiptOrdering(t *testing.T) {
	for _, revokeFirst := range []bool{true, false} {
		name := "receipt first"
		if revokeFirst {
			name = "revoke first"
		}
		t.Run(name, func(t *testing.T) {
			db := newTestDatabase(t)
			observer := complianceConn(t, db)
			receiver := complianceConn(t, db)
			revoker := complianceConn(t, db)
			blocker := complianceConn(t, db)
			svc := complianceService(func() time.Time { return complianceDay })
			supplier, req := complianceFixture(t, svc, observer)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			po, err := svc.CreatePurchaseOrder(ctx, observer, req)
			require.NoError(t, err)
			tx, err := blocker.Begin(ctx)
			require.NoError(t, err)
			defer tx.Rollback(context.Background())
			var id uuid.UUID
			if revokeFirst {
				_, err = tx.Exec(ctx, "UPDATE compliance_certificates SET revoked_at = clock_timestamp() WHERE id = $1", supplier.ComplianceCertificate.ID)
			} else {
				err = tx.QueryRow(ctx, "SELECT product_id FROM inventory WHERE product_id = $1 FOR UPDATE", req.Items[0].ProductID).Scan(&id)
			}
			require.NoError(t, err)
			type result struct {
				gr  *supplychain.GoodsReceipt
				err error
			}
			grDone := make(chan result, 1)
			go func() {
				gr, err := svc.CreateGoodsReceipt(ctx, receiver, uuid.New(), uuid.NewString(), complianceReceipt(po.ID))
				grDone <- result{gr, err}
			}()
			waitComplianceLock(t, observer, receiver.Conn().PgConn().PID(), blocker.Conn().PgConn().PID())
			var revokeDone chan error
			if !revokeFirst {
				revokeDone = make(chan error, 1)
				go func() { revokeDone <- svc.RevokeCertificate(ctx, revoker, supplier.ComplianceCertificate.ID) }()
				waitComplianceLock(t, observer, revoker.Conn().PgConn().PID(), receiver.Conn().PgConn().PID())
			}
			require.NoError(t, tx.Commit(ctx))
			got := <-grDone
			if revokeFirst {
				require.ErrorIs(t, got.err, supplychain.ErrComplianceCertInvalid)
			} else {
				require.NoError(t, got.err)
				require.NoError(t, <-revokeDone)
				require.Nil(t, got.gr.ComplianceEvaluation.Certificate.RevokedAt)
				var decision supplychain.ComplianceDecision
				require.NoError(t, observer.QueryRow(ctx, "SELECT decision FROM compliance_decisions WHERE document_type = 'GR' AND document_id = $1", got.gr.ID).Scan(&decision))
				require.Equal(t, "ACCEPTED", decision.Outcome)
				require.Nil(t, decision.Certificate.RevokedAt)
			}
			certs, err := svc.GetSupplierCertificates(ctx, observer, supplier.ID)
			require.NoError(t, err)
			require.NotNil(t, certs[0].RevokedAt)
			require.Equal(t, complianceDay, certs[0].ExpiryDate, "revocation must preserve the original expiry")
			alerts, err := manager.NewRepository().GetComplianceAlerts(ctx, observer, 30)
			require.NoError(t, err)
			found := false
			for _, alert := range alerts.Items {
				if alert.CertificateID == supplier.ComplianceCertificate.ID.String() {
					found = true
					require.Equal(t, "REVOKED", alert.Status)
				}
			}
			require.True(t, found, "revocation must remain visible even when the original expiry is far away")
		})
	}
}

func TestCompliance_UpgradeRerunPreservesDecisions(t *testing.T) {
	db := newTestDatabase(t)
	conn := complianceConn(t, db)
	svc := complianceService(func() time.Time { return complianceDay })
	_, req := complianceFixture(t, svc, conn)
	po, err := svc.CreatePurchaseOrder(context.Background(), conn, req)
	require.NoError(t, err)
	// A populated pre-upgrade schema has neither revoked_at nor decision history.
	_, err = conn.Exec(context.Background(), `
		CREATE SCHEMA tenant_cert_upgrade;
		CREATE TABLE tenant_cert_upgrade.compliance_certificates (LIKE tenant_al_barakah_mart.compliance_certificates INCLUDING ALL);
		ALTER TABLE tenant_cert_upgrade.compliance_certificates DROP COLUMN revoked_at;
		INSERT INTO tenant_cert_upgrade.compliance_certificates (id, supplier_id, cert_type, certificate_number, issuing_authority, scope, valid_from, expiry_date)
		VALUES ('99999999-9999-9999-9999-999999999999', '99999999-9999-9999-9999-999999999998', 'HALAL_MUI', 'LEGACY-CERT', 'BPJPH', 'Food', '2020-01-01', '2040-01-01');
		INSERT INTO public.tenants (slug, company_name, schema_name) VALUES ('cert-upgrade', 'Certificate upgrade test', 'tenant_cert_upgrade');
	`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := conn.Exec(context.Background(), `DELETE FROM public.tenants WHERE slug = 'cert-upgrade'; DROP SCHEMA tenant_cert_upgrade CASCADE;`)
		require.NoError(t, err)
	})
	sql, err := os.ReadFile("../../scripts/06_certificate_validity.sql")
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		_, err = conn.Exec(context.Background(), string(sql))
		require.NoError(t, err)
	}
	var count int
	require.NoError(t, conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM compliance_decisions WHERE document_id = $1", po.ID).Scan(&count))
	require.Equal(t, 1, count)
	var expiry time.Time
	var revoked *time.Time
	require.NoError(t, conn.QueryRow(context.Background(), "SELECT expiry_date, revoked_at FROM tenant_cert_upgrade.compliance_certificates").Scan(&expiry, &revoked))
	require.Equal(t, "2040-01-01", expiry.Format(time.DateOnly))
	require.Nil(t, revoked)
	require.NoError(t, conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM tenant_cert_upgrade.compliance_decisions").Scan(&count))
	require.Zero(t, count, "upgrade must not fabricate historical decisions")
}

func TestCompliance_ConfigWriterSerializesWithReceipt(t *testing.T) {
	db := newTestDatabase(t)
	observer := complianceConn(t, db)
	receiver := complianceConn(t, db)
	writer := complianceConn(t, db)
	svc := complianceService(func() time.Time { return complianceDay })
	_, req := complianceFixture(t, svc, observer)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	po, err := svc.CreatePurchaseOrder(ctx, observer, req)
	require.NoError(t, err)
	var original []byte
	require.NoError(t, observer.QueryRow(ctx, "SELECT config_value FROM tenant_config WHERE config_key = 'compliance'").Scan(&original))
	defer func() {
		_, err := observer.Exec(context.Background(), "UPDATE tenant_config SET config_value = $1 WHERE config_key = 'compliance'", original)
		require.NoError(t, err)
	}()
	tx, err := writer.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(context.Background())
	_, err = tx.Exec(ctx, `UPDATE tenant_config SET config_value = '{"strict_compliance_mode":true,"required_compliance":["HALAL_OTHER"]}' WHERE config_key = 'compliance'`)
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() {
		_, err := svc.CreateGoodsReceipt(ctx, receiver, uuid.New(), uuid.NewString(), complianceReceipt(po.ID))
		done <- err
	}()
	waitComplianceLock(t, observer, receiver.Conn().PgConn().PID(), writer.Conn().PgConn().PID())
	require.NoError(t, tx.Commit(ctx))
	require.ErrorIs(t, <-done, supplychain.ErrComplianceCertInvalid)
}

func TestCompliance_ClockEvaluatedAfterWaitingForCertificate(t *testing.T) {
	db := newTestDatabase(t)
	observer := complianceConn(t, db)
	receiver := complianceConn(t, db)
	blocker := complianceConn(t, db)
	var clock atomic.Int64
	clock.Store(complianceDay.UnixNano())
	svc := complianceService(func() time.Time { return time.Unix(0, clock.Load()).UTC() })
	supplier, req := complianceFixture(t, svc, observer)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	po, err := svc.CreatePurchaseOrder(ctx, observer, req)
	require.NoError(t, err)
	tx, err := blocker.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(context.Background())
	var id uuid.UUID
	require.NoError(t, tx.QueryRow(ctx, "SELECT id FROM compliance_certificates WHERE id = $1 FOR UPDATE", supplier.ComplianceCertificate.ID).Scan(&id))
	done := make(chan error, 1)
	go func() {
		_, err := svc.CreateGoodsReceipt(ctx, receiver, uuid.New(), uuid.NewString(), complianceReceipt(po.ID))
		done <- err
	}()
	waitComplianceLock(t, observer, receiver.Conn().PgConn().PID(), blocker.Conn().PgConn().PID())
	clock.Store(complianceDay.Add(24 * time.Hour).UnixNano())
	require.NoError(t, tx.Commit(ctx))
	require.ErrorIs(t, <-done, supplychain.ErrComplianceCertExpired)
}

func TestCompliance_NonStrictWarningAndReadback(t *testing.T) {
	db := newTestDatabase(t)
	conn := complianceConn(t, db)
	_, err := conn.Exec(context.Background(), "SET search_path TO tenant_darussalam_store")
	require.NoError(t, err)
	svc := complianceService(func() time.Time { return complianceDay })
	supplier, err := svc.CreateSupplier(context.Background(), conn, &supplychain.CreateSupplierRequest{Code: "WARN-" + uuid.NewString()[:8], CompanyName: "Warning supplier"})
	require.NoError(t, err)
	po, err := svc.CreatePurchaseOrder(context.Background(), conn, &supplychain.CreatePurchaseOrderRequest{SupplierID: supplier.ID.String(), Items: []supplychain.CreatePOItemRequest{{ProductID: "20000000-0000-0000-0000-000000000001", Quantity: 1, UnitCost: 10000}}})
	require.NoError(t, err)
	require.Equal(t, "WARNING", po.ComplianceEvaluation.Outcome)
	detail, err := svc.GetPurchaseOrderDetail(context.Background(), conn, po.ID)
	require.NoError(t, err)
	require.Equal(t, po.ComplianceEvaluation, detail.ComplianceEvaluation)
	gr, err := svc.CreateGoodsReceipt(context.Background(), conn, uuid.New(), "warn-"+uuid.NewString(), &supplychain.CreateGoodsReceiptRequest{PurchaseOrderID: po.ID.String(), Items: []supplychain.CreateGRItemRequest{{ProductID: "20000000-0000-0000-0000-000000000001", ReceivedQuantity: 1}}})
	require.NoError(t, err)
	grDetail, err := svc.GetGoodsReceiptDetail(context.Background(), conn, gr.ID)
	require.NoError(t, err)
	require.Equal(t, gr.ComplianceEvaluation, grDetail.ComplianceEvaluation)
}
