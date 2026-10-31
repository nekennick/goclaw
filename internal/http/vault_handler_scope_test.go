package http

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/i18n"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

type vaultScopePGError struct{}

func (vaultScopePGError) Error() string    { return "private database constraint detail" }
func (vaultScopePGError) SQLState() string { return "23505" }

type vaultScopeSQLiteError struct{}

func (vaultScopeSQLiteError) Error() string { return "private database constraint detail" }
func (vaultScopeSQLiteError) Code() int     { return 2067 } // SQLITE_CONSTRAINT_UNIQUE

// Real HTTP payloads: the Web editor sends scope even for title-only edits;
// scope moves must submit the existing UUID to the separate ID-update store path.
func TestHandleUpdateDocument_ScopeTransitions(t *testing.T) {
	agentID, teamID, chatID := uuid.NewString(), uuid.NewString(), "chat-1"
	for _, tc := range []struct {
		name, role, initial, payload, scope string
		team, allowed                       bool
		status                              int
	}{
		{"personal to shared", "owner", "personal", `{"scope":"shared"}`, "shared", false, false, 200},
		{"team to shared", "owner", "team", `{"scope":"shared"}`, "shared", true, false, 200},
		{"personal to team", "owner", "personal", fmt.Sprintf(`{"scope":"team","team_id":%q}`, teamID), "team", false, false, 200},
		{"implicit team move", "owner", "personal", fmt.Sprintf(`{"team_id":%q}`, teamID), "team", false, false, 200},
		{"shared to personal lacks owner target", "owner", "shared", `{"scope":"personal"}`, "", false, false, 400},
		{"clear team lacks owner target", "owner", "team", `{"team_id":""}`, "", true, false, 400},
		{"invalid scope", "owner", "personal", `{"scope":"bogus"}`, "", false, false, 400},
		{"empty scope", "owner", "personal", `{"scope":""}`, "", false, false, 400},
		{"invalid team UUID owner", "owner", "personal", `{"team_id":"bogus"}`, "", false, false, 400},
		{"zero team UUID owner", "owner", "personal", `{"team_id":"00000000-0000-0000-0000-000000000000"}`, "", false, false, 400},
		{"team requires target", "owner", "personal", `{"scope":"team"}`, "", false, false, 400},
		{"conflicting scope team", "owner", "personal", fmt.Sprintf(`{"scope":"shared","team_id":%q}`, teamID), "", false, false, 400},
		{"member cannot share", "operator", "personal", `{"scope":"shared"}`, "", false, false, 403},
		{"admin not owner", "admin", "personal", `{"scope":"shared"}`, "", false, false, 403},
		{"member cannot change team", "operator", "personal", fmt.Sprintf(`{"team_id":%q}`, teamID), "", false, true, 403},
		{"source team membership", "operator", "team", `{"scope":"shared"}`, "", true, false, 403},
		{"web unchanged scope", "operator", "personal", `{"title":"New","doc_type":"note","scope":"personal"}`, "personal", false, false, 200},
		{"owner unchanged personal", "owner", "personal", `{"title":"New","scope":"personal"}`, "personal", false, false, 200},
		{"title-only personal", "operator", "personal", `{"title":"New"}`, "personal", false, false, 200},
		{"web unchanged shared", "operator", "shared", `{"title":"New","scope":"shared"}`, "shared", false, false, 200},
		{"mixed scope content", "owner", "personal", `{"scope":"shared","content":"replaced"}`, "", false, false, 400},
		{"mixed unchanged scope content", "owner", "personal", `{"scope":"personal","content":""}`, "", false, false, 400},
		{"mixed team empty content", "owner", "personal", fmt.Sprintf(`{"team_id":%q,"content":""}`, teamID), "", false, false, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newFakeVaultStore()
			docID := uuid.NewString()
			doc := &store.VaultDocument{ID: docID, TenantID: masterTenant.String(), Scope: tc.initial, Path: "note.md", Title: "Old", DocType: "note", ContentHash: "old-hash"}
			if tc.initial == "personal" {
				doc.AgentID = &agentID
				// Legacy residual chat ownership must not survive sharing.
				doc.ChatID = &chatID
			}
			if tc.team {
				doc.TeamID, doc.ChatID = &teamID, &chatID
			}
			st.docs[docID] = doc
			ws := t.TempDir()
			bus := &fakeEventBus{store: st}
			h := &VaultHandler{store: st, workspace: ws, eventBus: bus, teamAccess: &fakeTeamAccessStore{allowed: tc.allowed}}
			req := httptest.NewRequest(http.MethodPut, "/v1/vault/documents/"+docID, strings.NewReader(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(store.WithUserID(store.WithRole(masterCtx(), tc.role), "user"))
			req.SetPathValue("docID", docID)
			rr := httptest.NewRecorder()
			h.handleUpdateDocument(rr, req)
			if rr.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", rr.Code, tc.status, rr.Body.String())
			}
			if tc.status != 200 {
				if st.upsertCall != 0 || st.docs[docID].Scope != tc.initial {
					t.Fatal("rejected request mutated storage")
				}
			} else {
				got := st.docs[docID]
				var response store.VaultDocument
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("decode document response: %v", err)
				}
				if response.ID != docID || response.Scope != tc.scope {
					t.Fatalf("unexpected response identity/scope: %+v", response)
				}
				if got.ID != docID || got.Scope != tc.scope || len(st.docs) != 1 {
					t.Fatalf("identity/scope changed unexpectedly: %+v", got)
				}
				if tc.initial == "personal" && tc.scope == "personal" {
					if got.AgentID == nil || *got.AgentID != agentID || got.ChatID == nil || *got.ChatID != chatID {
						t.Fatalf("unchanged personal scope lost owner/chat isolation: %+v", got)
					}
					if response.ChatID == nil || *response.ChatID != chatID {
						t.Fatalf("unchanged personal response lost chat isolation: %+v", response)
					}
				}
				if tc.scope == "shared" && (got.AgentID != nil || got.TeamID != nil || got.ChatID != nil) {
					t.Fatalf("shared ownership not cleared: %+v", got)
				}
				if tc.scope == "team" && (got.AgentID != nil || got.TeamID == nil || *got.TeamID != teamID || got.ChatID != nil) {
					t.Fatalf("team ownership invalid: %+v", got)
				}
			}
			if _, err := os.Stat(filepath.Join(ws, "note.md")); !os.IsNotExist(err) {
				t.Fatalf("scope update touched disk: %v", err)
			}
			if len(bus.published) != 0 {
				t.Fatal("metadata-only or rejected scope update published enrichment")
			}
		})
	}
}

