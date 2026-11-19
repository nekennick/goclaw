import test from "node:test";
import assert from "node:assert/strict";
import { promises as fs } from "node:fs";
import path from "node:path";
import os from "node:os";
import { SafeFileReader } from "../src/storage/safe-file-reader.js";
import { SAMPLE_DOC_00_CONTENT, SAMPLE_DOC_00_HASH } from "./fixtures/mock-source-files.js";

test("SafeFileReader - reads regular UTF-8 files and calculates raw SHA-256", async () => {
  const tempDir = await fs.mkdtemp(path.join(os.tmpdir(), "kb-reader-test-"));
  const warehouseDir = path.join(tempDir, "warehouse");
  const yagamiDir = path.join(tempDir, "yagami");
  await fs.mkdir(warehouseDir);
  await fs.mkdir(yagamiDir);

  const testFile = path.join(warehouseDir, "00-TONG-QUAN-NHAN-SU-VA-NOI-QUY.md");
  await fs.writeFile(testFile, SAMPLE_DOC_00_CONTENT, "utf-8");

  const reader = new SafeFileReader({
    warehouseKbRoot: warehouseDir,
    yagamiKbRoot: yagamiDir,
  });

  const res = await reader.readVerifiedFile(
    "warehouse-knowledge-base",
    "00-TONG-QUAN-NHAN-SU-VA-NOI-QUY.md"
  );

  assert.equal(res.contentHash, SAMPLE_DOC_00_HASH);
  assert.equal(res.text, SAMPLE_DOC_00_CONTENT);
  assert.ok(res.lineCount > 5);

  // Clean up
  await fs.rm(tempDir, { recursive: true, force: true });
});

test("SafeFileReader - security: strictly blocks path traversal and forbidden extensions", async () => {
  const tempDir = await fs.mkdtemp(path.join(os.tmpdir(), "kb-reader-sec-"));
  const warehouseDir = path.join(tempDir, "warehouse");
  await fs.mkdir(warehouseDir);

  const reader = new SafeFileReader({
    warehouseKbRoot: warehouseDir,
    yagamiKbRoot: tempDir,
  });

  // Path traversal
  await assert.rejects(
    () => reader.readVerifiedFile("warehouse-knowledge-base", "../secret.txt"),
    /traversal|escapes/i
  );

  // Absolute path
  await assert.rejects(
    () => reader.readVerifiedFile("warehouse-knowledge-base", "/etc/passwd"),
    /absolute/i
  );

  // Disallowed extension
  await assert.rejects(
    () => reader.readVerifiedFile("warehouse-knowledge-base", "malicious.sh"),
    /unsupported extension/i
  );

  await fs.rm(tempDir, { recursive: true, force: true });
});
