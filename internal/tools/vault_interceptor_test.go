package tools

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

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/eventbus"
	"github.com/nextlevelbuilder/goclaw/internal/store"
	"github.com/nextlevelbuilder/goclaw/internal/vault"
)

type interceptorRegistrationStore struct {
	store.VaultStore
	docs                []store.VaultDocument
	writes, hashUpdates int
	lookupErr           error
	listErrAt           int
	listCalls           int
	getCalls            int
	upsertErr           error
	lastUpsert          *store.VaultDocument
}

func (s *interceptorRegistrationStore) ListDocuments(_ context.Context, tenant, _ string, opts store.VaultListOptions) ([]store.VaultDocument, error) {
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

func (s *interceptorRegistrationStore) GetDocument(ctx context.Context, tenant, agent, path string) (*store.VaultDocument, error) {
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

func (s *interceptorRegistrationStore) UpsertDocument(_ context.Context, doc *store.VaultDocument) error {
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

func (s *interceptorRegistrationStore) UpdateHash(_ context.Context, tenant, id, hash string) error {
	s.hashUpdates++
	for i := range s.docs {
		if s.docs[i].TenantID == tenant && s.docs[i].ID == id {
			s.docs[i].ContentHash = hash
			return nil
		}
	}
	return sql.ErrNoRows
}

func TestVaultInterceptorRegistrationReuse(t *testing.T) {
	tenant, agent, team := uuid.New(), uuid.New(), uuid.New().String()
	for _, mode := range []string{"text", "media", "read"} {
		for _, tc := range []struct {
			name, scope, tenant, contextTeam string
			agent, team, chat                *string
			reuse                            bool
		}{
			{"shared", "shared", tenant.String(), "", nil, nil, nil, true},
			{"personal", "personal", tenant.String(), "", new(agent.String()), nil, nil, true},
			{"team", "team", tenant.String(), team, nil, new(team), new("chat"), true},
			{"shared in team context", "shared", tenant.String(), team, nil, nil, nil, true},
			{"other agent", "personal", tenant.String(), "", new(uuid.New().String()), nil, nil, false},
			{"other tenant", "shared", uuid.New().String(), "", nil, nil, nil, false},
			{"other team", "team", tenant.String(), team, nil, new(uuid.New().String()), nil, false},
			{"other chat", "team", tenant.String(), team, nil, new(team), new("other-chat"), false},
			{"shared with owner", "shared", tenant.String(), "", new(uuid.New().String()), nil, nil, false},
			{"shared with team", "shared", tenant.String(), "", nil, new(team), nil, false},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				workspace := t.TempDir()
				path := filepath.Join(workspace, "note.md")
				if err := os.WriteFile(path, []byte("new content"), 0600); err != nil {
					t.Fatal(err)
				}
				ctx := store.WithTenantID(context.Background(), tenant)
				ctx = store.WithAgentID(ctx, agent)
				ctx = store.WithRunContext(ctx, &store.RunContext{TeamID: tc.contextTeam, TeamIsolated: tc.contextTeam != ""})
				ctx = WithWorkspaceChatID(ctx, "chat")
				ctx = WithDelegationID(ctx, "new-delegation")
				old := store.VaultDocument{ID: "original", TenantID: tc.tenant, Path: "note.md", AgentID: tc.agent, TeamID: tc.team, ChatID: tc.chat, Scope: tc.scope,
					Title: "Curated title", DocType: "document", Summary: "Curated summary", CustomScope: new("custom"), Metadata: map[string]any{"keep": "value", "delegation_id": "original-delegation", "created_in": "original-origin"}, ContentHash: "old", UpdatedAt: time.Unix(123, 456)}
				s := &interceptorRegistrationStore{docs: []store.VaultDocument{old}}
				bus := &recordingDelegateEventBus{}
				interceptor := NewVaultInterceptor(s, workspace, bus)
				switch mode {
				case "text":
					interceptor.AfterWrite(ctx, path, "new content")
				case "media":
					interceptor.AfterWriteMedia(ctx, path, "New caption", "image/png")
				case "read":
					interceptor.BeforeRead(ctx, path)
				}
				if !reflect.DeepEqual(old.Metadata, map[string]any{"keep": "value", "delegation_id": "original-delegation", "created_in": "original-origin"}) {
					t.Fatal("mutated lookup result metadata in place")
				}
				if !tc.reuse {
					if !reflect.DeepEqual(old, s.docs[0]) || s.hashUpdates != 0 {
						t.Fatalf("mutated unrelated registration: %+v", s.docs)
					}
					if tc.name == "other chat" {
						if s.writes != 0 || len(bus.snapshot()) != 0 {
							t.Fatal("wrote across chat boundary")
						}
						return
					}
					if mode != "read" && (s.writes != 1 || len(s.docs) != 2) {
						t.Fatalf("missing new registration: %+v", s.docs)
					}
					return
				}
				if len(s.docs) != 1 {
					t.Fatalf("duplicate registration: %+v", s.docs)
				}
				want := old
				want.ContentHash = vault.ContentHash([]byte("new content"))
				if mode == "media" {
					want.Summary = "New caption"
					want.Metadata = map[string]any{"keep": "value", "mime_type": "image/png", "delegation_id": "original-delegation", "created_in": "original-origin"}
				}
				if !reflect.DeepEqual(s.docs[0], want) {
					t.Fatalf("got %+v, want %+v", s.docs[0], want)
				}
				if mode == "read" {
					return
				}
				events := bus.snapshot()
				if len(events) != 1 {
					t.Fatalf("events: %+v", events)
				}
				owner := ""
				if old.AgentID != nil {
					owner = *old.AgentID
				}
				payload := events[0].Payload.(eventbus.VaultDocUpsertedPayload)
				if events[0].AgentID != owner || payload.AgentID != owner || payload.DocID != old.ID {
					t.Fatalf("incorrect event identity: %+v", events[0])
				}
			})
		}
	}
}

func TestVaultInterceptorLookupFailureDoesNotCreateDuplicate(t *testing.T) {
	workspace := t.TempDir()
	path := filepath.Join(workspace, "note.md")
	if err := os.WriteFile(path, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := store.WithTenantID(context.Background(), uuid.New())
	ctx = store.WithAgentID(ctx, uuid.New())
	s := &interceptorRegistrationStore{lookupErr: errors.New("lookup unavailable")}
	bus := &recordingDelegateEventBus{}
	interceptor := NewVaultInterceptor(s, workspace, bus)
	interceptor.AfterWrite(ctx, path, "content")
	interceptor.AfterWriteMedia(ctx, path, "caption", "image/png")
	interceptor.BeforeRead(ctx, path)
	if s.writes != 0 || s.hashUpdates != 0 || len(bus.snapshot()) != 0 {
		t.Fatal("lookup error must not create registrations or events")
	}
}

func TestVaultInterceptorSharedCandidates(t *testing.T) {
	for _, operation := range []string{"text", "media", "read"} {
		for _, mode := range []string{"mixed", "ambiguous", "later page error", "stale"} {
			if operation == "read" && mode == "stale" {
				continue
			}
			t.Run(operation+"/"+mode, func(t *testing.T) {
				tenant, agent := uuid.New(), uuid.New()
				ctx := store.WithAgentID(store.WithTenantID(context.Background(), tenant), agent)
				workspace := t.TempDir()
				path := filepath.Join(workspace, "note.md")
				if err := os.WriteFile(path, []byte("content"), 0600); err != nil {
					t.Fatal(err)
				}
				private := store.VaultDocument{ID: "private", TenantID: tenant.String(), Scope: "personal", AgentID: new(uuid.New().String()), Path: "note.md", ContentHash: "private-hash"}
				s := &interceptorRegistrationStore{docs: []store.VaultDocument{private}}
				for i := 0; i < 101; i++ {
					s.docs = append(s.docs, store.VaultDocument{ID: fmt.Sprint(i), TenantID: tenant.String(), Scope: "shared", Path: fmt.Sprintf("other-%d.md", i)})
				}
				shared := store.VaultDocument{ID: "shared", TenantID: tenant.String(), Scope: "shared", Path: "note.md", Title: "Curated", Metadata: map[string]any{"keep": true}, ContentHash: "old", UpdatedAt: time.Unix(123, 456)}
				s.docs = append(s.docs, shared)
				if mode == "ambiguous" || mode == "later page error" {
					for i := 0; i < 101; i++ {
						s.docs = append(s.docs, store.VaultDocument{ID: fmt.Sprintf("later-%d", i), TenantID: tenant.String(), Scope: "shared", Path: fmt.Sprintf("later-%d.md", i)})
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
				bus := &recordingDelegateEventBus{}
				v := NewVaultInterceptor(s, workspace, bus)
				switch operation {
				case "text":
					v.AfterWrite(ctx, path, "content")
				case "media":
					v.AfterWriteMedia(ctx, path, "caption", "image/png")
				case "read":
					v.BeforeRead(ctx, path)
				}
				if mode == "stale" {
					if s.writes != 1 || len(bus.snapshot()) != 0 || !reflect.DeepEqual(before, s.docs) {
						t.Fatal("stale CAS must not mutate registration or publish")
					}
					if s.lastUpsert == nil || s.lastUpsert.ID != shared.ID || !s.lastUpsert.UpdatedAt.Equal(shared.UpdatedAt) {
						t.Fatal("CAS identity/token lost")
					}
					return
				}
				if mode != "mixed" {
					if s.writes != 0 || s.hashUpdates != 0 || len(bus.snapshot()) != 0 || !reflect.DeepEqual(before, s.docs) {
						t.Fatal("ambiguous/incomplete lookup must fail closed")
					}
					return
				}
				if len(s.docs) != len(before) || !reflect.DeepEqual(s.docs[0], private) {
					t.Fatal("private adopted or duplicate created")
				}
				shared.ContentHash = vault.ContentHash([]byte("content"))
				if operation == "media" {
					shared.Summary = "caption"
					shared.Metadata = map[string]any{"keep": true, "mime_type": "image/png"}
				}
				if !reflect.DeepEqual(s.docs[len(s.docs)-1], shared) {
					t.Fatal("shared registration identity/metadata lost")
				}
				if operation == "read" {
					return
				}
				events := bus.snapshot()
				if len(events) != 1 || events[0].AgentID != "" || events[0].Payload.(eventbus.VaultDocUpsertedPayload).DocID != shared.ID {
					t.Fatal("incorrect shared event identity")
				}
			})
		}
	}
}

type exactInterceptorRegistrationStore struct {
	interceptorRegistrationStore
	find func(context.Context, string, string) (*store.VaultDocument, error)
}

func (s *exactInterceptorRegistrationStore) FindSharedDocumentByPath(ctx context.Context, tenant, path string) (*store.VaultDocument, error) {
	return s.find(ctx, tenant, path)
}

func TestVaultInterceptorOwnPrivateAndSharedFailClosed(t *testing.T) {
	for _, operation := range []string{"text", "media", "read"} {
		t.Run(operation, func(t *testing.T) {
			tenant, agent := uuid.New(), uuid.New()
			ctx := store.WithAgentID(store.WithTenantID(context.Background(), tenant), agent)
			workspace := t.TempDir()
			path := filepath.Join(workspace, "note.md")
			if err := os.WriteFile(path, []byte("content"), 0600); err != nil {
				t.Fatal(err)
			}
			private := store.VaultDocument{ID: "private", TenantID: tenant.String(), Path: "note.md", AgentID: new(agent.String()), Scope: "personal", ContentHash: "old"}
			shared := store.VaultDocument{ID: "shared", TenantID: tenant.String(), Path: private.Path, Scope: "shared", ContentHash: "old-shared"}
			s := &exactInterceptorRegistrationStore{interceptorRegistrationStore: interceptorRegistrationStore{docs: []store.VaultDocument{private, shared}}}
			calls := 0
			s.find = func(_ context.Context, gotTenant, gotPath string) (*store.VaultDocument, error) {
				calls++
				if s.getCalls != 0 {
					t.Fatal("shared lookup must precede owner lookup")
				}
				if gotTenant != tenant.String() || gotPath != private.Path {
					t.Fatal("incorrect exact lookup arguments")
				}
				return &shared, nil
			}
			before := append([]store.VaultDocument(nil), s.docs...)
			bus := &recordingDelegateEventBus{}
			v := NewVaultInterceptor(s, workspace, bus)
			switch operation {
			case "text":
				v.AfterWrite(ctx, path, "content")
			case "media":
				v.AfterWriteMedia(ctx, path, "caption", "image/png")
			case "read":
				v.BeforeRead(ctx, path)
			}
			if calls != 1 || s.listCalls != 0 || s.writes != 0 || s.hashUpdates != 0 || len(bus.snapshot()) != 0 || !reflect.DeepEqual(before, s.docs) {
				t.Fatal("coexisting own/shared registrations must fail closed")
			}
		})
	}
}

func TestVaultInterceptorExactSharedAbsenceNotRetried(t *testing.T) {
	workspace := t.TempDir()
	ctx := store.WithAgentID(store.WithTenantID(context.Background(), uuid.New()), uuid.New())
	calls := 0
	s := &exactInterceptorRegistrationStore{find: func(context.Context, string, string) (*store.VaultDocument, error) {
		calls++
		return nil, sql.ErrNoRows
	}}
	NewVaultInterceptor(s, workspace, nil).AfterWrite(ctx, filepath.Join(workspace, "note.md"), "content")
	if calls != 1 || s.getCalls != 1 || s.listCalls != 0 || s.writes != 1 {
		t.Fatalf("exact absence must be reused: exact=%d, owner=%d, list=%d, writes=%d", calls, s.getCalls, s.listCalls, s.writes)
	}
}
