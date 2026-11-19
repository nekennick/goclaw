import test from "node:test";
import assert from "node:assert/strict";
import { promises as fs } from "node:fs";
import path from "node:path";
import os from "node:os";
import { WarehouseKbService } from "../src/service/warehouse-kb-service.js";
import { PolicyEngine } from "../src/policy/policy-engine.js";
import { SafeFileReader } from "../src/storage/safe-file-reader.js";
import { CursorCodec } from "../src/cursor/cursor-codec.js";
import { GoclawVaultClient } from "../src/upstream/goclaw-vault-client.js";
import {
  VALID_TENANT_ID,
  FIXTURE_VERIFIED_VAULT_DOCS,
  type VaultDocMetadata,
} from "./fixtures/goclaw-vault-responses.js";
import {
  SAMPLE_DOC_00_CONTENT,
  SAMPLE_DOC_00_HASH,
  generateLargeDocContent,
} from "./fixtures/mock-source-files.js";
import { ERROR_CODES } from "../src/contracts/error-codes.js";

async function setupTestEnvironment() {
  const tempDir = await fs.mkdtemp(path.join(os.tmpdir(), "service-test-"));
  const warehouseDir = path.join(tempDir, "warehouse-knowledge-base");
  const yagamiDir = path.join(tempDir, "yagami-knowledge-base");
  await fs.mkdir(warehouseDir, { recursive: true });
  await fs.mkdir(yagamiDir, { recursive: true });

  // Write sample 00 file
  const doc00Path = path.join(warehouseDir, "00-TONG-QUAN-NHAN-SU-VA-NOI-QUY.md");
  await fs.writeFile(doc00Path, SAMPLE_DOC_00_CONTENT, "utf-8");

  // Write large file for pagination
  const large = generateLargeDocContent(300);
  const largePath = path.join(warehouseDir, "01-TIMELINE-NHIP-VAN-HANH-HANG-NGAY.md");
  await fs.writeFile(largePath, large.content, "utf-8");

  // Write dummy files for remaining fixtures to avoid missing file errors
  await fs.writeFile(path.join(warehouseDir, "02-DONG-GOI-BAO-QUAN-VA-4-MA-SIET-TON.md"), "dummy", "utf-8");
  await fs.writeFile(path.join(yagamiDir, "KNOWLEDGE_MANIFEST.json"), "{}", "utf-8");
  await fs.writeFile(path.join(yagamiDir, "yagami-branches.csv"), "col1,col2", "utf-8");

  return {
    tempDir,
    warehouseDir,
    yagamiDir,
    largeHash: large.hash,
    cleanup: async () => {
      await fs.rm(tempDir, { recursive: true, force: true });
    },
  };
}

function createMockVaultClient(docs: VaultDocMetadata[]) {
  const map = new Map(docs.map((d) => [d.id, d]));
  return new GoclawVaultClient("http://mock-origin", "mock-token", 1000, async (url) => {
    const urlObj = new URL(url.toString());
    if (urlObj.pathname === "/v1/vault/documents") {
      return new Response(JSON.stringify({ documents: docs, total: docs.length }), { status: 200 });
    }
    const match = urlObj.pathname.match(/\/v1\/vault\/documents\/(.+)/);
    if (match) {
      const doc = map.get(decodeURIComponent(match[1]));
      if (doc) {
        return new Response(JSON.stringify(doc), { status: 200 });
      }
      return new Response(JSON.stringify({ error: "not found" }), { status: 404 });
    }
    return new Response(JSON.stringify({ error: "unknown route" }), { status: 404 });
  });
}