func TestHandleUpdateDocument_MixedScopeContentPreservesExistingFile(t *testing.T) {
	st := newFakeVaultStore()
	agentID, docID := uuid.NewString(), uuid.NewString()
	st.docs[docID] = &store.VaultDocument{ID: docID, TenantID: masterTenant.String(), AgentID: &agentID, Scope: "personal", Path: "note.md"}
	h := &VaultHandler{store: st, workspace: t.TempDir()}
	if _, err := h.writeDocumentContent(h.workspace, "note.md", []byte("original")); err != nil {
		t.Fatal(err)
	}
	req := newJSONRequest(t, http.MethodPut, "/", map[string]string{"scope": "shared", "content": "replacement"}).WithContext(store.WithRole(masterCtx(), "owner"))
	req.SetPathValue("docID", docID)
	rr := httptest.NewRecorder()
	h.handleUpdateDocument(rr, req)
	if rr.Code != http.StatusBadRequest || st.upsertCall != 0 {
		t.Fatalf("mixed update was not rejected: %d %s", rr.Code, rr.Body.String())
	}
	got, err := os.ReadFile(filepath.Join(h.workspace, "note.md"))
	if err != nil || string(got) != "original" {
		t.Fatalf("rejected scope update changed file: %q, %v", got, err)
	}
}

