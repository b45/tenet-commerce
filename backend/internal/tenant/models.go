package tenant

import (
	"regexp"
	"time"
)

var schemaNameRegex = regexp.MustCompile(`^[a-z0-9_]+$`)

// IsValidSchemaName ensures the tenant schema name conforms to strict identifier rules
// and avoids unsafe characters before being used in search_path queries.
func IsValidSchemaName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}
	return schemaNameRegex.MatchString(name)
}

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