test("WarehouseKbService - search: returns verified results and accurate snippets", async () => {
  const env = await setupTestEnvironment();

  const vaultDocs = [
    {
      ...FIXTURE_VERIFIED_VAULT_DOCS[0],
      content_hash: SAMPLE_DOC_00_HASH,
    },
  ];

  const policyEngine = new PolicyEngine(VALID_TENANT_ID);
  const fileReader = new SafeFileReader({
    warehouseKbRoot: env.warehouseDir,
    yagamiKbRoot: env.yagamiDir,
  });
  const vaultClient = createMockVaultClient(vaultDocs);
  const cursorCodec = new CursorCodec("test-secret");

  const service = new WarehouseKbService({
    policyEngine,
    fileReader,
    vaultClient,
    cursorCodec,
  });

  const res = await service.search("kiểm kê");
  assert.equal(res.ok, true);
  if (res.ok) {
    assert.equal(res.contract_version, 1);
    assert.ok(res.results.length >= 1);
    assert.equal(res.results[0].doc_id, FIXTURE_VERIFIED_VAULT_DOCS[0].id);
    assert.ok(res.results[0].snippet.includes("Thứ 4"));
  }

  await env.cleanup();
});

test("WarehouseKbService - search: returns SOURCE_NOT_READY if physical file hash diverges from Vault registry", async () => {
  const env = await setupTestEnvironment();

  // Vault registered an old hash, but physical file on disk has changed
  const vaultDocs = [
    {
      ...FIXTURE_VERIFIED_VAULT_DOCS[0],
      content_hash: "0000000000000000000000000000000000000000000000000000000000000000",
    },
  ];

  const service = new WarehouseKbService({
    policyEngine: new PolicyEngine(VALID_TENANT_ID),
    fileReader: new SafeFileReader({
      warehouseKbRoot: env.warehouseDir,
      yagamiKbRoot: env.yagamiDir,
    }),
    vaultClient: createMockVaultClient(vaultDocs),
    cursorCodec: new CursorCodec("test-secret"),
  });

  const res = await service.search("kiểm kê");
  assert.equal(res.ok, false);
  if (!res.ok) {
    assert.equal(res.code, ERROR_CODES.SOURCE_NOT_READY);
  }

  await env.cleanup();
});

test("WarehouseKbService - read: succeeds and handles pagination smoothly", async () => {
  const env = await setupTestEnvironment();

  const docId = FIXTURE_VERIFIED_VAULT_DOCS[1].id;
  const vaultDocs = [
    {
      ...FIXTURE_VERIFIED_VAULT_DOCS[1],
      content_hash: env.largeHash,
    },
  ];

  const service = new WarehouseKbService({
    policyEngine: new PolicyEngine(VALID_TENANT_ID),
    fileReader: new SafeFileReader({
      warehouseKbRoot: env.warehouseDir,
      yagamiKbRoot: env.yagamiDir,
    }),
    vaultClient: createMockVaultClient(vaultDocs),
    cursorCodec: new CursorCodec("test-secret"),
  });

  // Read page 1
  const page1 = await service.read(docId, env.largeHash);
  assert.equal(page1.ok, true);
  if (page1.ok) {
    assert.equal(page1.doc_id, docId);
    assert.equal(page1.line_start, 1);
    assert.equal(page1.has_more, true);
    assert.ok(page1.next_cursor);

    // Read page 2 using cursor
    const page2 = await service.read(docId, env.largeHash, page1.next_cursor);
    assert.equal(page2.ok, true);
    if (page2.ok) {
      assert.equal(page2.line_start, page1.line_end + 1);
    }
  }

  await env.cleanup();
});

test("WarehouseKbService - read: returns SOURCE_CHANGED if expected_hash does not match current source", async () => {
  const env = await setupTestEnvironment();

  const docId = FIXTURE_VERIFIED_VAULT_DOCS[0].id;
  const vaultDocs = [
    {
      ...FIXTURE_VERIFIED_VAULT_DOCS[0],
      content_hash: SAMPLE_DOC_00_HASH,
    },
  ];

  const service = new WarehouseKbService({
    policyEngine: new PolicyEngine(VALID_TENANT_ID),
    fileReader: new SafeFileReader({
      warehouseKbRoot: env.warehouseDir,
      yagamiKbRoot: env.yagamiDir,
    }),
    vaultClient: createMockVaultClient(vaultDocs),
    cursorCodec: new CursorCodec("test-secret"),
  });

  // Client requests with stale hash
  const res = await service.read(docId, "1111111111111111111111111111111111111111111111111111111111111111");
  assert.equal(res.ok, false);
  if (!res.ok) {
    assert.equal(res.code, ERROR_CODES.SOURCE_CHANGED);
  }

  await env.cleanup();
});
