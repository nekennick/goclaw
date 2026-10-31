package pg

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

func TestVaultContentUpdate(t *testing.T) {
	_, vs, tenant, agent, team, _ := vaultIdentityFixture(t)
	atomic, ok := vs.(interface {
		UpsertDocumentWithContent(context.Context, *store.VaultDocument, func() (func() error, error)) error
	})
	if !ok {
		t.Fatal("optional atomic content method missing")
	}
	ctx := context.Background()
	read := func(id string) *store.VaultDocument {
		t.Helper()
		doc, err := vs.GetDocumentByID(ctx, tenant, id)
		if err != nil {
			t.Fatal(err)
		}
		return doc
	}
	doc := &store.VaultDocument{TenantID: tenant, AgentID: &agent, Scope: "personal", Path: "atomic-content.md", DocType: "note"}
	if err := vs.UpsertDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}
	stale := read(doc.ID)
	doc.Scope, doc.AgentID, doc.TeamID = "team", nil, &team
	if err := vs.UpsertDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}
	called := false
	if err := atomic.UpsertDocumentWithContent(ctx, stale, func() (func() error, error) {
		called = true
		return nil, nil
	}); !errors.Is(err, sql.ErrNoRows) || called {
		t.Fatalf("stale callback ran or unsafe error: called=%v err=%v", called, err)
	}
	content := "original"
	writeErr := errors.New("partial write failed")
	restoreErr := errors.New("restore failed")
	t.Run("callback failure rolls back database and restores content", func(t *testing.T) {
		attempt := read(doc.ID)
		before := *attempt
		attempt.ContentHash = "must roll back"
		err := atomic.UpsertDocumentWithContent(ctx, attempt, func() (func() error, error) {
			content = "partial"
			return func() error { content = "original"; return restoreErr }, writeErr
		})
		if !errors.Is(err, writeErr) || !errors.Is(err, restoreErr) || content != "original" ||
			!reflect.DeepEqual(read(doc.ID), &before) || !attempt.UpdatedAt.Equal(before.UpdatedAt) {
			t.Fatalf("failure did not restore content/database/token: %v", err)
		}
	})
	t.Run("commit failure restores content", func(t *testing.T) {
		attempt := read(doc.ID)
		before := *attempt
		attempt.ContentHash = "uncommitted"
		cancelCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		err := atomic.UpsertDocumentWithContent(cancelCtx, attempt, func() (func() error, error) {
			content = "uncommitted"
			cancel()
			return func() error { content = "original"; return nil }, nil
		})
		if err == nil || content != "original" || !reflect.DeepEqual(read(doc.ID), &before) ||
			!attempt.UpdatedAt.Equal(before.UpdatedAt) {
			t.Fatalf("commit failure left changed state: %v", err)
		}
	})
	t.Run("success commits and returns reusable token", func(t *testing.T) {
		attempt := read(doc.ID)
		oldToken := attempt.UpdatedAt
		attempt.ContentHash = "committed"
		restored := false
		if err := atomic.UpsertDocumentWithContent(ctx, attempt, func() (func() error, error) {
			content = "committed"
			return func() error { restored = true; return nil }, nil
		}); err != nil {
			t.Fatal(err)
		}
		if restored || content != "committed" || read(doc.ID).ContentHash != "committed" || !attempt.UpdatedAt.After(oldToken) {
			t.Fatal("successful content update did not commit correctly")
		}
		attempt.Title = "follow-up"
		if err := vs.UpsertDocument(ctx, attempt); err != nil {
			t.Fatalf("returned token unusable: %v", err)
		}
	})
	t.Run("collision rejects before callback", func(t *testing.T) {
		target := &store.VaultDocument{TenantID: tenant, Scope: "team", TeamID: &team, Path: "atomic-target.md", DocType: "note"}
		if err := vs.UpsertDocument(ctx, target); err != nil {
			t.Fatal(err)
		}
		attempt := read(doc.ID)
		before := *attempt
		attempt.Path = target.Path
		called := false
		err := atomic.UpsertDocumentWithContent(ctx, attempt, func() (func() error, error) {
			called = true
			return nil, nil
		})
		if err == nil || called || !reflect.DeepEqual(read(doc.ID), &before) {
			t.Fatalf("collision callback ran or row changed: called=%v err=%v", called, err)
		}
	})
}
