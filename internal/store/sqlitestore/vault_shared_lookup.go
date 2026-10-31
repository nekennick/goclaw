//go:build sqlite || sqliteonly

package sqlitestore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// FindSharedDocumentByPath is an optional exact-path lookup for sync callers.
// It returns sql.ErrNoRows when absent and an error rather than an arbitrary
// document when multiple orphan/live shared registrations have the same path.
func (s *SQLiteVaultStore) FindSharedDocumentByPath(ctx context.Context, tenantID, path string) (*store.VaultDocument, error) {

	rows, err := s.db.QueryContext(ctx, `SELECT id, tenant_id, agent_id, team_id, chat_id,
		scope, custom_scope, path, path_basename, title, doc_type, content_hash,
		summary, metadata, created_at, updated_at
		FROM vault_documents
		WHERE tenant_id = ? AND scope = 'shared' AND agent_id IS NULL
		  AND team_id IS NULL AND path = ?
		LIMIT 2`, tenantID, path)
	if err != nil {
		return nil, fmt.Errorf("vault find shared document: %w", err)
	}
	defer rows.Close()
	var doc *store.VaultDocument
	for rows.Next() {
		if doc != nil {
			return nil, fmt.Errorf("vault find shared document: ambiguous shared path")
		}
		doc, err = scanVaultDocRow(rows)
		if err != nil {
			return nil, fmt.Errorf("vault find shared document: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vault find shared document: %w", err)
	}
	if doc == nil {
		return nil, sql.ErrNoRows
	}
	return doc, nil
}
