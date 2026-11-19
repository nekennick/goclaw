import test from "node:test";
import assert from "node:assert/strict";
import { PolicyEngine } from "../src/policy/policy-engine.js";
import {
  VALID_TENANT_ID,
  FIXTURE_VERIFIED_VAULT_DOCS,
  FIXTURE_REVOKED_DOC,
  FIXTURE_FOREIGN_TENANT_DOC,
} from "./fixtures/goclaw-vault-responses.js";

const engine = new PolicyEngine(VALID_TENANT_ID);

test("PolicyEngine - approves verified shared documents from allowlist", () => {
  for (const doc of FIXTURE_VERIFIED_VAULT_DOCS) {
    const result = engine.verify(doc);
    assert.equal(result.allowed, true);
    if (result.allowed) {
      assert.equal(result.item.doc_id, doc.id);
      assert.equal(result.item.expected_scope, "shared");
    }
  }
});

test("PolicyEngine - rejects unapproved / foreign document IDs", () => {
  const fakeDoc = {
    id: "01999999-0000-0000-0000-000000000000",
    tenant_id: VALID_TENANT_ID,
    scope: "shared",
    agent_id: null,
    team_id: null,
    path: "quan-ly-kim/zlo-ql-kim/group_zlo-ql-kim_6244108299423959453/data/warehouse-knowledge-base/hack.md",
    title: "Hack",
    content_hash: "71604f0fc3d42635ec9d4c8545813ce70e08e6ab4fb0a91ef5e016dd0493f99e",
    updated_at: "2026-10-31T13:55:10Z",
  };

  const result = engine.verify(fakeDoc);
  assert.equal(result.allowed, false);
});

test("PolicyEngine - rejects foreign tenant document (cross-tenant boundary enforcement)", () => {
  const result = engine.verify(FIXTURE_FOREIGN_TENANT_DOC);
  assert.equal(result.allowed, false);
});

test("PolicyEngine - rejects revoked documents (scope changed to personal)", () => {
  const result = engine.verify(FIXTURE_REVOKED_DOC);
  assert.equal(result.allowed, false);
});

test("PolicyEngine - rejects shared doc with attached team_id or agent_id", () => {
  const sample = FIXTURE_VERIFIED_VAULT_DOCS[0];

  const withTeam = { ...sample, team_id: "0193a5b0-1111-2222-3333-444455556666" };
  assert.equal(engine.verify(withTeam).allowed, false);

  const withAgent = { ...sample, agent_id: "0193a5b0-1111-2222-3333-444455556666" };
  assert.equal(engine.verify(withAgent).allowed, false);

  const withChat = { ...sample, chat_id: "chat-123" };
  assert.equal(engine.verify(withChat).allowed, false);
});

test("PolicyEngine - rejects document if registered path does not match relative path", () => {
  const sample = FIXTURE_VERIFIED_VAULT_DOCS[0];
  const tamperedPath = { ...sample, path: "wrong-folder/another-file.md" };
  assert.equal(engine.verify(tamperedPath).allowed, false);
});
