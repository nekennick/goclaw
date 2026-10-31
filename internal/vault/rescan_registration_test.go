package vault

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/nextlevelbuilder/goclaw/internal/eventbus"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// Model the existing lookup: empty agent means ANY agent, not shared scope.
type rescanRegistrationStore struct {
	store.VaultStore
	docs       []store.VaultDocument
	writes     int
	lookupErr  error
	listErrAt  int
	listCalls  int
	getCalls   int
	upsertErr  error
	lastUpsert *store.VaultDocument
}

func (s *rescanRegistrationStore) ListDocuments(_ context.Context, tenant, _ string, opts store.VaultListOptions) ([]store.VaultDocument, error) {
	s.listCalls++
	if s.lookupErr != nil {
		return nil, s.lookupErr
	}
	if s.listErrAt > 0 && opts.Offset >= s.listErrAt {
		return nil, errors.New("later page unavailable")
	}
	var docs []store.VaultDocument
	for _, doc := range s.docs {
		if doc.TenantID == tenant && doc.Scope == opts.Scope && (opts.TeamID == nil || *opts.TeamID == "" && doc.TeamID == nil) {
			docs = append(docs, doc)
		}
	}
	if opts.Offset >= len(docs) {
		return nil, nil
	}
	return docs[opts.Offset:min(opts.Offset+opts.Limit, len(docs))], nil
}

func (s *rescanRegistrationStore) GetDocument(ctx context.Context, tenant, agent, path string) (*store.VaultDocument, error) {
	s.getCalls++
	if s.lookupErr != nil {
		return nil, s.lookupErr
	}
	for _, doc := range s.docs {
		if doc.TenantID != tenant || doc.Path != path || (agent != "" && (doc.AgentID == nil || *doc.AgentID != agent)) {
			continue
		}
		if rc := store.RunContextFromCtx(ctx); rc != nil {
			if rc.TeamID == "" && doc.TeamID != nil || rc.TeamID != "" && (doc.TeamID == nil || *doc.TeamID != rc.TeamID) {
				continue
			}
		}
		return &doc, nil
	}
	return nil, sql.ErrNoRows
}

func (s *rescanRegistrationStore) UpsertDocument(_ context.Context, doc *store.VaultDocument) error {
	s.writes++
	preserved := *doc
	s.lastUpsert = &preserved
	if s.upsertErr != nil {
		return s.upsertErr
	}
	for i, old := range s.docs {
		if old.TenantID == doc.TenantID && old.Path == doc.Path && old.Scope == doc.Scope && reflect.DeepEqual(old.AgentID, doc.AgentID) && reflect.DeepEqual(old.TeamID, doc.TeamID) {
			doc.ID = old.ID
			s.docs[i] = *doc
			return nil
		}
	}
	doc.ID = "new-id"
	s.docs = append(s.docs, *doc)
	return nil
}

