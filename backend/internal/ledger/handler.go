package ledger

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetChartOfAccounts handles GET /api/v1/ledger/accounts
func (h *Handler) GetChartOfAccounts(c *gin.Context) {
	connVal, exists := c.Get("db_conn")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_conn not found in context"})
		return
	}
	conn := connVal.(*pgxpool.Conn)

	accounts, err := h.service.GetChartOfAccounts(c.Request.Context(), conn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    accounts,
		"meta":    gin.H{"total": len(accounts)},
	})
}

// GetJournalEntries handles GET /api/v1/ledger/entries
func (h *Handler) GetJournalEntries(c *gin.Context) {
	connVal, exists := c.Get("db_conn")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_conn not found in context"})
		return
	}
	conn := connVal.(*pgxpool.Conn)

	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	entries, err := h.service.GetJournalEntries(c.Request.Context(), conn, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    entries,
		"meta":    gin.H{"limit": limit, "offset": offset, "total": len(entries)},
	})
}

// CreateManualEntry handles POST /api/v1/ledger/entries
func (h *Handler) CreateManualEntry(c *gin.Context) {
	connVal, exists := c.Get("db_conn")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_conn not found in context"})
		return
	}
	conn := connVal.(*pgxpool.Conn)

	var req CreateEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Only MANUAL_ADJUSTMENT is allowed through this endpoint
	if req.SourceDocumentType != SourceDocManualAdjustment {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only MANUAL_ADJUSTMENT is allowed for manual entries"})
		return
	}

	entry, err := h.service.CreateManualEntry(c.Request.Context(), conn, &req)
	if err != nil {
		// Differentiate validation errors from system errors
		if err == ErrUnbalancedEntry || err == ErrZeroAmountEntry || err == ErrInsufficientLines {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    entry,
	})
}

// GetTrialBalance handles GET /api/v1/ledger/trial-balance
func (h *Handler) GetTrialBalance(c *gin.Context) {
	connVal, exists := c.Get("db_conn")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_conn not found in context"})
		return
	}
	conn := connVal.(*pgxpool.Conn)

	asOfDate := c.Query("as_of_date")

	summary, err := h.service.GetTrialBalance(c.Request.Context(), conn, asOfDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}
