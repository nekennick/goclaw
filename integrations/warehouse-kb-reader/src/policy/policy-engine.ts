import { VERIFIED_INVENTORY, type VerifiedDocItem } from "./verified-inventory.js";

export interface VaultDocCandidate {
  id: string;
  tenant_id: string;
  scope: string;
  agent_id: string | null;
  team_id: string | null;
  chat_id?: string | null;
  custom_scope?: string | null;
  path: string;
  title: string;
  content_hash: string;
  updated_at: string;
}

export type PolicyCheckResult =
  | { allowed: true; item: VerifiedDocItem }
  | { allowed: false; reason: string };

export class PolicyEngine {
  constructor(private readonly expectedTenantId: string) {}

  verify(doc: VaultDocCandidate): PolicyCheckResult {
    // 1. Must be in verified inventory allowlist
    const item = VERIFIED_INVENTORY.get(doc.id);
    if (!item) {
      return { allowed: false, reason: `Document ${doc.id} is not in verified inventory allowlist` };
    }

    // 2. Strict tenant check
    if (doc.tenant_id !== this.expectedTenantId) {
      return { allowed: false, reason: `Tenant mismatch: expected ${this.expectedTenantId}, got ${doc.tenant_id}` };
    }

    // 3. Strict scope check (shared only)
    if (doc.scope !== "shared") {
      return { allowed: false, reason: `Scope is not shared: got ${doc.scope}` };
    }

    // 4. Team, Agent, Chat, Custom Scope must be empty for shared KB
    if (doc.agent_id !== null && doc.agent_id !== undefined && doc.agent_id !== "") {
      return { allowed: false, reason: `agent_id must be null for shared doc` };
    }
    if (doc.team_id !== null && doc.team_id !== undefined && doc.team_id !== "") {
      return { allowed: false, reason: `team_id must be null for shared doc` };
    }
    if (doc.chat_id !== null && doc.chat_id !== undefined && doc.chat_id !== "") {
      return { allowed: false, reason: `chat_id must be null for shared doc` };
    }
    if (doc.custom_scope !== null && doc.custom_scope !== undefined && doc.custom_scope !== "") {
      return { allowed: false, reason: `custom_scope must be null for shared doc` };
    }

    // 5. Path must match relative path
    const normalizedPath = doc.path.replace(/\\/g, "/");
    if (!normalizedPath.endsWith(item.relative_path)) {
      return {
        allowed: false,
        reason: `Path mismatch: registered ${doc.path} does not match expected relative path ${item.relative_path}`,
      };
    }

    return { allowed: true, item };
  }
}
