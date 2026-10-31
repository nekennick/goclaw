package http

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nextlevelbuilder/goclaw/internal/eventbus"
	"github.com/nextlevelbuilder/goclaw/internal/i18n"
	"github.com/nextlevelbuilder/goclaw/internal/store"
	"github.com/nextlevelbuilder/goclaw/internal/vault"
)

// Optional capability: stores validate CAS and constraints under their DB lock
// before writing content, then restore disk on callback/commit failure. Keeping
// it separate leaves metadata-only VaultStore implementations source-compatible.
type vaultContentUpdater interface {
	UpsertDocumentWithContent(context.Context, *store.VaultDocument, func() (func() error, error)) error
}

// writeDocumentContentWithRestore snapshots bytes only inside the workspace.
// The returned undo is independent of request cancellation and is also returned
// after a partial write error, so the store can restore before releasing its lock.
func (h *VaultHandler) writeDocumentContentWithRestore(workspace, relPath string, content []byte) (func() error, error) {
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return nil, err
	}
	realWS, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		return nil, err
	}
	target := filepath.Join(realWS, filepath.Clean(relPath))
	inside := func(path string) bool {
		return path != realWS && strings.HasPrefix(path, realWS+string(os.PathSeparator))
	}
	if !inside(target) {
		return nil, os.ErrInvalid
	}
	root, err := os.OpenRoot(realWS)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := root.Close(); err != nil {
			slog.Warn("vault.content snapshot root close failed", "error", err)
		}
	}()
	rootInfo, err := root.Stat(".")
	if err != nil {
		return nil, err
	}
	var previous []byte
	existed := false
	// Lstat before any resolution/open: even a dangling symlink is an
	// existing entry, never an absent file that an undo may safely remove.
	entry, err := root.Lstat(relPath)
	switch {
	case err == nil:
		if !entry.Mode().IsRegular() {
			return nil, os.ErrInvalid
		}
		// Root confines ancestor symlinks; O_NOFOLLOW closes the final-entry
		// replacement race on Unix. Verify the opened handle, not just Lstat.
		f, err := root.OpenFile(relPath, os.O_RDONLY|oNoFollow, 0)
		if err != nil {
			return nil, err
		}
		fi, statErr := f.Stat()
		if statErr != nil || !fi.Mode().IsRegular() {
			return nil, errors.Join(statErr, os.ErrInvalid, f.Close())
		}
		previous, err = io.ReadAll(f)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		existed = true
	case !os.IsNotExist(err):
		return nil, err
	}
	restore := func() error {
		restoreRoot, err := os.OpenRoot(realWS)
		if err != nil {
			return err
		}
		defer func() {
			if err := restoreRoot.Close(); err != nil {
				slog.Warn("vault.content restore root close failed", "error", err)
			}
		}()
		// Do not restore into a replacement workspace after a directory swap.
		currentRootInfo, err := restoreRoot.Stat(".")
		if err != nil {
			return err
		}
		if !os.SameFile(rootInfo, currentRootInfo) {
			return os.ErrInvalid
		}
		if existed {
			f, err := restoreRoot.OpenFile(relPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|oNoFollow, 0o644)
			if err != nil {
				return err
			}
			_, writeErr := f.Write(previous)
			return errors.Join(writeErr, f.Close())
		}
		// Root.Remove confines ancestor resolution, unlike check-then-Remove.
		if err := restoreRoot.Remove(relPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	// Keep the actual write under the same confined root as the snapshot.
	// For an absent target, O_EXCL must not overwrite an entry created after
	// the snapshot (and its undo must never remove that other entry).
	if err := root.MkdirAll(filepath.Dir(relPath), 0o755); err != nil {
		return nil, err
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC | oNoFollow
	if !existed {
		flags |= os.O_EXCL
	}
	f, err := root.OpenFile(relPath, flags, 0o644)
	if err != nil {
		return nil, err
	}
	_, writeErr := f.Write(content)
	return restore, errors.Join(writeErr, f.Close())
}

// writeDocumentContent writes the given content bytes to a tenant-workspace-
// relative path, after validating the path stays inside the workspace and uses
// an allowed extension. Returns the SHA-256 content hash on success.
//
// This is the shared writer used by handleCreateDocument and handleUpdateDocument
// to materialise inline JSON `content` payloads on disk, so the enrichment
// pipeline (which reads files by path) can later compute summaries, embeddings
// and links.
func (h *VaultHandler) writeDocumentContent(workspace, relPath string, content []byte) (string, error) {
	// Ensure the tenant workspace exists before resolving symlinks. Non-master
	// tenants live under workspace/tenants/{slug}/ which handleUpload creates on
	// demand (see vault_handler_upload.go) — without this, the very first JSON
	// content write into a fresh tenant would fail EvalSymlinks with ErrNotExist
	// and 500 the request.
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return "", err
	}

	// Resolve the workspace root through any symlinks so all containment checks
	// compare canonical paths.
	realWS, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		return "", err
	}
	target := filepath.Join(realWS, filepath.Clean(relPath))

	// Lexical containment: blocks "../" and absolute-path escapes.
	if target != realWS && !strings.HasPrefix(target, realWS+string(os.PathSeparator)) {
		slog.Warn("security.vault_symlink_escape",
			"site", "path_traversal", "attempted", target, "workspace", realWS)
		return "", os.ErrInvalid
	}

	// Symlink containment: a lexical prefix check is not enough — a symlink that
	// lives *inside* the workspace but points outside it (e.g. workspace/link ->
	// /etc) would otherwise let the write follow it and clobber a file beyond
	// the workspace. Resolve the deepest already-existing ancestor of the target
	// and confirm it still canonicalises to a path inside the workspace, before
	// any directory is created or byte written.
	for ancestor := filepath.Dir(target); ; {
		resolved, rerr := filepath.EvalSymlinks(ancestor)
		if rerr == nil {
			if resolved != realWS && !strings.HasPrefix(resolved, realWS+string(os.PathSeparator)) {
				slog.Warn("security.vault_symlink_escape",
					"site", "ancestor", "resolved", resolved, "workspace", realWS)
				return "", os.ErrInvalid
			}
			break
		}
		if !os.IsNotExist(rerr) {
			return "", rerr
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			break // walked to the filesystem root without an existing ancestor
		}
		ancestor = parent
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}

	// Fast-fail when the final component is already a symlink — gives a clean
	// rejection + security log without parsing OS-specific error codes. The
	// O_NOFOLLOW open below closes the TOCTOU window this Lstat leaves open
	// (an attacker could swap the file for a symlink between calls).
	if fi, lerr := os.Lstat(target); lerr == nil && fi.Mode()&os.ModeSymlink != 0 {
		slog.Warn("security.vault_symlink_escape",
			"site", "final_lstat", "target", target, "workspace", realWS)
		return "", os.ErrInvalid
	}

	// O_NOFOLLOW makes the kernel refuse to follow a symlink at the final
	// component atomically with the open, removing the Lstat→Open race entirely
	// on Unix. On Windows oNoFollow == 0 and the Lstat above is best-effort.
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|oNoFollow, 0o644)
	if err != nil {
		if isSymlinkLoopErr(err) {
			slog.Warn("security.vault_symlink_escape",
				"site", "open_nofollow", "target", target, "workspace", realWS)
			return "", os.ErrInvalid
		}
		return "", err
	}
	if _, werr := f.Write(content); werr != nil {
		_ = f.Close() // best-effort cleanup; the write error is the real one
		return "", werr
	}
	// Explicit Close so a delayed FS error (e.g. disk-full at flush) surfaces
	// instead of being silently swallowed by a deferred close.
	if cerr := f.Close(); cerr != nil {
		return "", cerr
	}
	return vault.ContentHash(content), nil
}

