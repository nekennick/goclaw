//go:build sqlite || sqliteonly

package sqlitestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// UpsertDocumentWithContent stages an existing document update under CAS before
// invoking write, then commits. write must return a restore function whenever it
// may have changed content, including when returning an error. The callback must
// not re-enter the store. Restore must not depend on the request context, which
// may already be canceled. Restore errors are joined with the original error.
// This coordinates runtime failures; it is not crash-atomic across DB and disk.
func (s *SQLiteVaultStore) UpsertDocumentWithContent(ctx context.Context, doc *store.VaultDocument, write func() (restore func() error, err error)) error {
	if doc.ID == "" || doc.UpdatedAt.IsZero() {
		return fmt.Errorf("vault content update: existing snapshot required: %w", sql.ErrNoRows)
	}
	if write == nil {
		return fmt.Errorf("vault content update: write callback required")
	}
	return s.upsertDocument(ctx, doc, write)
}

// commitVaultContent runs only after CAS and all document constraints succeed.
// The caller keeps its transaction rollback deferred until this returns.
func commitVaultContent(tx *sql.Tx, write func() (func() error, error)) error {
	var restore func() error
	var err error
	if write != nil {
		restore, err = write()
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		if restore != nil {
			if restoreErr := restore(); restoreErr != nil {
				err = errors.Join(err, fmt.Errorf("vault content restore: %w", restoreErr))
			}
		}
		return fmt.Errorf("vault content update: %w", err)
	}
	return nil
}
