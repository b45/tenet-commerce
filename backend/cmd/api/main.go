package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	
	"github.com/b45/tenet-commerce/backend/pkg/database"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
)

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8081"
	}

	// Initialize Database Connection Pool
	ctx := context.Background()
	db, err := database.NewPostgresDB(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Repositories
	tenantRepo := tenant.NewRepository(db)

	// Setup Gin Engine
	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("APP_DEBUG") == "true" {
		gin.SetMode(gin.DebugMode)
	}
	
	router := gin.Default()

	// ---------------------------------------------------------
	// PUBLIC ROUTES
	// ---------------------------------------------------------
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "tenet-commerce-api",
		})
	})

	// ---------------------------------------------------------
	// PROTECTED MULTI-TENANT ROUTES
	// ---------------------------------------------------------
	v1 := router.Group("/api/v1")
	
	// Apply Schema-per-Tenant Middleware
	v1.Use(tenant.ContextMiddleware(db, tenantRepo))

	v1.GET("/tenant/info", func(c *gin.Context) {
		t, exists := c.Get("tenant")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "tenant context lost"})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": t,
			"message": "You are currently querying the isolated schema for this tenant.",
		})
	})

	// Demonstrate schema-isolated query:
	// This query runs 'SELECT * FROM products' without schema prefix.
	// Because of 'SET search_path', PostgreSQL automatically fetches from tenant_{slug}.products!
	v1.GET("/products", func(c *gin.Context) {
		connVal, exists := c.Get("db_conn")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection context lost"})
			return
		}
		
		conn, ok := connVal.(*pgxpool.Conn)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid connection type"})
			return
		}

		rows, err := conn.Query(c.Request.Context(), "SELECT sku, name, unit_price, is_halal_certified FROM products")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed scanning product"})
				return
			}
			products = append(products, p)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    products,
			"total":   len(products),
		})
	})

	// Start API Server
	log.Printf("Tenet Commerce API starting on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

