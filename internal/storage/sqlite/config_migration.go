package sqlite

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/steveyegge/beads/internal/config"
	"github.com/steveyegge/beads/internal/utils"
)

// MigrateConfigToYAML automatically migrates issue_prefix from DB to config.yaml
// if config.yaml doesn't have it set. This provides backward compatibility
// for users upgrading from pre-GH-209 versions.
// Called automatically on storage initialization.
func MigrateConfigToYAML(ctx context.Context, db *sql.DB) error {
	// Check if config.yaml already has issue-prefix set
	configPrefix := config.GetIssuePrefix()
	if configPrefix != "" {
		// Already set, no migration needed
		return nil
	}

	// Try to get prefix from database
	var dbPrefix string
	err := db.QueryRowContext(ctx, `SELECT value FROM config WHERE key = ?`, "issue_prefix").Scan(&dbPrefix)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if dbPrefix == "" {
		// No prefix in DB either - try to detect from existing issues
		var firstIssueID string
		err = db.QueryRowContext(ctx, `SELECT id FROM issues ORDER BY created_at LIMIT 1`).Scan(&firstIssueID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		
		if firstIssueID != "" {
			dbPrefix = utils.ExtractIssuePrefix(firstIssueID)
		}
	}

	// If we found a prefix, migrate it to config.yaml
	if dbPrefix != "" {
		if err := config.SetIssuePrefix(dbPrefix); err != nil {
			// Log warning but don't fail - might be test environment
			if os.Getenv("BD_DEBUG") != "" {
				log.Printf("Warning: failed to migrate issue_prefix to config.yaml: %v\n", err)
			}
			return nil
		}
		
		if os.Getenv("BD_DEBUG") != "" {
			log.Printf("Migrated issue_prefix to config.yaml: %s\n", dbPrefix)
		}
	}

	return nil
}