func TestRescanWorkspaceRegistrationReuse(t *testing.T) {
	const team = "550e8400-e29b-41d4-a716-446655440000"
	for _, tc := range []struct {
		name, path, scope string
		agent, team       *string
		unchanged, reuse  bool
	}{
		{"shared changed", "bot/note.md", "shared", nil, nil, false, true},
		{"shared unchanged", "bot/note.md", "shared", nil, nil, true, true},
		{"root shared changed", "note.md", "shared", nil, nil, false, true},
		{"root shared unchanged", "note.md", "shared", nil, nil, true, true},
		{"other tenant", "bot/note.md", "shared", nil, nil, true, false},
		{"legacy prefix shared", "agents/bot/note.md", "shared", nil, nil, false, true},
		{"personal changed", "bot/note.md", "personal", new("agent"), nil, false, true},
		{"personal unchanged", "bot/note.md", "personal", new("agent"), nil, true, true},
		{"other agent private", "bot/note.md", "personal", new("other"), nil, true, false},
		{"malformed shared owner", "bot/note.md", "shared", new("other"), nil, true, false},
		{"malformed shared team", "bot/note.md", "shared", nil, new(team), true, false},
		{"root private", "note.md", "personal", new("other"), nil, true, false},
		{"team changed", "teams/" + team + "/note.md", "team", nil, new(team), false, true},
		{"team shared registration", "teams/" + team + "/note.md", "shared", nil, nil, false, true},
		{"other team", "teams/" + team + "/note.md", "team", nil, new("other-team"), true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workspace := t.TempDir()
			path := filepath.Join(workspace, filepath.FromSlash(tc.path))
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("new content"), 0600); err != nil {
				t.Fatal(err)
			}
			old := store.VaultDocument{ID: "original", TenantID: "tenant", Path: tc.path, Scope: tc.scope, AgentID: tc.agent, TeamID: tc.team,
				Title: "Curated title", DocType: "document", Summary: "Curated summary", CustomScope: new("custom"), Metadata: map[string]any{"keep": "value"}, ContentHash: "old", UpdatedAt: time.Unix(123, 456)}
			if tc.name == "other tenant" {
				old.TenantID = "other-tenant"
			}
			if tc.unchanged {
				old.ContentHash = ContentHash([]byte("new content"))
			}
			s := &rescanRegistrationStore{docs: []store.VaultDocument{old}}
			result, err := RescanWorkspace(context.Background(), RescanParams{TenantID: "tenant", Workspace: workspace, AgentMap: map[string]string{"bot": "agent"}, TeamSet: map[string]bool{team: true}}, s, eventbus.NewDomainEventBus(eventbus.DefaultConfig()))
			if err != nil {
				t.Fatal(err)
			}
			if result.Errors != 0 {
				t.Fatalf("result: %+v", result)
			}
			if !tc.reuse {
				if result.New != 1 || len(s.docs) != 2 || !reflect.DeepEqual(old, s.docs[0]) {
					t.Fatalf("adopted unrelated registration: %+v, %+v", result, s.docs)
				}
				return
			}
			if len(s.docs) != 1 {
				t.Fatalf("duplicate registration: %+v", s.docs)
			}
			if tc.unchanged {
				if result.Unchanged != 1 || s.writes != 0 || len(result.PendingEvents) != 0 {
					t.Fatalf("unchanged file updated: %+v", result)
				}
				return
			}
			want := old
			want.ContentHash = ContentHash([]byte("new content"))
			if result.Updated != 1 || !reflect.DeepEqual(s.docs[0], want) {
				t.Fatalf("registration not preserved: got %+v, want %+v; result %+v", s.docs[0], want, result)
			}
			if len(result.PendingEvents) != 1 {
				t.Fatalf("events: %+v", result.PendingEvents)
			}
			event := result.PendingEvents[0]
			owner := ""
			if old.AgentID != nil {
				owner = *old.AgentID
			}
			payload := event.Payload.(eventbus.VaultDocUpsertedPayload)
			if event.AgentID != owner || payload.AgentID != owner || payload.DocID != old.ID {
				t.Fatalf("incorrect event identity: %+v", event)
			}
			// A subsequent reload must find precisely the same registration.
			again, err := RescanWorkspace(context.Background(), RescanParams{TenantID: "tenant", Workspace: workspace, AgentMap: map[string]string{"bot": "agent"}, TeamSet: map[string]bool{team: true}}, s, nil)
			if err != nil || again.Unchanged != 1 || s.writes != 1 {
				t.Fatalf("reload: %+v, %v", again, err)
			}
		})
	}
}

func TestRescanWorkspaceLookupFailureDoesNotCreateDuplicate(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "note.md"), []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &rescanRegistrationStore{lookupErr: errors.New("lookup unavailable")}
	result, err := RescanWorkspace(context.Background(), RescanParams{TenantID: "tenant", Workspace: workspace}, s, nil)
	if err != nil || result.Errors != 1 || result.New != 0 || s.writes != 0 {
		t.Fatalf("lookup failure created registration: %+v, %v", result, err)
	}
}

