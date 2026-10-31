package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// These fixtures characterize ID-less conflict upserts and exercise supplied-ID
// mutations against real constraints, rather than a fake VaultStore.
func TestVaultUpsertDocumentIdentity(t *testing.T) {
	db, vs, tenant, agent, team, otherTenant := vaultIdentityFixture(t)
	ctx := context.Background()
	exec := func(query string, args ...any) {
		t.Helper()
		n := 0
		query = strings.ReplaceAll(query, "?", "%s")
		parts := strings.Split(query, "%s")
		query = parts[0]
		for _, part := range parts[1:] {
			n++
			query += fmt.Sprintf("$%d", n) + part
		}
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("exec %s: %v", query, err)
		}
	}
	read := func(id string) *store.VaultDocument {
		t.Helper()
		got, err := vs.GetDocumentByID(ctx, tenant, id)
		if err != nil {
			t.Fatalf("read %s: %v", id, err)
		}
		return got
	}
	upsert := func(doc *store.VaultDocument) {
		t.Helper()
		if err := vs.UpsertDocument(ctx, doc); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}
	newDoc := func(path string) *store.VaultDocument {
		return &store.VaultDocument{TenantID: tenant, AgentID: &agent,
			Scope: "personal", Path: path, Title: "original", DocType: "note",
			ContentHash: "hash-1", Summary: "summary-1", Metadata: map[string]any{"key": "original"}}
	}

	t.Run("ID-less conflict preserves identity and links", func(t *testing.T) {
		original, peer := newDoc("baseline.md"), newDoc("baseline-peer.md")
		upsert(original)
		upsert(peer)
		exec("INSERT INTO vault_links (id, from_doc_id, to_doc_id) VALUES (?, ?, ?)",
			uuid.NewString(), original.ID, peer.ID)
		before := read(original.ID)
		replacement := newDoc(original.Path)
		replacement.Title = "replacement"
		upsert(replacement)
		after := read(original.ID)
		if replacement.ID != original.ID || after.Title != "replacement" || !after.CreatedAt.Equal(before.CreatedAt) {
			t.Fatalf("ID-less conflict changed identity/creation time or missed content: %+v", after)
		}
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM vault_links WHERE from_doc_id = $1 AND to_doc_id = $2", original.ID, peer.ID).Scan(&count); err != nil || count != 1 {
			t.Fatalf("links after conflict: count=%d err=%v", count, err)
		}
	})

	t.Run("supplied ID moves normalized ownership and preserves references", func(t *testing.T) {
		doc, peer := newDoc("original.md"), newDoc("peer.md")
		upsert(doc)
		upsert(peer)
		id := doc.ID
		before := read(id)
		for _, pair := range [][2]string{{id, peer.ID}, {peer.ID, id}} {
			exec("INSERT INTO vault_links (id, from_doc_id, to_doc_id) VALUES (?, ?, ?)",
				uuid.NewString(), pair[0], pair[1])
		}
		exec("INSERT INTO vault_versions (doc_id, content) VALUES (?, 'version content')", id)
		exec("INSERT INTO vault_document_chunks (tenant_id, document_id, chunk_index, text) VALUES (?, ?, 0, 'chunk body')", tenant, id)
		doc.Path = "folder/Renamed.MD"
		doc.PathBasename = "original.md" // stale value from GetDocumentByID
		doc.Title, doc.DocType = "renamed", "reference"
		doc.ContentHash, doc.Summary = "hash-2", "summary-2"
		doc.Metadata = map[string]any{"key": "updated"}
		chat := "chat-1"
		doc.Scope, doc.AgentID, doc.TeamID, doc.ChatID = "team", nil, &team, &chat
		upsert(doc)
		got := read(id)
		if doc.ID != id || got.Path != doc.Path || got.PathBasename != "renamed.md" ||
			got.Scope != "team" || got.AgentID != nil || got.TeamID == nil || *got.TeamID != team ||
			got.ChatID == nil || *got.ChatID != chat || got.Title != doc.Title || got.DocType != doc.DocType ||
			got.ContentHash != doc.ContentHash || got.Summary != doc.Summary ||
			!reflect.DeepEqual(got.Metadata, doc.Metadata) || !got.CreatedAt.Equal(before.CreatedAt) ||
			got.UpdatedAt.Before(before.UpdatedAt) {
			t.Fatalf("identity/fields not preserved or updated: %+v", got)
		}
		custom := "project"
		doc.Scope, doc.TeamID, doc.CustomScope, doc.ChatID = "custom", nil, &custom, nil
		upsert(doc)
		got = read(id)
		if got.Scope != "custom" || got.CustomScope == nil || *got.CustomScope != custom || got.ChatID != nil || got.TeamID != nil {
			t.Fatalf("custom scope / nullable fields not updated: %+v", got)
		}
		doc.Scope, doc.CustomScope = "shared", nil
		upsert(doc)
		got = read(id)
		if got.Scope != "shared" || got.AgentID != nil || got.TeamID != nil || got.CustomScope != nil {
			t.Fatalf("shared normalization not persisted: %+v", got)
		}
		doc.Scope, doc.AgentID = "personal", &agent
		upsert(doc)
		got = read(id)
		if doc.ID != id || got.Scope != "personal" || got.AgentID == nil || *got.AgentID != agent {
			t.Fatalf("return to personal failed: %+v", got)
		}
		var links int
		query := "SELECT COUNT(*) FROM vault_links WHERE from_doc_id = ? OR to_doc_id = ?"
		query = strings.Replace(query, "?", "$1", 1)
		query = strings.Replace(query, "?", "$2", 1)
		if err := db.QueryRowContext(ctx, query, id, id).Scan(&links); err != nil || links != 2 {
			t.Fatalf("incoming/outgoing links lost: count=%d err=%v", links, err)
		}
		for _, query := range []string{
			"SELECT COUNT(*) FROM vault_versions WHERE doc_id = $1",
			"SELECT COUNT(*) FROM vault_document_chunks WHERE document_id = $1",
		} {
			var count int
			if err := db.QueryRowContext(ctx, query, id).Scan(&count); err != nil || count != 1 {
				t.Fatalf("dependent row lost: %s count=%d err=%v", query, count, err)
			}
		}
	})

	t.Run("collision fails without overwriting either document", func(t *testing.T) {
		source, target := newDoc("collision-source.md"), newDoc("collision-target.md")
		upsert(source)
		upsert(target)
		beforeSource, beforeTarget := read(source.ID), read(target.ID)
		id := source.ID
		source.Path, source.Title = target.Path, "must not overwrite target"
		if err := vs.UpsertDocument(ctx, source); err == nil {
			t.Fatal("expected unique-path collision error")
		}
		if source.ID != id || !reflect.DeepEqual(read(id), beforeSource) ||
			!reflect.DeepEqual(read(target.ID), beforeTarget) {
			t.Fatal("collision changed source identity or persisted document data")
		}
	})

	t.Run("stale snapshot cannot revert newer ownership or content", func(t *testing.T) {
		doc := newDoc("stale-snapshot.md")
		upsert(doc)
		if doc.UpdatedAt.IsZero() {
			t.Fatal("insert must return an optimistic concurrency token")
		}
		stale := read(doc.ID)
		fresh := read(doc.ID)
		fresh.Scope, fresh.AgentID, fresh.TeamID = "team", nil, &team
		fresh.Title = "newer title"
		upsert(fresh)
		if !fresh.UpdatedAt.After(stale.UpdatedAt) {
			t.Fatal("successful update must advance and return its concurrency token")
		}
		before := read(doc.ID)
		stale.Title, stale.Summary = "stale title", "stale content"
		withoutToken := *fresh
		withoutToken.UpdatedAt = time.Time{}
		if err := vs.UpsertDocument(ctx, &withoutToken); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("missing token must fail closed: %v", err)
		}
		oldToken := stale.UpdatedAt
		if err := vs.UpsertDocument(ctx, stale); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("stale update must return recognizable safe error: %v", err)
		}
		if !reflect.DeepEqual(read(doc.ID), before) || !stale.UpdatedAt.Equal(oldToken) {
			t.Fatal("stale update changed ownership, content, or caller token")
		}
		fresh.Title = "follow-up using returned token"
		upsert(fresh)
		if read(doc.ID).Title != fresh.Title {
			t.Fatal("returned token did not allow subsequent update")
		}
	})

	t.Run("orphan move cannot bypass occupied live team key", func(t *testing.T) {
		orphan := newDoc("orphan-collision.md")
		orphan.Scope, orphan.AgentID = "shared", nil
		upsert(orphan)
		exec("UPDATE vault_documents SET orphaned_from_id = ? WHERE id = ?", uuid.NewString(), orphan.ID)
		target := newDoc(orphan.Path)
		target.Scope, target.AgentID, target.TeamID = "team", nil, &team
		upsert(target)
		beforeOrphan, beforeTarget := read(orphan.ID), read(target.ID)
		orphan.Scope, orphan.TeamID = "team", &team
		if err := vs.UpsertDocument(ctx, orphan); err == nil {
			t.Fatal("orphan provenance bypassed occupied live destination")
		}
		if !reflect.DeepEqual(read(orphan.ID), beforeOrphan) || !reflect.DeepEqual(read(target.ID), beforeTarget) {
			t.Fatal("failed orphan move changed source or target")
		}
	})

	t.Run("missing and foreign-tenant IDs fail closed", func(t *testing.T) {
		doc := newDoc("tenant-protected.md")
		upsert(doc)
		before := read(doc.ID)
		for _, tc := range []struct{ name, id, tenant string }{
			{"missing", uuid.NewString(), tenant},
			{"foreign tenant", doc.ID, otherTenant},
		} {
			t.Run(tc.name, func(t *testing.T) {
				attempt := *doc
				attempt.ID, attempt.TenantID, attempt.Title = tc.id, tc.tenant, "must not persist"
				if err := vs.UpsertDocument(ctx, &attempt); !errors.Is(err, sql.ErrNoRows) {
					t.Fatalf("want wrapped sql.ErrNoRows, got %v", err)
				}
				if attempt.ID != tc.id || !reflect.DeepEqual(read(doc.ID), before) {
					t.Fatal("failed update mutated identity or existing document")
				}
				query := "SELECT COUNT(*) FROM vault_documents WHERE tenant_id = ? AND path = ?"
				query = strings.Replace(query, "?", "$1", 1)
				query = strings.Replace(query, "?", "$2", 1)
				var count int
				if err := db.QueryRowContext(ctx, query, tc.tenant, doc.Path).Scan(&count); err != nil {
					t.Fatal(err)
				}
				want := 1
				if tc.tenant != tenant {
					want = 0
				}
				if count != want {
					t.Fatalf("failed update inserted a row: count=%d want=%d", count, want)
				}
			})
		}
	})
}

func vaultIdentityFixture(t *testing.T) (*sql.DB, store.VaultStore, string, string, string, string) {
	t.Helper()
	db := hooksTestDB(t)
	tenant, agent := seedTenantAndAgent(t, db)
	otherTenant, _ := seedTenantAndAgent(t, db)
	team := uuid.NewString()
	if _, err := db.Exec("INSERT INTO agent_teams (id, tenant_id, name, lead_agent_id, created_by) VALUES ($1, $2, $3, $4, 'owner')",
		team, tenant, "vault-"+team, agent); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, query := range []string{
			"DELETE FROM vault_documents WHERE tenant_id = $1",
			"DELETE FROM agent_teams WHERE tenant_id = $1",
		} {
			if _, err := db.Exec(query, tenant); err != nil {
				t.Errorf("cleanup: %v", err)
			}
		}
	})
	return db, NewPGVaultStore(db), tenant.String(), agent.String(), team, otherTenant.String()
}