// publishDocUpserted enqueues the standard EventVaultDocUpserted so the
// enrichment worker re-runs summary/embedding/link extraction for this doc.
// No-op when the event bus is not wired (some test setups).
func (h *VaultHandler) publishDocUpserted(tenantID uuid.UUID, agentID, docID, relPath, hash, workspace string) {
	if h.eventBus == nil {
		return
	}
	tenantIDStr := tenantID.String()
	// Start enrichment progress before publishing so the worker pool can't drain
	// the event and finish before the progress tracker registers it — same
	// ordering handleUpload relies on to avoid that race.
	if h.enrichProgress != nil {
		h.enrichProgress.Start(1, tenantID)
	}
	h.eventBus.Publish(eventbus.DomainEvent{
		ID:        uuid.Must(uuid.NewV7()).String(),
		Type:      eventbus.EventVaultDocUpserted,
		SourceID:  docID + ":" + hash,
		TenantID:  tenantIDStr,
		AgentID:   agentID,
		Timestamp: time.Now(),
		Payload: eventbus.VaultDocUpsertedPayload{
			DocID:       docID,
			TenantID:    tenantIDStr,
			AgentID:     agentID,
			Path:        relPath,
			ContentHash: hash,
			Workspace:   workspace,
		},
	})
}

// handleListAllDocuments lists vault documents across all agents in tenant.
// Optional query param agent_id to filter by specific agent.
func (h *VaultHandler) handleListAllDocuments(w http.ResponseWriter, r *http.Request) {
	tenantID := store.TenantIDFromContext(r.Context())
	agentID := r.URL.Query().Get("agent_id")
	opts := h.parseListOpts(r)

	// Validate team membership if specific team requested.
	if opts.TeamID != nil && *opts.TeamID != "" {
		if !h.validateTeamMembership(r.Context(), w, *opts.TeamID) {
			return
		}
	}
	// Non-owner without team_id filter: show personal + user's teams.
	if opts.TeamID == nil && !store.IsOwnerRole(r.Context()) {
		h.applyNonOwnerTeamScope(r.Context(), &opts)
	}

	docs, err := h.store.ListDocuments(r.Context(), tenantID.String(), agentID, opts)
	if err != nil {
		slog.Warn("vault.list_all failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if docs == nil {
		docs = []store.VaultDocument{}
	}
	total, cntErr := h.store.CountDocuments(r.Context(), tenantID.String(), agentID, opts)
	if cntErr != nil {
		slog.Warn("vault.count failed", "error", cntErr)
	}
	writeJSON(w, http.StatusOK, vaultDocListResponse{Documents: docs, Total: total})
}

// handleListDocuments lists vault documents for a specific agent.
func (h *VaultHandler) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	tenantID := store.TenantIDFromContext(r.Context())
	agentID := r.PathValue("agentID")
	opts := h.parseListOpts(r)

	if opts.TeamID != nil && *opts.TeamID != "" {
		if !h.validateTeamMembership(r.Context(), w, *opts.TeamID) {
			return
		}
	}
	if opts.TeamID == nil && !store.IsOwnerRole(r.Context()) {
		h.applyNonOwnerTeamScope(r.Context(), &opts)
	}

	docs, err := h.store.ListDocuments(r.Context(), tenantID.String(), agentID, opts)
	if err != nil {
		slog.Warn("vault.list failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if docs == nil {
		docs = []store.VaultDocument{}
	}
	total, cntErr := h.store.CountDocuments(r.Context(), tenantID.String(), agentID, opts)
	if cntErr != nil {
		slog.Warn("vault.count failed", "error", cntErr)
	}
	writeJSON(w, http.StatusOK, vaultDocListResponse{Documents: docs, Total: total})
}

// handleGetDocument returns a single vault document by ID, scoped to the agent.
func (h *VaultHandler) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	tenantID := store.TenantIDFromContext(r.Context())
	agentID := r.PathValue("agentID")
	docID := r.PathValue("docID")

	doc, err := h.store.GetDocumentByID(r.Context(), tenantID.String(), docID)
	if err != nil {
		slog.Warn("vault.get failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if doc == nil || (agentID != "" && doc.AgentID != nil && *doc.AgentID != agentID) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
		return
	}
	// Verify team boundary — non-owner must be team member to view team docs.
	if doc.TeamID != nil && *doc.TeamID != "" && !store.IsOwnerRole(r.Context()) {
		if !h.validateTeamMembership(r.Context(), w, *doc.TeamID) {
			return
		}
	}
	writeJSON(w, http.StatusOK, doc)
}

// handleCreateDocument creates a new vault document.
//
// `content` semantics — symmetric with handleUpdateDocument:
//   - field omitted (nil pointer): metadata-only stub, no file write, no event
//     (typically followed by a rescan that materialises the file from disk).
//   - field present and non-empty: bytes materialised at <tenant-workspace>/<path>,
//     SHA-256 hash stored on the row, EventVaultDocUpserted emitted so the
//     enrichment worker computes summary/embeddings/links — same code path the
//     multipart /v1/vault/upload endpoint and the filesystem rescan use.
//   - field present and empty (""): a 0-byte file is written and the event still
//     fires. Same behaviour as PUT — pick the right shape on the client side.
func (h *VaultHandler) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	locale := extractLocale(r)
	tenantID := store.TenantIDFromContext(r.Context())
	agentID := r.PathValue("agentID")

	var body struct {
		Path     string         `json:"path"`
		Title    string         `json:"title"`
		Content  *string        `json:"content"` // nil=no write; ""=write empty file; "data"=write bytes (mirrors PUT)
		DocType  string         `json:"doc_type"`
		Scope    string         `json:"scope"`
		TeamID   string         `json:"team_id"`
		Metadata map[string]any `json:"metadata"`
	}
	if !bindJSON(w, r, locale, &body) {
		return
	}
	if body.Path == "" || body.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path and title are required"})
		return
	}
	if strings.Contains(body.Path, "..") || strings.HasPrefix(body.Path, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid path"})
		return
	}
	if body.DocType == "" {
		body.DocType = "note"
	}
	if body.Scope == "" {
		body.Scope = "personal"
	}
	if !validDocType(body.DocType) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid doc_type"})
		return
	}
	if !validScope(body.Scope) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": i18n.T(locale, i18n.MsgVaultInvalidScope)})
		return
	}
	// Validate extension upfront if caller wants to write content — matches the
	// whitelist enforced by /v1/vault/upload so the two endpoints accept the
	// same file types. nil pointer = "no content field", which skips the check.
	if body.Content != nil {
		ext := strings.ToLower(filepath.Ext(body.Path))
		if !allowedUploadExts[ext] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported file type: " + ext})
			return
		}
	}

	doc := &store.VaultDocument{
		TenantID: tenantID.String(),
		Path:     body.Path,
		Title:    body.Title,
		DocType:  body.DocType,
		Scope:    body.Scope,
		Metadata: body.Metadata,
	}
	if agentID != "" {
		doc.AgentID = &agentID
	} else if doc.Scope == "personal" {
		doc.Scope = "shared" // no agent → shared scope
	}
	if body.TeamID != "" {
		if !h.validateTeamMembership(r.Context(), w, body.TeamID) {
			return
		}
		doc.TeamID = &body.TeamID
		if body.Scope == "personal" {
			doc.Scope = "team"
		}
	}

	// Persist file content if provided, before the DB upsert so the
	// content_hash column is set in a single write.
	var (
		wsPath        string
		writtenHash   string
		contentWasSet bool
	)
	if body.Content != nil {
		wsPath = h.resolveTenantWorkspace(r.Context())
		if wsPath == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspace not available"})
			return
		}
		hash, werr := h.writeDocumentContent(wsPath, body.Path, []byte(*body.Content))
		if werr != nil {
			slog.Warn("vault.create: write content failed", "path", body.Path, "error", werr)
			if errors.Is(werr, os.ErrInvalid) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path escapes workspace"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to write content"})
			return
		}
		doc.ContentHash = hash
		writtenHash = hash
		contentWasSet = true
	}

	if err := h.store.UpsertDocument(r.Context(), doc); err != nil {
		slog.Warn("vault.create failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Fire enrichment after the DB row exists so the worker can locate the doc.
	if contentWasSet {
		evAgent := ""
		if doc.AgentID != nil {
			evAgent = *doc.AgentID
		}
		h.publishDocUpserted(tenantID, evAgent, doc.ID, body.Path, writtenHash, wsPath)
	}

	// Re-fetch by ID (set via RETURNING) — unambiguous even when same path exists across teams.
	created, _ := h.store.GetDocumentByID(r.Context(), tenantID.String(), doc.ID)
	if created != nil {
		writeJSON(w, http.StatusCreated, created)
	} else {
		writeJSON(w, http.StatusCreated, doc)
	}
}

// handleUpdateDocument updates an existing vault document.
func (h *VaultHandler) handleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	locale := extractLocale(r)
	tenantID := store.TenantIDFromContext(r.Context())
	agentID := r.PathValue("agentID")
	docID := r.PathValue("docID")

	existing, err := h.store.GetDocumentByID(r.Context(), tenantID.String(), docID)
	if err != nil || existing == nil || (agentID != "" && existing.AgentID != nil && *existing.AgentID != agentID) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
		return
	}
	// Verify team boundary before any metadata or content rewrite. PUT can
	// materialize bytes on disk, so it must match GET/DELETE team access rules.
	if existing.TeamID != nil && *existing.TeamID != "" && !store.IsOwnerRole(r.Context()) {
		if !h.validateTeamMembership(r.Context(), w, *existing.TeamID) {
			return
		}
	}

	var body struct {
		Title    *string        `json:"title"`
		Content  *string        `json:"content"` // nil=no change; "" clears the file; non-empty rewrites it
		DocType  *string        `json:"doc_type"`
		Scope    *string        `json:"scope"`
		TeamID   *string        `json:"team_id"` // nil=no change, ""=clear, "uuid"=set
		Metadata map[string]any `json:"metadata"`
	}
	if !bindJSON(w, r, locale, &body) {
		return
	}

	// Scope moves are metadata-only: a later namespace collision must never
	// leave rewritten file bytes behind. Even an unchanged scope field must
	// be omitted from a content request.
	if body.Content != nil && (body.Scope != nil || body.TeamID != nil) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": i18n.T(locale, i18n.MsgVaultContentScopeSeparate)})
		return
	}
	if body.Scope != nil && !validScope(*body.Scope) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": i18n.T(locale, i18n.MsgVaultInvalidScope)})
		return
	}
	// Work on a detached value; rejected validation must not mutate a row
	// returned by an in-memory store or another caller-owned implementation.
	next := *existing
	existing = &next
	targetScope := existing.Scope
	if body.Scope != nil {
		targetScope = *body.Scope
	}
	if body.TeamID != nil {
		if *body.TeamID != "" {
			if body.Scope != nil && targetScope != "team" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": i18n.T(locale, i18n.MsgVaultTeamScopeRequired)})
				return
			}
			targetScope = "team"
		} else if body.Scope == nil {
			targetScope = "personal"
		}
	}
	scopeChanged := targetScope != existing.Scope
	teamChanged := body.TeamID != nil && ((existing.TeamID == nil && *body.TeamID != "") || (existing.TeamID != nil && *existing.TeamID != *body.TeamID))
	// Keep the existing owner-only reassignment policy for all namespace
	// changes, not just team_id. RoleAdmin alone is not a tenant-admin check;
	// this handler has no TenantStore to verify tenant administration.
	if (scopeChanged || teamChanged) && !store.IsOwnerRole(r.Context()) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": i18n.T(locale, i18n.MsgVaultScopeOwnerOnly)})
		return
	}
	if body.Scope != nil || body.TeamID != nil {
		switch targetScope {
		case "shared":
			if scopeChanged || teamChanged {
				existing.AgentID, existing.TeamID, existing.ChatID = nil, nil, nil
			}
		case "personal":
			// PUT does not accept an agent owner target. Never infer one from
			// the route (which is only an access filter), or restore a lost owner.
			if existing.AgentID == nil || *existing.AgentID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": i18n.T(locale, i18n.MsgVaultPersonalOwnerRequired)})
				return
			}
			if scopeChanged || teamChanged {
				existing.TeamID, existing.ChatID = nil, nil
			}
		case "team":
			targetTeam := existing.TeamID
			if body.TeamID != nil {
				targetTeam = body.TeamID
			}
			if targetTeam == nil || *targetTeam == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": i18n.T(locale, i18n.MsgRequired, "team_id")})
				return
			}
			// Validate before the owner/nil-store membership bypass.
			tid, err := uuid.Parse(*targetTeam)
			if err != nil || tid == uuid.Nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": i18n.T(locale, i18n.MsgInvalidID, "team")})
				return
			}
			if !h.validateTeamMembership(r.Context(), w, *targetTeam) {
				return
			}
			existing.AgentID, existing.TeamID = nil, targetTeam
			if scopeChanged || teamChanged {
				existing.ChatID = nil
			}
		}
		existing.Scope = targetScope
	}

	if body.Title != nil {
		existing.Title = *body.Title
	}
	if body.DocType != nil {
		existing.DocType = *body.DocType
	}
	if body.Metadata != nil {
		existing.Metadata = body.Metadata
	}

	// Content writes must run only after the store validates CAS under lock.
	var (
		wsPath        string
		writtenHash   string
		contentWasSet bool
		updateErr     error
	)
	if body.Content != nil {
		ext := strings.ToLower(filepath.Ext(existing.Path))
		if !allowedUploadExts[ext] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported file type: " + ext})
			return
		}
		wsPath = h.resolveTenantWorkspace(r.Context())
		if wsPath == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspace not available"})
			return
		}
		updater, ok := h.store.(vaultContentUpdater)
		if !ok {
			// Never fall back to write-before-CAS with an older implementation.
			writeJSON(w, http.StatusNotImplemented, map[string]string{"error": i18n.T(locale, i18n.MsgVaultContentUpdateUnavailable)})
			return
		}
		content := []byte(*body.Content)
		writtenHash = vault.ContentHash(content)
		existing.ContentHash = writtenHash
		updateErr = updater.UpsertDocumentWithContent(r.Context(), existing, func() (func() error, error) {
			// No store calls from this callback: the store holds its row/write lock.
			return h.writeDocumentContentWithRestore(wsPath, existing.Path, content)
		})
		contentWasSet = true
	} else {
		updateErr = h.store.UpsertDocument(r.Context(), existing)
	}

	if err := updateErr; err != nil {
		slog.Warn("vault.update failed", "error", err)
		// Preserve the fetched UpdatedAt CAS token through the metadata edit.
		// Both stores wrap ErrNoRows for stale/missing ID updates. Never retry
		// without the token: that could restore obsolete scope or ownership.
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": i18n.T(locale, i18n.MsgVaultUpdateConflict)})
			return
		}
		// Inspect typed/wrapped driver errors without leaking DB details or
		// matching fragile error strings. Both store implementations wrap errors.
		var pgErr interface{ SQLState() string }
		var sqliteErr interface{ Code() int }
		if (errors.As(err, &pgErr) && pgErr.SQLState() == "23505") ||
			(errors.As(err, &sqliteErr) && (sqliteErr.Code() == 2067 || sqliteErr.Code() == 1555)) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": i18n.T(locale, i18n.MsgVaultScopeConflict)})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": i18n.T(locale, i18n.MsgVaultUpdateFailed)})
		return
	}

	if contentWasSet {
		evAgent := ""
		if existing.AgentID != nil {
			evAgent = *existing.AgentID
		}
		h.publishDocUpserted(tenantID, evAgent, existing.ID, existing.Path, writtenHash, wsPath)
	}

	updated, _ := h.store.GetDocumentByID(r.Context(), tenantID.String(), docID)
	if updated != nil {
		writeJSON(w, http.StatusOK, updated)
	} else {
		writeJSON(w, http.StatusOK, existing)
	}
}

// handleDeleteDocument deletes a vault document by ID.
func (h *VaultHandler) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	tenantID := store.TenantIDFromContext(r.Context())
	agentID := r.PathValue("agentID")
	docID := r.PathValue("docID")

	existing, err := h.store.GetDocumentByID(r.Context(), tenantID.String(), docID)
	if err != nil || existing == nil || (agentID != "" && existing.AgentID != nil && *existing.AgentID != agentID) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
		return
	}

	// Verify team boundary before deletion.
	if existing.TeamID != nil && *existing.TeamID != "" && !store.IsOwnerRole(r.Context()) {
		if !h.validateTeamMembership(r.Context(), w, *existing.TeamID) {
			return
		}
	}

	// DeleteDocument without RunContext applies no team_id filter (broad match on tenant+agent+path).
	// This is safe because we pre-validated team membership above and use server-derived existing.Path.
	// Use the doc's actual agent_id (may be empty for team/shared docs).
	deleteAgentID := ""
	if existing.AgentID != nil {
		deleteAgentID = *existing.AgentID
	}
	if err := h.store.DeleteDocument(r.Context(), tenantID.String(), deleteAgentID, existing.Path); err != nil {
		slog.Warn("vault.delete failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
