package ledger

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	pkgIdempotency "github.com/b45/tenet-commerce/backend/pkg/idempotency"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	pkgRedis "github.com/b45/tenet-commerce/backend/pkg/redis"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes mounts all ledger accounting endpoints with RBAC permission guards
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, rdb *pkgRedis.Client) {
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
		pkgIdempotency.DurableIdempotencyMiddleware(rdb, 24*time.Hour),
		h.CreateManualEntry,
	)
	rg.POST("/entries/:id/reverse",
		internalAuth.RequirePermission("ledger:write"),
		pkgIdempotency.DurableIdempotencyMiddleware(rdb, 24*time.Hour),
		h.ReverseEntry,
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

// ReverseEntry handles POST /api/v1/ledger/entries/:id/reverse
func (h *Handler) ReverseEntry(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	connVal, exists := c.Get("db_conn")
	if !exists {
		log.Error("Database connection context not found during journal entry reversal")
		response.InternalServerError(c, "DATABASE_CONTEXT_LOST", "Database connection context not found")
		return
	}
	conn := connVal.(*pgxpool.Conn)

	idStr := c.Param("id")
	entryID, err := uuid.Parse(idStr)
	if err != nil {
		log.Warn("Invalid journal entry UUID in reversal request", "id", idStr)
		response.BadRequest(c, "INVALID_ENTRY_ID", "Invalid journal entry ID format")
		return
	}

	var req ReverseEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid reversal request payload", "error", err)
		response.BadRequest(c, "VALIDATION_ERROR", "Invalid reversal request payload", err.Error())
		return
	}

	reversalEntry, err := h.service.ReverseManualEntry(c.Request.Context(), conn, entryID, req.Reason)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Warn("Attempted to reverse non-existent journal entry", "entry_id", entryID)
			response.NotFound(c, "ENTRY_NOT_FOUND", "Journal entry not found")
			return
		}
		if errors.Is(err, ErrAlreadyReversed) {
			log.Warn("Attempted to reverse already reversed journal entry", "entry_id", entryID)
			response.UnprocessableEntity(c, "ENTRY_ALREADY_REVERSED", "Journal entry has already been reversed")
			return
		}
		log.Error("Failed to reverse journal entry", "entry_id", entryID, "error", err)
		response.InternalServerError(c, "REVERSAL_FAILED", err.Error())
		return
	}

	log.Info("Journal entry reversed successfully",
		"original_entry_id", entryID,
		"reversal_entry_id", reversalEntry.ID,
		"reversal_entry_number", reversalEntry.EntryNumber,
	)
	response.Created(c, reversalEntry)
}
