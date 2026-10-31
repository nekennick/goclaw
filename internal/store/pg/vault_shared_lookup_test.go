package pg

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

func TestFindSharedDocumentByPath(t *testing.T) {
	db, vs, tenant, agent, team, otherTenant := vaultIdentityFixture(t)
	ctx := context.Background()
	finder, ok := vs.(interface {
		FindSharedDocumentByPath(context.Context, string, string) (*store.VaultDocument, error)
	})
	if !ok {
		t.Fatal("concrete store lacks optional shared lookup")
	}
	const path = "exact/note.md"
	shared := &store.VaultDocument{TenantID: tenant, Scope: "shared", Path: path,
		Title: "Shared", DocType: "note", Metadata: map[string]any{"key": "preserve"}}
	for _, doc := range []*store.VaultDocument{
		{TenantID: tenant, Scope: "personal", AgentID: &agent, Path: path, DocType: "note"},
		{TenantID: tenant, Scope: "team", TeamID: &team, Path: path, DocType: "note"},
		{TenantID: tenant, Scope: "custom", Path: path, DocType: "note"},
		{TenantID: otherTenant, Scope: "shared", Path: path, DocType: "note"},
		{TenantID: tenant, Scope: "shared", Path: path + ".backup", DocType: "note"},
		shared,
	} {
		if err := vs.UpsertDocument(ctx, doc); err != nil {
			t.Fatal(err)
		}
	}
	got, err := finder.FindSharedDocumentByPath(ctx, tenant, path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := vs.GetDocumentByID(ctx, tenant, shared.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) || got.UpdatedAt.IsZero() {
		t.Fatalf("lookup returned wrong document or lost CAS token: %+v", got)
	}
	for _, missing := range []string{"note.md", "exact/Note.md", "absent.md"} {
		if got, err := finder.FindSharedDocumentByPath(ctx, tenant, missing); got != nil || !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("exact missing path %q: got=%+v err=%v", missing, got, err)
		}
	}

	// PG permits multiple orphan registrations. SQLite's current unique index
	// prevents them, so emulate a legacy/corrupt duplicate only in this temp DB.
	_, err = db.Exec(`INSERT INTO vault_documents (id, tenant_id, scope, path, orphaned_from_id)
		VALUES ($1, $2, 'shared', $3, $4)`, uuid.NewString(), tenant, path, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	if got, err := finder.FindSharedDocumentByPath(ctx, tenant, path); got != nil || err == nil || errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("ambiguous registration must fail closed: got=%+v err=%v", got, err)
	}
}
