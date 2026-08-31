package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// placeholderProductsHandler is a Phase 1 verification stub.
// TODO(Phase 2): Replace with the real handler from internal/inventory package.
func placeholderProductsHandler(c *gin.Context) {
	connVal, exists := c.Get("db_conn")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_conn not found in context"})
		return
	}

	conn, ok := connVal.(*pgxpool.Conn)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid db_conn type"})
		return
	}

	rows, err := conn.Query(c.Request.Context(),
		"SELECT sku, name, unit_price, COALESCE(compliance_tags ? 'HALAL_MUI', FALSE) as is_halal_certified FROM products")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type ProductItem struct {
		SKU              string  `json:"sku"`
		Name             string  `json:"name"`
		UnitPrice        float64 `json:"unit_price"`
		IsHalalCertified bool    `json:"is_halal_certified"`
	}

	var products []ProductItem
	for rows.Next() {
		var p ProductItem
		if err := rows.Scan(&p.SKU, &p.Name, &p.UnitPrice, &p.IsHalalCertified); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		products = append(products, p)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    products,
		"meta":    gin.H{"total": len(products)},
	})
}

// placeholderManagerDashboardHandler is a Phase 1 RBAC verification stub.
// TODO(Phase 2): Replace with the real handler from internal/manager package.
func placeholderManagerDashboardHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"status": "Active", "message": "Manager Dashboard - Phase 2 will implement this"},
	})
}

// placeholderLedgerHandler is a Phase 1 RBAC verification stub.
// TODO(Phase 3): Replace with the real handler from internal/ledger package.
func placeholderLedgerHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"status": "Balanced", "message": "Finance Ledger - Phase 3 will implement this"},
	})
}
