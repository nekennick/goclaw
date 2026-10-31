# Vault shared-scope rollout

This fix concerns Vault document identity and ownership, not channel routing or warehouse policy. Deploy it before changing the Yagami knowledge packs to shared. Do not deploy or change production document scope merely because a build succeeds.

## Invariants

- Scope changes retain document ID and path. Links and PostgreSQL body chunks remain attached to that ID.
- Shared documents have no agent, team, or chat owner. Scope changes require an authorized administrative request, not an agent's prompt.
- A destination collision must fail, not merge or overwrite another registration.
- Reloading or writing an already shared source must update that registration rather than create a private copy. Never adopt another agent's private registration as shared.
- Missing identity, conflicting ownership, and ambiguous registrations must fail closed. Tenant boundaries remain enforced.

## Pre-deployment checks

Run unit tests for HTTP transitions, real-store identity/collision tests, and rescan/write registration regression tests. Check PostgreSQL and SQLite builds. Verify both Web Vault editors send a supported transition. Preserve unrelated local Zalo changes and do not run a repository-wide destructive reset or automatic rewrite.

## Production handoff (separate approval)

1. Retain the currently deployed image digest and snapshot the exact target document metadata (ID, tenant, ownership, scope, path, content hash, body-index hash). Keep database backups under existing backup policy; never copy credentials into knowledge packs.
2. Build a separately tagged candidate image. Do not overwrite the known-good rollback tag. Run transition/reload/source-update checks against an isolated database first.
3. Obtain approval for the restart window and the exact sharing audience. `shared` makes documents readable by agents within the tenant, not only a named team.
4. Replace only the gateway image in the deployment's existing Compose files. Never use `down -v`, delete a database volume, or recreate credentials to apply this fix.
5. After gateway health passes, change only the approved KB registrations. Preserve their paths so existing cron scripts continue to find source files. Grant Vault discovery/read tools to intended agents separately; do not remove their existing tools.
6. Verify search/read from each intended agent, reload twice, and publish a controlled source update. Confirm one registration per intended identity, unchanged IDs/paths, current content hashes/body indexes, and preserved links. Recheck scheduled reports outside the active reporting window.

If validation fails, stop the rollout, restore the gateway's previous image, and restore only explicitly changed scope metadata after checking for concurrent edits. Do not restore an entire database merely to revert scope settings. No schema migration is intended for this behavioral fix.

## Remaining configuration/data work

The Yagami manifest and canonical JSON versions must agree before agents treat the pack as a released revision. JSONL is a derivative format; the current 185 rows match the canonical JSON already indexed. Embeddings are optional for this patch: current keyword/body search must continue working without an embedding provider. Private learner records must not become shared.

## Verification record — 2026-10-31

- Go 1.26 container; PostgreSQL tests use a separate temporary pgvector/PG18 container with no production mounts or published ports.
- Relevant Vault/HTTP/shared-lookup/store tests passed with race detection; SQLite Vault/HTTP/integration regressions passed.
- PostgreSQL integration `TestStoreVault*` passed with race detection against the temporary database, including stable ID, links/chunks, collision, and tenant guards.
- Package-wide vault/tools/http/i18n tests passed with Docker `--init`, excluding Ollama URL tests. Those Ollama tests also fail on an unmodified HEAD archive because localhost test endpoints are rewritten to host.docker.internal inside Docker. Without `--init`, process-reaping tests also fail; they pass with the container init process.
- Both PostgreSQL/SQLite builds and `go vet` passed. For final container checks `-buildvcs=false` avoids a slow `git status` scan over the Windows bind mount; source code and runtime behavior are unchanged. `go fix -diff` was inspected without applying a repository-wide rewrite; the checkout contains widespread CRLF-related diffs and unrelated user changes must be preserved.
- Final production read-only check: all 17 KB registrations remain personal. No production sharing, image deployment, gateway restart, migration, cron modification, or credential change was performed.
- Source/API docs include the scope transition and optional transactional content-update contract. Existing Web payloads remain supported. Desktop UI and in-repo CLI have no Vault scope consumer to change; SQLite backend parity is included. External CLI/runtime release parity remains unverified.

Content rollback handles reported write/commit failures, not abrupt machine/process crashes across the database and filesystem. Direct external edits to the same files remain outside the database transaction; maintain the existing backup policy and refresh indexing after publishing source changes.
