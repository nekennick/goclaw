# Kim — acknowledgement and intermediate delivery

Configured via native GoClaw HTTP administration API on 2026-11-01 (Asia/Bangkok). No Go/UI code or prompt changes, no new agent, no gateway restart and no Zalo test messages.

## Settings

- Agent `quan-ly-kim`: custom `other_config.delivery_behavior`; see delivery-behavior.json.
- Quick acknowledgement: fixed template, zero delay: "Em đã nhận yêu cầu, em bắt đầu xử lý ngay ạ." Does not need an extra LLM call. No generated-ack timeout can suppress this fixed template.
- Intermediate: native sidecar_generated, provider `openai-codex`, model `gpt-6.1-terra`, timeout 10000ms, 256 output-token budget, 180-character limit.
- Channel `zalokim`: removed only `config.chat_behavior` so it inherits Agent delivery policy. Existing channel-level 2500ms/40/60-token overrides would otherwise take precedence.
- All other channel config and agent other_config values preserved and compared. Credentials fingerprints before/after match. Kim's main model remains gpt-6.1-sol; no cron changes. Channel reload reconnected Zalo successfully.

## Actual native behavior and limits

This is delivery-only messaging, not hidden chain-of-thought. The generated update is triggered by a tool-call event and is not a periodic timer or an 'after N seconds' threshold. The 10000ms timeout is a sidecar generation deadline, not a wait before reporting progress. The generated-progress path sends at most one update per run; separate explicit block replies/announcements from the main agent can also be delivered. A run doing no tools has no generated tool-progress event. Fast completion or an explicit block reply can suppress an in-flight sidecar update. Sidecar failure logs a warning and skips that update; it does not require deleting the agent.

## Verification

- Existing ChatBehavior/QuickAck/Intermediate/Delivery tests plus a configuration-specific mocked channel test passed with Go race detector in a read-only Docker mount.
- Simulated run emitted the exact acknowledgement before tool work and then a progress update, without repeated generated progress on another tool call; no messages were sent to real chats.
- Native provider Verify succeeded for gpt-6.1-terra. This confirms provider/model access, not a live Zalo sidecar message.
- Zalo log confirmed connection and listener start after native config reload; gateway health OK.
- Live end-to-end group delivery still needs a genuine user command; do not claim it was tested by the internal simulation.

## Restore without deleting the agent

The generated before-image is at `.artifacts/kim-delivery-20261101/before.clixml`. Keep it private; never index it into Vault. From the repository root:

```powershell
./data/agent-setups/quan-ly-kim/delivery-admin.ps1 -Action Rollback
```

Rollback restores only the two delivery-related fields and retains unrelated current settings. Do not delete/recreate the agent or database.