func TestHandleUpdateDocument_UnchangedTeamScopeChecksMembershipAndPreservesChat(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		t.Run(fmt.Sprint(allowed), func(t *testing.T) {
			st := newFakeVaultStore()
			teamID, docID, chatID := uuid.NewString(), uuid.NewString(), "chat-1"
			st.docs[docID] = &store.VaultDocument{ID: docID, TenantID: masterTenant.String(), TeamID: &teamID, ChatID: &chatID, Scope: "team", Path: "note.md"}
			h := &VaultHandler{store: st, teamAccess: &fakeTeamAccessStore{allowed: allowed}}
			req := newJSONRequest(t, http.MethodPut, "/", map[string]string{"scope": "team", "title": "New"}).WithContext(store.WithUserID(store.WithRole(masterCtx(), "operator"), "user"))
			req.SetPathValue("docID", docID)
			rr := httptest.NewRecorder()
			h.handleUpdateDocument(rr, req)
			want := http.StatusForbidden
			if allowed {
				want = http.StatusOK
			}
			if rr.Code != want {
				t.Fatalf("status = %d, want %d: %s", rr.Code, want, rr.Body.String())
			}
			if !allowed && st.upsertCall != 0 {
				t.Fatal("non-member update persisted")
			}
			if st.docs[docID].ChatID == nil || *st.docs[docID].ChatID != chatID {
				t.Fatal("unchanged team scope lost chat isolation")
			}
		})
	}
}

func TestHandleUpdateDocument_ScopeStoreErrorsAreSanitized(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{"pg collision", fmt.Errorf("upsert: %w", vaultScopePGError{}), 409},
		{"sqlite collision", fmt.Errorf("upsert: %w", vaultScopeSQLiteError{}), 409},
		{"other failure", fmt.Errorf("private database connection detail"), 500},
		{"stale update", fmt.Errorf("private database detail: %w", sql.ErrNoRows), 409},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newFakeVaultStore()
			agentID, docID := uuid.NewString(), uuid.NewString()
			st.docs[docID] = &store.VaultDocument{ID: docID, TenantID: masterTenant.String(), AgentID: &agentID, Scope: "personal", Path: "note.md"}
			st.upsertErr = tc.err
			h := &VaultHandler{store: st}
			req := newJSONRequest(t, http.MethodPut, "/", map[string]string{"scope": "shared"}).WithContext(store.WithRole(masterCtx(), "owner"))
			req.SetPathValue("docID", docID)
			rr := httptest.NewRecorder()
			h.handleUpdateDocument(rr, req)
			if rr.Code != tc.status || strings.Contains(rr.Body.String(), "private database") {
				t.Fatalf("unsafe error response: %d %s", rr.Code, rr.Body.String())
			}
			if st.docs[docID].Scope != "personal" {
				t.Fatal("failed upsert mutated row")
			}
		})
	}
}

