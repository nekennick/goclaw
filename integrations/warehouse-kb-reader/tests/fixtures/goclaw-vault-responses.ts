/**
 * GoClaw Vault GET API response fixtures based on actual PostgreSQL /v1/vault/documents contract.
 * Crucial rule: Vault GET responses DO NOT include full text content (E1, E2).
 */

import type { VaultDocMetadata, VaultListResponse } from "../../src/upstream/goclaw-vault-client.js";

export type { VaultDocMetadata, VaultListResponse };

export const VALID_TENANT_ID = "0193a5b0-7000-7000-8000-000000000001";

/**
 * Fixture representing verified 17 documents in Vault metadata.
 */
export const FIXTURE_VERIFIED_VAULT_DOCS: VaultDocMetadata[] = [
  {
    id: "01a1900e-ab0b-749c-9808-af69babace6c",
    tenant_id: VALID_TENANT_ID,
    scope: "shared",
    agent_id: null,
    team_id: null,
    chat_id: null,
    custom_scope: null,
    path: "quan-ly-kim/zlo-ql-kim/group_zlo-ql-kim_6244108299423959453/data/warehouse-knowledge-base/00-TONG-QUAN-NHAN-SU-VA-NOI-QUY.md",
    title: "00-TONG-QUAN-NHAN-SU-VA-NOI-QUY",
    doc_type: "note",
    content_hash: "71604f0fc3d42635ec9d4c8545813ce70e08e6ab4fb0a91ef5e016dd0493f99e",
    updated_at: "2026-10-31T13:55:10.270704Z",
  },
  {
    id: "01a1900e-ab0d-7aee-90cc-c66d35860036",
    tenant_id: VALID_TENANT_ID,
    scope: "shared",
    agent_id: null,
    team_id: null,
    chat_id: null,
    custom_scope: null,
    path: "quan-ly-kim/zlo-ql-kim/group_zlo-ql-kim_6244108299423959453/data/warehouse-knowledge-base/01-TIMELINE-NHIP-VAN-HANH-HANG-NGAY.md",
    title: "01-TIMELINE-NHIP-VAN-HANH-HANG-NGAY",
    doc_type: "note",
    content_hash: "7f2e8b74c0a012f7b3d9994a82097d45a3035fed0c7c312ffb8b4f51e83f4724",
    updated_at: "2026-10-31T13:55:37.046566Z",
  },
  {
    id: "01a1900e-ab10-7017-ba03-48b32b9774f0",
    tenant_id: VALID_TENANT_ID,
    scope: "shared",
    agent_id: null,
    team_id: null,
    chat_id: null,
    custom_scope: null,
    path: "quan-ly-kim/zlo-ql-kim/group_zlo-ql-kim_6244108299423959453/data/warehouse-knowledge-base/02-DONG-GOI-BAO-QUAN-VA-4-MA-SIET-TON.md",
    title: "02-DONG-GOI-BAO-QUAN-VA-4-MA-SIET-TON",
    doc_type: "note",
    content_hash: "10e836d649acc36150375b53a90dd50bce1ba99f027fed1dc720b144efc4df1c",
    updated_at: "2026-10-31T13:55:47.260081Z",
  },
  {
    id: "01a1900e-ab2c-7afe-81a8-c264fef62ba2",
    tenant_id: VALID_TENANT_ID,
    scope: "shared",
    agent_id: null,
    team_id: null,
    chat_id: null,
    custom_scope: null,
    path: "quan-ly-kim/zlo-ql-kim/group_zlo-ql-kim_6244108299423959453/data/yagami-knowledge-base/KNOWLEDGE_MANIFEST.json",
    title: "KNOWLEDGE_MANIFEST",
    doc_type: "note",
    content_hash: "1f72b1cede2539615bc844a9147ad058d0158abee7ef0b27ee77407442b4e41e",
    updated_at: "2026-10-31T13:57:11.217505Z",
  },
  {
    id: "01a1900e-ab36-7719-a26f-b7327f69bbea",
    tenant_id: VALID_TENANT_ID,
    scope: "shared",
    agent_id: null,
    team_id: null,
    chat_id: null,
    custom_scope: null,
    path: "quan-ly-kim/zlo-ql-kim/group_zlo-ql-kim_6244108299423959453/data/yagami-knowledge-base/yagami-branches.csv",
    title: "yagami-branches",
    doc_type: "note",
    content_hash: "7980ddf693c74bc0e40d1d8b1ae66a5fab3dddb4d53c3c1e06d37604c9dfc25f",
    updated_at: "2026-10-31T13:57:39.050734Z",
  },
];

/**
 * Fixture: Document with revoked permission (scope changed from shared to personal).
 */
export const FIXTURE_REVOKED_DOC: VaultDocMetadata = {
  id: "01a1900e-ab0b-749c-9808-af69babace6c",
  tenant_id: VALID_TENANT_ID,
  scope: "personal", // No longer shared!
  agent_id: "019ec947-8f55-70fe-8a81-11975e921399",
  team_id: null,
  path: "quan-ly-kim/zlo-ql-kim/group_zlo-ql-kim_6244108299423959453/data/warehouse-knowledge-base/00-TONG-QUAN-NHAN-SU-VA-NOI-QUY.md",
  title: "00-TONG-QUAN-NHAN-SU-VA-NOI-QUY",
  doc_type: "note",
  content_hash: "71604f0fc3d42635ec9d4c8545813ce70e08e6ab4fb0a91ef5e016dd0493f99e",
  updated_at: "2026-11-06T10:00:00.000000Z",
};

/**
 * Fixture: Document belonging to another tenant (must be rejected, no path leaked).
 */
export const FIXTURE_FOREIGN_TENANT_DOC: VaultDocMetadata = {
  id: "01999999-9999-9999-9999-999999999999",
  tenant_id: "01999999-0000-0000-0000-000000000002", // Foreign tenant
  scope: "shared",
  agent_id: null,
  team_id: null,
  path: "other-tenant/data/secret.md",
  title: "Secret",
  doc_type: "note",
  content_hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  updated_at: "2026-11-06T10:00:00.000000Z",
};
