package ledger

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes mounts all ledger accounting endpoints with RBAC permission guards
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/accounts",
		internalAuth.RequirePermission("ledger:read"),
		h.GetChartOfAccounts,
	)
	rg.GET("/entries",
		internalAuth.RequirePermission("ledger:read"),
		h.GetJournalEntries,
	)
	rg.POST("/entries",
		internalAuth.RequirePermission("ledger:write"),
		h.CreateManualEntry,
	)
	rg.GET("/trial-balance",
		internalAuth.RequirePermission("ledger:read"),
		h.GetTrialBalance,
	)
}

// GetChartOfAccounts handles GET /api/v1/ledger/accounts
func (h *Handler) GetChartOfAccounts(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	connVal, exists := c.Get("db_conn")
	if !exists {
		log.Error("Database connection context not found during COA fetch")
		response.InternalServerError(c, "DATABASE_CONTEXT_LOST", "Database connection context not found")
		return
	}
	conn := connVal.(*pgxpool.Conn)

	accounts, err := h.service.GetChartOfAccounts(c.Request.Context(), conn)
	if err != nil {
		log.Error("Failed to fetch chart of accounts", "error", err)
		response.InternalServerError(c, "COA_FETCH_FAILED", err.Error())
		return
	}

	log.Debug("Chart of accounts retrieved successfully", "count", len(accounts))
	response.OKWithMeta(c, accounts, response.Meta{Total: len(accounts)})
}

// GetJournalEntries handles GET /api/v1/ledger/entries
func (h *Handler) GetJournalEntries(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	connVal, exists := c.Get("db_conn")
	if !exists {
		log.Error("Database connection context not found during journal entries fetch")
		response.InternalServerError(c, "DATABASE_CONTEXT_LOST", "Database connection context not found")
		return
	}
	conn := connVal.(*pgxpool.Conn)

	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	entries, err := h.service.GetJournalEntries(c.Request.Context(), conn, limit, offset)
	if err != nil {
		log.Error("Failed to fetch journal entries", "error", err, "limit", limit, "offset", offset)
		response.InternalServerError(c, "JOURNAL_FETCH_FAILED", err.Error())
		return
	}

	log.Debug("Journal entries retrieved", "count", len(entries), "limit", limit, "offset", offset)
	response.OKWithMeta(c, entries, response.Meta{
		Limit:  limit,
		Offset: offset,
		Total:  len(entries),
	})
}

// CreateManualEntry handles POST /api/v1/ledger/entries
func (h *Handler) CreateManualEntry(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	connVal, exists := c.Get("db_conn")
	if !exists {
		log.Error("Database connection context not found during manual entry creation")
		response.InternalServerError(c, "DATABASE_CONTEXT_LOST", "Database connection context not found")
		return
	}
	conn := connVal.(*pgxpool.Conn)

	var req CreateEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Manual journal entry validation failed", "error", err)
		response.BadRequest(c, "VALIDATION_ERROR", "Invalid manual entry format", err.Error())
		return
	}

	// Only MANUAL_ADJUSTMENT is allowed through this endpoint
	if req.SourceDocumentType != SourceDocManualAdjustment {
		log.Warn("Manual entry rejected: invalid source document type", "type", req.SourceDocumentType)
		response.BadRequest(c, "INVALID_SOURCE_DOC", "Only MANUAL_ADJUSTMENT is allowed for manual entries")
		return
	}

	entry, err := h.service.CreateManualEntry(c.Request.Context(), conn, &req)
	if err != nil {
		if errors.Is(err, ErrUnbalancedEntry) || errors.Is(err, ErrZeroAmountEntry) || errors.Is(err, ErrInsufficientLines) {
			log.Warn("Manual journal entry violated double-entry balancing invariant", "error", err)
			response.UnprocessableEntity(c, "LEDGER_INVARIANT_VIOLATION", err.Error())
			return
		}
		log.Error("Failed to post manual journal entry", "error", err)
		response.InternalServerError(c, "ENTRY_CREATION_FAILED", err.Error())
		return
	}

	log.Info("Manual journal entry successfully committed and verified",
		"entry_id", entry.ID,
		"entry_number", entry.EntryNumber,
		"total_debit", entry.TotalDebit,
		"total_credit", entry.TotalCredit,
		"line_count", len(entry.Lines),
	)
	response.Created(c, entry)
}

// GetTrialBalance handles GET /api/v1/ledger/trial-balance
func (h *Handler) GetTrialBalance(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	connVal, exists := c.Get("db_conn")
	if !exists {
		log.Error("Database connection context not found during trial balance generation")
		response.InternalServerError(c, "DATABASE_CONTEXT_LOST", "Database connection context not found")
		return
	}
	conn := connVal.(*pgxpool.Conn)

	asOfDate := c.Query("as_of_date")

	summary, err := h.service.GetTrialBalance(c.Request.Context(), conn, asOfDate)
	if err != nil {
		log.Error("Failed to generate trial balance", "error", err, "as_of_date", asOfDate)
		response.InternalServerError(c, "TRIAL_BALANCE_FAILED", err.Error())
		return
	}

	log.Info("Trial balance generated successfully",
		"as_of_date", summary.AsOfDate,
		"total_debits", summary.TotalDebits,
		"total_credits", summary.TotalCredits,
		"is_balanced", summary.IsBalanced,
	)
	response.OK(c, summary)
}