func TestHandleUpdateDocument_ScopeErrorsLocalized(t *testing.T) {
	for _, locale := range []string{i18n.LocaleEN, i18n.LocaleVI, i18n.LocaleZH} {
		for _, tc := range []struct {
			name, role, scope, payload, key string
			args                            []any
			status                          int
			storeErr                        error
		}{
			{"mixed content", "owner", "personal", `{"scope":"shared","content":""}`, i18n.MsgVaultContentScopeSeparate, nil, 400, nil},
			{"invalid scope", "owner", "personal", `{"scope":"invalid"}`, i18n.MsgVaultInvalidScope, nil, 400, nil},
			{"scope team mismatch", "owner", "personal", `{"scope":"shared","team_id":"nonempty"}`, i18n.MsgVaultTeamScopeRequired, nil, 400, nil},
			{"owner required", "operator", "personal", `{"scope":"shared"}`, i18n.MsgVaultScopeOwnerOnly, nil, 403, nil},
			{"personal owner required", "owner", "shared", `{"scope":"personal"}`, i18n.MsgVaultPersonalOwnerRequired, nil, 400, nil},
			{"team required", "owner", "personal", `{"scope":"team"}`, i18n.MsgRequired, []any{"team_id"}, 400, nil},
			{"invalid team", "owner", "personal", `{"team_id":"invalid"}`, i18n.MsgInvalidID, []any{"team"}, 400, nil},
			{"collision", "owner", "personal", `{"scope":"shared"}`, i18n.MsgVaultScopeConflict, nil, 409, vaultScopePGError{}},
			{"update failed", "owner", "personal", `{"scope":"shared"}`, i18n.MsgVaultUpdateFailed, nil, 500, fmt.Errorf("private database detail")},
			{"stale update", "owner", "personal", `{"scope":"shared"}`, i18n.MsgVaultUpdateConflict, nil, 409, fmt.Errorf("private database detail: %w", sql.ErrNoRows)},
		} {
			t.Run(locale+"/"+tc.name, func(t *testing.T) {
				st := newFakeVaultStore()
				agentID, docID := uuid.NewString(), uuid.NewString()
				doc := &store.VaultDocument{ID: docID, TenantID: masterTenant.String(), Scope: tc.scope, Path: "note.md"}
				if tc.scope == "personal" {
					doc.AgentID = &agentID
				}
				st.docs[docID], st.upsertErr = doc, tc.storeErr
				h := &VaultHandler{store: st}
				req := httptest.NewRequest(http.MethodPut, "/v1/vault/documents/"+docID, strings.NewReader(tc.payload))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept-Language", locale)
				req = req.WithContext(store.WithRole(masterCtx(), tc.role))
				req.SetPathValue("docID", docID)
				rr := httptest.NewRecorder()
				h.handleUpdateDocument(rr, req)
				var response struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				want := i18n.T(locale, tc.key, tc.args...)
				if rr.Code != tc.status || response.Error != want {
					t.Fatalf("response = %d %q, want %d %q", rr.Code, response.Error, tc.status, want)
				}
				if want == tc.key || strings.Contains(want, "%!") || strings.Contains(want, "private database") {
					t.Fatalf("missing, malformed or unsafe translation: %q", want)
				}
				if locale != i18n.LocaleEN && want == i18n.T(i18n.LocaleEN, tc.key, tc.args...) {
					t.Fatalf("%s unexpectedly fell back to English for %s", locale, tc.key)
				}
			})
		}
	}
}

func TestHandleUpdateDocument_StaleTitleCannotRestoreOlderScope(t *testing.T) {
	for _, payload := range []string{`{"title":"stale title"}`, `{"title":"stale title","scope":"personal"}`} {
		t.Run(payload, func(t *testing.T) {
			st := newFakeVaultStore()
			agentID, docID := uuid.NewString(), uuid.NewString()
			initialToken := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
			st.docs[docID] = &store.VaultDocument{ID: docID, TenantID: masterTenant.String(), AgentID: &agentID, Scope: "personal", Title: "Old", Path: "note.md", UpdatedAt: initialToken}
			// Another authorized request commits sharing after this handler's
			// detached read, before its title save. The fake uses the stores'
			// actual UpdatedAt/ErrNoRows convention rather than aliasing rows.
			st.beforeUpsert = func() {
				newer := *st.docs[docID]
				newer.Scope, newer.AgentID, newer.TeamID, newer.ChatID = "shared", nil, nil, nil
				newer.Title, newer.UpdatedAt = "Newer", initialToken.Add(time.Second)
				st.docs[docID] = &newer
			}
			bus := &fakeEventBus{store: st}
			h := &VaultHandler{store: st, eventBus: bus}
			req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(store.WithRole(masterCtx(), "operator"))
			req.SetPathValue("docID", docID)
			rr := httptest.NewRecorder()
			h.handleUpdateDocument(rr, req)
			var response struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if rr.Code != http.StatusConflict || response.Error != i18n.T(i18n.LocaleEN, i18n.MsgVaultUpdateConflict) {
				t.Fatalf("stale save response: %d %s", rr.Code, rr.Body.String())
			}
			got := st.docs[docID]
			if got.Scope != "shared" || got.AgentID != nil || got.Title != "Newer" || !got.UpdatedAt.Equal(initialToken.Add(time.Second)) {
				t.Fatalf("stale title restored obsolete snapshot: %+v", got)
			}
			if st.upsertCall != 1 || len(bus.published) != 0 {
				t.Fatal("stale save retried or published enrichment")
			}
		})
	}
}

