package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// FindSharedDocumentByPath is an optional exact-path lookup for sync callers.
// It returns sql.ErrNoRows when absent and an error rather than an arbitrary
// document when multiple orphan/live shared registrations have the same path.
func (s *PGVaultStore) FindSharedDocumentByPath(ctx context.Context, tenantID, path string) (*store.VaultDocument, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, fmt.Errorf("vault find shared document: tenant: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, tenant_id, agent_id, team_id, chat_id,
		scope, custom_scope, path, path_basename, title, doc_type, content_hash,
		summary, metadata, created_at, updated_at
		FROM vault_documents
		WHERE tenant_id = $1 AND scope = 'shared' AND agent_id IS NULL
		  AND team_id IS NULL AND path = $2
		LIMIT 2`, tid, path)
	if err != nil {
		return nil, fmt.Errorf("vault find shared document: %w", err)
	}
	defer rows.Close()
	var doc *store.VaultDocument
	for rows.Next() {
		if doc != nil {
			return nil, fmt.Errorf("vault find shared document: ambiguous shared path")
		}
		var row vaultDocRow
		if err := rows.Scan(&row.ID, &row.TenantID, &row.AgentID, &row.TeamID, &row.ChatID,
			&row.Scope, &row.CustomScope, &row.Path, &row.PathBasename, &row.Title,
			&row.DocType, &row.ContentHash, &row.Summary, &row.MetaJSON,
			&row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("vault find shared document: %w", err)
		}
		result := row.toVaultDocument()
		doc = &result
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vault find shared document: %w", err)
	}
	if doc == nil {
		return nil, sql.ErrNoRows
	}
	return doc, nil
}
