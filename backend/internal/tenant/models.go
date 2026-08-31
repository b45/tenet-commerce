package tenant

import (
	"time"
)

// Tenant represents an enterprise tenant in the system.
// This data maps to the `public.tenants` registry table.
type Tenant struct {
	ID         string    `json:"id"`
	Slug       string    `json:"slug"`
	Name       string    `json:"company_name"`
	SchemaName string    `json:"schema_name"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
