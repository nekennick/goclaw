//go:build integration

package integration

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// TestStoreVault_ScopeTransitionIdentity uses the real PG constraints and
// ownership triggers. Default nil Metadata is intentional: existing callers
// need not initialize it before updating a document.
func TestStoreVault_ScopeTransitionIdentity(t *testing.T) {
	db := testDB(t)
	tenant, agent := seedTenantAgent(t, db)
	otherTenant, _ := seedTenantAgent(t, db)
	ctx := tenantCtx(tenant)
	vs := newVaultStore(db)
	tid, aid, team := tenant.String(), agent.String(), uuid.NewString()
	if _, err := db.Exec(`INSERT INTO agent_teams (id, tenant_id, name, lead_agent_id, created_by)
		VALUES ($1, $2, $3, $4, 'test-owner')`, team, tenant, "vault-"+team, agent); err != nil {
		t.Fatal(err)
	}
	upsert := func(doc *store.VaultDocument) {
		t.Helper()
		if err := vs.UpsertDocument(ctx, doc); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}
	read := func(id string) *store.VaultDocument {
		t.Helper()
		doc, err := vs.GetDocumentByID(ctx, tid, id)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		return doc
	}
	doc := makeVaultDoc(tid, aid, "transitions/source.md", "Source")
	peer := makeVaultDoc(tid, aid, "transitions/peer.md", "Peer")
	upsert(doc)
	upsert(peer)
	id := doc.ID
	before := read(id)
	if err := vs.ReplaceDocumentChunks(ctx, tid, id, doc.ContentHash,
		[]store.VaultChunk{{Index: 0, StartLine: 1, EndLine: 1, Text: "retained body"}}); err != nil {
		t.Fatal(err)
	}
	for _, edge := range [][2]string{{id, peer.ID}, {peer.ID, id}} {
		if _, err := db.Exec("INSERT INTO vault_links (id, from_doc_id, to_doc_id) VALUES ($1, $2, $3)",
			uuid.NewString(), edge[0], edge[1]); err != nil {
			t.Fatal(err)
		}
	}
	assertReferences := func() {
		t.Helper()
		var chunks, links int
		if err := db.QueryRow("SELECT COUNT(*) FROM vault_document_chunks WHERE document_id = $1 AND text = 'retained body'", id).Scan(&chunks); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow("SELECT COUNT(*) FROM vault_links WHERE from_doc_id = $1 OR to_doc_id = $1", id).Scan(&links); err != nil {
			t.Fatal(err)
		}
		if chunks != 1 || links != 2 || doc.ID != id {
			t.Fatalf("identity/references changed: ID=%s chunks=%d links=%d", doc.ID, chunks, links)
		}
	}

	// The handler supplies normalized scope/ownership; the store must persist it
	// rather than letting owner-deletion triggers reinterpret an explicit move.
	chat := "chat-before-move"
	doc.Scope, doc.AgentID, doc.TeamID, doc.ChatID = "team", nil, &team, &chat
	doc.Path = "transitions/Renamed.MD"
	upsert(doc)
	got := read(id)
	if got.Scope != "team" || got.AgentID != nil || got.TeamID == nil || *got.TeamID != team ||
		got.Path != doc.Path || got.PathBasename != "renamed.md" || !got.CreatedAt.Equal(before.CreatedAt) {
		t.Fatalf("team move not persisted: %+v", got)
	}
	assertReferences()
	doc.Scope, doc.TeamID, doc.ChatID = "shared", nil, nil
	upsert(doc)
	got = read(id)
	if got.Scope != "shared" || got.AgentID != nil || got.TeamID != nil || got.ChatID != nil {
		t.Fatalf("shared move not persisted: %+v", got)
	}
	var provenance sql.NullString
	if err := db.QueryRow("SELECT orphaned_from_id FROM vault_documents WHERE id = $1", id).Scan(&provenance); err != nil {
		t.Fatal(err)
	}
	if provenance.Valid {
		t.Fatal("explicit move was incorrectly marked as an owner-deletion orphan")
	}
	assertReferences()

	// Moving a personal document onto an existing shared key must fail even
	// though the owner-deletion trigger temporarily assigns orphan provenance.
	target := makeSharedVaultDoc(tid, "transitions/collision.md", "Do not overwrite")
	upsert(target)
	doc.Scope, doc.AgentID = "personal", &aid
	upsert(doc)
	beforeSource, beforeTarget := read(id), read(target.ID)
	doc.Scope, doc.AgentID, doc.Path, doc.Title = "shared", nil, target.Path, "Attempted overwrite"
	if err := vs.UpsertDocument(ctx, doc); err == nil {
		t.Fatal("expected collision error")
	}
	if !reflect.DeepEqual(read(id), beforeSource) || !reflect.DeepEqual(read(target.ID), beforeTarget) {
		t.Fatal("collision modified a source or target row")
	}
	assertReferences()

	t.Run("orphan cannot bypass occupied team destination", func(t *testing.T) {
		orphan := makeSharedVaultDoc(tid, "transitions/orphan.md", "Orphan")
		upsert(orphan)
		if _, err := db.Exec("UPDATE vault_documents SET orphaned_from_id = $1 WHERE id = $2", uuid.NewString(), orphan.ID); err != nil {
			t.Fatal(err)
		}
		occupied := makeTeamVaultDoc(tid, team, orphan.Path, "Occupied")
		upsert(occupied)
		beforeOrphan, beforeOccupied := read(orphan.ID), read(occupied.ID)
		orphan.Scope, orphan.TeamID = "team", &team
		if err := vs.UpsertDocument(ctx, orphan); err == nil {
			t.Fatal("orphan provenance bypassed live team collision")
		}
		if !reflect.DeepEqual(read(orphan.ID), beforeOrphan) || !reflect.DeepEqual(read(occupied.ID), beforeOccupied) {
			t.Fatal("orphan collision changed persisted rows")
		}
		orphan.Path = "transitions/unoccupied-team.md"
		upsert(orphan)
		var provenance sql.NullString
		if err := db.QueryRow("SELECT orphaned_from_id FROM vault_documents WHERE id = $1", orphan.ID).Scan(&provenance); err != nil {
			t.Fatal(err)
		}
		if provenance.Valid || read(orphan.ID).Scope != "team" {
			t.Fatal("explicit owned move retained orphan provenance")
		}
		shared := makeSharedVaultDoc(tid, orphan.Path, "Occupied shared destination")
		upsert(shared)
		beforeOwned, beforeShared := read(orphan.ID), read(shared.ID)
		orphan.Scope, orphan.TeamID = "shared", nil
		if err := vs.UpsertDocument(ctx, orphan); err == nil {
			t.Fatal("owner-deletion trigger provenance bypassed live shared collision")
		}
		if !reflect.DeepEqual(read(orphan.ID), beforeOwned) || !reflect.DeepEqual(read(shared.ID), beforeShared) {
			t.Fatal("failed owned-to-shared move changed persisted rows")
		}
	})

	t.Run("stale read cannot undo ownership move", func(t *testing.T) {
		stale, fresh := read(id), read(id)
		fresh.Scope, fresh.AgentID, fresh.TeamID = "team", nil, &team
		upsert(fresh)
		before := read(id)
		stale.Title = "stale content must not persist"
		if err := vs.UpsertDocument(ctx, stale); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("stale update: want safe error, got %v", err)
		}
		if !reflect.DeepEqual(read(id), before) {
			t.Fatal("stale update reverted ownership or content")
		}
		assertReferences()
		fresh.Title = "follow-up"
		upsert(fresh)
		assertReferences()
	})

	beforeSource = read(id)
	for _, tc := range []struct{ name, id, tenant string }{
		{"foreign tenant", id, otherTenant.String()},
		{"missing ID", uuid.NewString(), tid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attempt := *beforeSource
			attempt.ID, attempt.TenantID = tc.id, tc.tenant
			if err := vs.UpsertDocument(ctx, &attempt); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("want wrapped sql.ErrNoRows, got %v", err)
			}
			if attempt.ID != tc.id || !reflect.DeepEqual(read(id), beforeSource) {
				t.Fatal("rejected update changed identity or document")
			}
		})
	}
	assertReferences()
}
