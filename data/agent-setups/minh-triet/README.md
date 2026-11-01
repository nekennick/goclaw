# Minh Triết · Q&A YGM

Agent key: `minh-triet`. Provider/model: `openai-codex` / `gpt-6.1-sol`.
Team: `YGM-AI-TEAM-CAOLANH`; role: `reviewer`; one-way Agent Link: Kim → Minh Triết.

## Native configuration

Created using the GoClaw HTTP API and WebSocket RPC only; no gateway source changes, direct SQL writes, new Zalo account, cron, or broader sharing permissions.

Supported agent files: AGENTS.md, SOUL.md, IDENTITY.md, CAPABILITIES.md, USER_PREDEFINED.md, HEARTBEAT.md. The installed API does not expose direct edits to AGENTS_CORE.md / AGENTS_TASK.md; the mandatory policy is in CAPABILITIES.md, which is loaded in full, task and minimal modes. All three previews are verified.

Read-only tools: vault_search, vault_read, read_file, list_files (plus the native Read alias). No team_tasks tool: runtime records the assigned review's response and handles completion, so the reviewer cannot invoke task approval or mutations through that tool. No exec, writes, delegate, messaging, cron, private session access, self-evolution, shared memory or proactive heartbeat.

## Operating policy

- New/materially revised SOP/training and substantive warehouse/customer/personnel analyses: Kim requests review before treating them as verified.
- Routine facts, cosmetic edits and unchanged recurring reports: no extra review unless requested or a material data exception exists.
- Verdict: APPROVED, REJECTED: NEEDS_CHANGES, or REJECTED: INSUFFICIENT_EVIDENCE, with evidence/impact/correction.
- Two revision rounds maximum; unresolved material issues go to Anh Khoa.
- Task completion is NOT a passed quality review. Dependency ordering is NOT an enforced quality gate. AI review does not authorize stock/debt changes, disposal, employee decisions or SOP publication.
- Team channel restrictions remain unchanged. Denied access must be disclosed, not bypassed by broadening settings. Direct Agent Link review can be used where authorized, without pretending it created a task-board approval gate.

## Administration

`admin.ps1` performs native API operations and never prints gateway credentials. Setup deliberately refuses to overwrite an existing agent; ResumeSetup is only for an incomplete inactive setup.

Actions: Setup, ResumeSetup, Inspect, Verify, Test, LinkTest, ConnectKim, TeamTest, CheckTeamTest.
ConnectKim appends a marked review policy to Kim's AGENTS.md and CAPABILITIES.md, preserving old content and checking for concurrent changes. Before-images are backed up under `.artifacts/minh-triet-20261101/kim-prompts-before.clixml`; they may contain internal business rules and must not be published or indexed into Vault.

Test calls return only through HTTP or dashboard task scope, never staff/boss Zalo groups. Test tasks are visibly marked `[TEST]` and are retained for audit, not silently deleted.

## Verification completed — 2026-11-01

- Agent UUID: `01a19795-7ce1-758a-a54d-cd3502ba5c16`, active; exact requested display name preserved.
- Effective tool preview contains only list_files/read_file/vault_search/vault_read plus Read alias. Full/task/minimal policies verified for Minh Triết and Kim.
- Incorrect SOP test: REJECTED: NEEDS_CHANGES. Seven real tool calls; identified incorrect counting time, reversed import/export restrictions, unauthorized stock balancing, and incorrect extension to Binh Duong; cited actually read KB documents.
- Native Team task `01a19798-175f-745d-872d-836839c685a7`: completed with REJECTED: INSUFFICIENT_EVIDENCE, correctly refusing an unsupported all-branches conclusion rather than alleging the conclusion was proven false.
- Real Kim → Minh Triết sync Agent Link test: reviewer child trace completed under parent `01a19798-e739-74e6-8884-561e250e821d`; APPROVED for a deliberately cautious logical statement. Kim returned the actual verdict. No task-board gate claimed for this direct delegation.
- KB fingerprint unchanged: 17 documents, `65443271fe36d2a6080c9693c35d8617`.
- No system Go/UI code changes, restart, provider/model edits to existing agents, Team channel/access/sharing changes, cron changes or Zalo test messages.
- Existing Team channel-type versus instance-name restriction remains unchanged; the successful internal tests do not assert every production Zalo task-board path is unrestricted.

## Disable / rollback

Use Agent settings to set `minh-triet` inactive; remove its Team membership and Kim → Minh Triết link using the native management UI if required. Remove ONLY the `YGM_MINH_TRIET_REVIEW_POLICY_V1` marked section from Kim's AGENTS.md and CAPABILITIES.md; preserve all other rules. The backup exists for comparison, not for blindly overwriting newer user edits.
