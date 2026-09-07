package inventory

import (
	"time"

	"github.com/google/uuid"
)

// Movement Types
const (
	MovementTypeOpening    = "OPENING"
	MovementTypeIn         = "IN"
	MovementTypeOut        = "OUT"
	MovementTypeAdjustment = "ADJUSTMENT"
)

// Source Document Types
const (
	SourceDocOpeningBalance    = "OPENING_BALANCE"
	SourceDocGoodsReceipt      = "GOODS_RECEIPT"
	SourceDocPOSSale           = "POS_SALE"
	SourceDocPOSVoid           = "POS_VOID"
	SourceDocManualAdjustment  = "MANUAL_ADJUSTMENT"
)

// DefaultCoreLocation is the authoritative single operational warehouse location for core
const DefaultCoreLocation = "MAIN_STORE"

// StockMovement represents an immutable, append-only stock movement entry
type StockMovement struct {
	ID                   uuid.UUID  `json:"id"`
	ProductID            uuid.UUID  `json:"product_id"`
	WarehouseLocation    string     `json:"warehouse_location"`
	QuantityDelta        int        `json:"quantity_delta"`
	MovementType         string     `json:"movement_type"`
	SourceDocumentType   string     `json:"source_document_type"`
	SourceDocumentID     *uuid.UUID `json:"source_document_id,omitempty"`
	SourceDocumentLineID *uuid.UUID `json:"source_document_line_id,omitempty"`
	ActorID              *uuid.UUID `json:"actor_id,omitempty"`
	Reason               *string    `json:"reason,omitempty"`
	OccurredAt           time.Time  `json:"occurred_at"`
	CreatedAt            time.Time  `json:"created_at"`
}

// PostMovementParams contains required parameters to atomically post a stock movement
type PostMovementParams struct {
	ProductID            uuid.UUID
	WarehouseLocation    string
	QuantityDelta        int
	MovementType         string
	SourceDocumentType   string
	SourceDocumentID     *uuid.UUID
	SourceDocumentLineID *uuid.UUID
	ActorID              *uuid.UUID
	Reason               *string
	OccurredAt           time.Time
}

// CurrentBalance represents the calculated stock balance for a product at a given location
type CurrentBalance struct {
	ProductID         uuid.UUID `json:"product_id"`
	WarehouseLocation string    `json:"warehouse_location"`
	OnHandQuantity    int       `json:"on_hand_quantity"`
	OpeningQuantity   int       `json:"opening_quantity"`
	SubsequentDelta   int       `json:"subsequent_delta"`
}