func TestRescanWorkspaceSharedCandidates(t *testing.T) {
	for _, path := range []string{"bot/note.md", "note.md"} {
		for _, mode := range []string{"mixed", "unchanged", "ambiguous", "later page error", "stale"} {
			t.Run(path+"/"+mode, func(t *testing.T) {
				workspace := t.TempDir()
				absPath := filepath.Join(workspace, filepath.FromSlash(path))
				if err := os.MkdirAll(filepath.Dir(absPath), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(absPath, []byte("content"), 0600); err != nil {
					t.Fatal(err)
				}
				private := store.VaultDocument{ID: "private", TenantID: "tenant", AgentID: new("other"), Scope: "personal", Path: path, ContentHash: "private-hash"}
				s := &rescanRegistrationStore{docs: []store.VaultDocument{private}}
				// Match lies beyond the store's default first page.
				for i := 0; i < 101; i++ {
					s.docs = append(s.docs, store.VaultDocument{ID: fmt.Sprint(i), TenantID: "tenant", Scope: "shared", Path: fmt.Sprintf("other-%d.md", i)})
				}
				shared := store.VaultDocument{ID: "shared", TenantID: "tenant", Scope: "shared", Path: path, Title: "Curated", Metadata: map[string]any{"keep": true}, ContentHash: "old", UpdatedAt: time.Unix(123, 456)}
				if mode == "unchanged" {
					shared.ContentHash = ContentHash([]byte("content"))
				}
				s.docs = append(s.docs, shared)
				if mode == "ambiguous" || mode == "later page error" {
					for i := 0; i < 101; i++ {
						s.docs = append(s.docs, store.VaultDocument{ID: fmt.Sprintf("later-%d", i), TenantID: "tenant", Scope: "shared", Path: fmt.Sprintf("later-%d.md", i)})
					}
				}
				if mode == "ambiguous" {
					second := shared
					second.ID = "orphan-shared"
					s.docs = append(s.docs, second)
				}
				if mode == "later page error" {
					s.listErrAt = 200
				}
				if mode == "stale" {
					s.upsertErr = sql.ErrNoRows
				}
				before := append([]store.VaultDocument(nil), s.docs...)
				result, err := RescanWorkspace(context.Background(), RescanParams{TenantID: "tenant", Workspace: workspace, AgentMap: map[string]string{"bot": "agent"}}, s, eventbus.NewDomainEventBus(eventbus.DefaultConfig()))
				if err != nil {
					t.Fatal(err)
				}
				if mode == "stale" {
					if result.Errors != 1 || result.Updated != 0 || len(result.PendingEvents) != 0 || !reflect.DeepEqual(before, s.docs) {
						t.Fatal("stale CAS must not mutate registration or publish")
					}
					if s.lastUpsert == nil || s.lastUpsert.ID != shared.ID || !s.lastUpsert.UpdatedAt.Equal(shared.UpdatedAt) {
						t.Fatal("CAS identity/token lost")
					}
					return
				}
				if mode == "ambiguous" || mode == "later page error" {
					if result.Errors != 1 || s.writes != 0 || len(result.PendingEvents) != 0 || !reflect.DeepEqual(before, s.docs) {
						t.Fatalf("not fail closed: %+v", result)
					}
					return
				}
				if mode == "unchanged" {
					if result.Unchanged != 1 || s.writes != 0 {
						t.Fatalf("unchanged: %+v", result)
					}
					return
				}
				if result.Updated != 1 || result.New != 0 || len(s.docs) != len(before) || !reflect.DeepEqual(s.docs[0], private) {
					t.Fatalf("incorrect registration: %+v", result)
				}
				shared.ContentHash = ContentHash([]byte("content"))
				if !reflect.DeepEqual(s.docs[len(s.docs)-1], shared) {
					t.Fatal("shared metadata/identity lost")
				}
				if len(result.PendingEvents) != 1 || result.PendingEvents[0].AgentID != "" {
					t.Fatal("incorrect effective owner")
				}
			})
		}
	}
}

func TestRescanWorkspaceUsesExactSharedLookupPerFile(t *testing.T) {
	workspace := t.TempDir()
	s := &exactSharedRegistrationStore{}
	calls := 0
	s.find = func(_ context.Context, tenant, path string) (*store.VaultDocument, error) {
		calls++
		for _, doc := range s.docs {
			if doc.TenantID == tenant && doc.Path == path {
				return &doc, nil
			}
		}
		return nil, sql.ErrNoRows
	}
	for _, path := range []string{"a.md", "b.md", "c.md"} {
		if err := os.WriteFile(filepath.Join(workspace, path), []byte("content"), 0600); err != nil {
			t.Fatal(err)
		}
		s.docs = append(s.docs, store.VaultDocument{ID: path, TenantID: "tenant", Scope: "shared", Path: path, ContentHash: "old"})
	}
	result, err := RescanWorkspace(context.Background(), RescanParams{TenantID: "tenant", Workspace: workspace}, s, nil)
	if err != nil || result.Updated != 3 || result.Errors != 0 {
		t.Fatalf("rescan: %+v, %v", result, err)
	}
	if s.listCalls != 0 || calls != 3 {
		t.Fatalf("expected one exact lookup per file, no OFFSET listing: exact=%d, list=%d", calls, s.listCalls)
	}
}

func TestRescanWorkspaceOwnPrivateAndSharedFailClosed(t *testing.T) {
	for _, unchanged := range []bool{false, true} {
		t.Run(fmt.Sprint("unchanged=", unchanged), func(t *testing.T) {
			workspace := t.TempDir()
			if err := os.MkdirAll(filepath.Join(workspace, "bot"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(workspace, "bot", "note.md"), []byte("content"), 0600); err != nil {
				t.Fatal(err)
			}
			private := store.VaultDocument{ID: "private", TenantID: "tenant", Path: "bot/note.md", AgentID: new("agent"), Scope: "personal", ContentHash: "old"}
			if unchanged {
				private.ContentHash = ContentHash([]byte("content"))
			}
			shared := store.VaultDocument{ID: "shared", TenantID: "tenant", Path: private.Path, Scope: "shared", ContentHash: "old-shared"}
			s := &exactSharedRegistrationStore{rescanRegistrationStore: rescanRegistrationStore{docs: []store.VaultDocument{private, shared}}}
			calls := 0
			s.find = func(_ context.Context, _, _ string) (*store.VaultDocument, error) {
				calls++
				if s.getCalls != 0 {
					t.Fatal("shared lookup must precede owner lookup")
				}
				return &shared, nil
			}
			before := append([]store.VaultDocument(nil), s.docs...)
			result, err := RescanWorkspace(context.Background(), RescanParams{TenantID: "tenant", Workspace: workspace, AgentMap: map[string]string{"bot": "agent"}}, s, eventbus.NewDomainEventBus(eventbus.DefaultConfig()))
			if err != nil || result.Errors != 1 || result.Unchanged != 0 || s.writes != 0 || len(result.PendingEvents) != 0 || !reflect.DeepEqual(before, s.docs) {
				t.Fatalf("coexisting own/shared rows must fail closed: %+v, %v", result, err)
			}
			if calls != 1 || s.listCalls != 0 {
				t.Fatal("must reuse a single exact shared lookup")
			}
		})
	}
}

func TestRescanWorkspaceExactSharedAbsenceNotRetried(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "bot"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "bot", "note.md"), []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	s := &exactSharedRegistrationStore{find: func(context.Context, string, string) (*store.VaultDocument, error) {
		calls++
		return nil, sql.ErrNoRows
	}}
	result, err := RescanWorkspace(context.Background(), RescanParams{TenantID: "tenant", Workspace: workspace, AgentMap: map[string]string{"bot": "agent"}}, s, nil)
	if err != nil || result.New != 1 || result.Errors != 0 || calls != 1 || s.getCalls != 1 || s.listCalls != 0 {
		t.Fatalf("exact absence must be reused: %+v, %v; exact=%d, owner=%d, list=%d", result, err, calls, s.getCalls, s.listCalls)
	}
}

type exactSharedRegistrationStore struct {
	rescanRegistrationStore
	find func(context.Context, string, string) (*store.VaultDocument, error)
}

func (s *exactSharedRegistrationStore) FindSharedDocumentByPath(ctx context.Context, tenantID, path string) (*store.VaultDocument, error) {
	return s.find(ctx, tenantID, path)
}

func TestFindSharedRegistrationOptionalLookup(t *testing.T) {
	want := &store.VaultDocument{
		ID: "shared", TenantID: "tenant", Path: "bot/note.md", Scope: "shared",
		Title: "Curated", Metadata: map[string]any{"keep": true}, UpdatedAt: time.Unix(123, 456),
	}
	ambiguous := errors.New("ambiguous shared registrations")
	for _, tc := range []struct {
		name string
		doc  *store.VaultDocument
		err  error
	}{
		{"found", want, nil},
		{"missing", nil, nil},
		{"no rows", nil, sql.ErrNoRows},
		{"ambiguous", nil, ambiguous},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			calls := 0
			s := &exactSharedRegistrationStore{find: func(gotCtx context.Context, tenantID, path string) (*store.VaultDocument, error) {
				calls++
				if gotCtx != ctx || tenantID != want.TenantID || path != want.Path {
					t.Fatal("exact lookup arguments changed")
				}
				return tc.doc, tc.err
			}}
			got, err := FindSharedRegistration(ctx, s, want.TenantID, want.Path)
			if calls != 1 || s.listCalls != 0 {
				t.Fatal("optional lookup must not fall back to listing, including on errors")
			}
			if errors.Is(tc.err, sql.ErrNoRows) {
				if got != nil || err != nil {
					t.Fatalf("missing registration: %v, %v", got, err)
				}
				return
			}
			if !errors.Is(err, tc.err) || !reflect.DeepEqual(got, tc.doc) {
				t.Fatalf("got %+v, %v; want %+v, %v", got, err, tc.doc, tc.err)
			}
		})
	}
}