func TestHandleUpdateDocument_StaleContentPreservesFile(t *testing.T) {
	for _, content := range []string{"", "stale replacement"} {
		t.Run(content, func(t *testing.T) {
			st := newFakeVaultStore()
			agentID, docID := uuid.NewString(), uuid.NewString()
			token := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
			st.docs[docID] = &store.VaultDocument{ID: docID, TenantID: masterTenant.String(), AgentID: &agentID, Scope: "personal", Path: "note.md", UpdatedAt: token}
			bus := &fakeEventBus{store: st}
			h := &VaultHandler{store: st, workspace: t.TempDir(), eventBus: bus}
			hash, err := h.writeDocumentContent(h.workspace, "note.md", []byte("newer file bytes"))
			if err != nil {
				t.Fatal(err)
			}
			st.beforeUpsert = func() {
				newer := *st.docs[docID]
				newer.Scope, newer.AgentID = "shared", nil
				newer.ContentHash, newer.UpdatedAt = hash, token.Add(time.Second)
				st.docs[docID] = &newer
			}
			req := newJSONRequest(t, http.MethodPut, "/", map[string]string{"content": content})
			req.SetPathValue("docID", docID)
			rr := httptest.NewRecorder()
			h.handleUpdateDocument(rr, req)
			if rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), i18n.T(i18n.LocaleEN, i18n.MsgVaultUpdateConflict)) {
				t.Fatalf("stale content response: %d %s", rr.Code, rr.Body.String())
			}
			got, err := os.ReadFile(filepath.Join(h.workspace, "note.md"))
			if err != nil || string(got) != "newer file bytes" {
				t.Fatalf("stale content changed disk: %q, %v", got, err)
			}
			if doc := st.docs[docID]; doc.Scope != "shared" || doc.AgentID != nil || doc.ContentHash != hash {
				t.Fatalf("stale content changed newer row: %+v", doc)
			}
			if st.upsertCall != 1 || len(bus.published) != 0 {
				t.Fatal("stale content retried or published enrichment")
			}
		})
	}
}

func TestHandleUpdateDocument_ContentCommitFailureRestoresDisk(t *testing.T) {
	for _, existed := range []bool{false, true} {
		t.Run(fmt.Sprint(existed), func(t *testing.T) {
			st := newFakeVaultStore()
			docID := uuid.NewString()
			st.docs[docID] = &store.VaultDocument{ID: docID, TenantID: masterTenant.String(), Scope: "shared", Path: "note.md", ContentHash: "old-hash"}
			st.contentCommitErr = fmt.Errorf("private database commit detail")
			bus := &fakeEventBus{store: st}
			h := &VaultHandler{store: st, workspace: t.TempDir(), eventBus: bus}
			if existed {
				if _, err := h.writeDocumentContent(h.workspace, "note.md", []byte("original")); err != nil {
					t.Fatal(err)
				}
			}
			req := newJSONRequest(t, http.MethodPut, "/", map[string]string{"content": "replacement"})
			req.SetPathValue("docID", docID)
			rr := httptest.NewRecorder()
			h.handleUpdateDocument(rr, req)
			if rr.Code != http.StatusInternalServerError || strings.Contains(rr.Body.String(), "private database") {
				t.Fatalf("unsafe commit failure response: %d %s", rr.Code, rr.Body.String())
			}
			got, err := os.ReadFile(filepath.Join(h.workspace, "note.md"))
			if existed && (err != nil || string(got) != "original") {
				t.Fatalf("commit failure did not restore bytes: %q, %v", got, err)
			}
			if !existed && !os.IsNotExist(err) {
				t.Fatalf("commit failure left new file: %q, %v", got, err)
			}
			if st.docs[docID].ContentHash != "old-hash" || len(bus.published) != 0 {
				t.Fatal("commit failure changed hash or published enrichment")
			}
		})
	}
}

func TestHandleUpdateDocument_ContentConstraintFailureNeverWrites(t *testing.T) {
	st := newFakeVaultStore()
	docID := uuid.NewString()
	st.docs[docID] = &store.VaultDocument{ID: docID, TenantID: masterTenant.String(), Scope: "shared", Path: "note.md", ContentHash: "old-hash"}
	st.upsertErr = vaultScopePGError{}
	h := &VaultHandler{store: st, workspace: t.TempDir()}
	if _, err := h.writeDocumentContent(h.workspace, "note.md", []byte("original")); err != nil {
		t.Fatal(err)
	}
	req := newJSONRequest(t, http.MethodPut, "/", map[string]string{"content": "replacement"})
	req.SetPathValue("docID", docID)
	rr := httptest.NewRecorder()
	h.handleUpdateDocument(rr, req)
	if rr.Code != http.StatusConflict || strings.Contains(rr.Body.String(), "private database") {
		t.Fatalf("unsafe constraint failure response: %d %s", rr.Code, rr.Body.String())
	}
	got, err := os.ReadFile(filepath.Join(h.workspace, "note.md"))
	if err != nil || string(got) != "original" || st.docs[docID].ContentHash != "old-hash" {
		t.Fatalf("constraint failure changed content: %q, %v", got, err)
	}
}

func TestHandleUpdateDocument_ContentCallbackFailureRollsBackRow(t *testing.T) {
	st := newFakeVaultStore()
	docID := uuid.NewString()
	st.docs[docID] = &store.VaultDocument{ID: docID, TenantID: masterTenant.String(), Scope: "shared", Path: "note.md", ContentHash: "old-hash"}
	bus := &fakeEventBus{store: st}
	h := &VaultHandler{store: st, workspace: t.TempDir(), eventBus: bus}
	// A directory where a regular content file is expected fails in callback,
	// after CAS staging. Neither the staged hash nor an event may survive.
	if err := os.Mkdir(filepath.Join(h.workspace, "note.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	req := newJSONRequest(t, http.MethodPut, "/", map[string]string{"content": "replacement"})
	req.SetPathValue("docID", docID)
	rr := httptest.NewRecorder()
	h.handleUpdateDocument(rr, req)
	if rr.Code != http.StatusInternalServerError || st.docs[docID].ContentHash != "old-hash" || len(bus.published) != 0 {
		t.Fatalf("callback failure was not rolled back: %d %s", rr.Code, rr.Body.String())
	}
}

// Embedding only the legacy interface deliberately hides the optional method.
type metadataOnlyVaultStore struct{ store.VaultStore }

func TestHandleUpdateDocument_ContentUnsupportedNeverWrites(t *testing.T) {
	for _, locale := range []string{i18n.LocaleEN, i18n.LocaleVI, i18n.LocaleZH} {
		t.Run(locale, func(t *testing.T) {
			st := newFakeVaultStore()
			docID := uuid.NewString()
			st.docs[docID] = &store.VaultDocument{ID: docID, TenantID: masterTenant.String(), Scope: "shared", Path: "note.md"}
			h := &VaultHandler{store: &metadataOnlyVaultStore{st}, workspace: t.TempDir()}
			req := newJSONRequest(t, http.MethodPut, "/", map[string]string{"content": "replacement"})
			req.Header.Set("Accept-Language", locale)
			req.SetPathValue("docID", docID)
			rr := httptest.NewRecorder()
			h.handleUpdateDocument(rr, req)
			var response struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if rr.Code != http.StatusNotImplemented || response.Error != i18n.T(locale, i18n.MsgVaultContentUpdateUnavailable) || st.upsertCall != 0 {
				t.Fatalf("unsafe fallback: %d %s", rr.Code, rr.Body.String())
			}
			if _, err := os.Stat(filepath.Join(h.workspace, "note.md")); !os.IsNotExist(err) {
				t.Fatalf("unsupported store wrote content: %v", err)
			}
		})
	}
}

func TestHandleCreateDocument_InvalidScopeLocalized(t *testing.T) {
	for _, locale := range []string{i18n.LocaleEN, i18n.LocaleVI, i18n.LocaleZH} {
		t.Run(locale, func(t *testing.T) {
			h := &VaultHandler{}
			req := newJSONRequest(t, http.MethodPost, "/v1/vault/documents", map[string]string{"path": "note.md", "title": "Note", "scope": "invalid"})
			req.Header.Set("Accept-Language", locale)
			rr := httptest.NewRecorder()
			h.handleCreateDocument(rr, req)
			var response struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if rr.Code != http.StatusBadRequest || response.Error != i18n.T(locale, i18n.MsgVaultInvalidScope) {
				t.Fatalf("unexpected localized create error: %d %s", rr.Code, rr.Body.String())
			}
		})
	}
}
