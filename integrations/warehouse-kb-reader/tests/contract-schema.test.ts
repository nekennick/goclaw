import test from "node:test";
import assert from "node:assert/strict";
import {
  SearchArgsSchema,
  ReadArgsSchema,
  SearchSuccessSchema,
  ReadSuccessSchema,
  ErrorEnvelopeSchema,
  LIMIT_DEFAULT,
  LIMIT_MAX,
} from "../src/contracts/tool-schemas.js";
import { ERROR_CODES, formatMcpErrorContent } from "../src/contracts/error-codes.js";

test("SearchArgsSchema - accepts valid inputs and enforces defaults", () => {
  const parsed1 = SearchArgsSchema.parse({ query: "quy trình kiểm kho" });
  assert.equal(parsed1.query, "quy trình kiểm kho");
  assert.equal(parsed1.limit, LIMIT_DEFAULT);

  const parsed2 = SearchArgsSchema.parse({ query: "lịch xuất hóa đơn", limit: 3 });
  assert.equal(parsed2.query, "lịch xuất hóa đơn");
  assert.equal(parsed2.limit, 3);

  const parsedMax = SearchArgsSchema.parse({ query: "test", limit: LIMIT_MAX });
  assert.equal(parsedMax.limit, LIMIT_MAX);
});

test("SearchArgsSchema - strictly rejects extra fields (prevent parameter injection)", () => {
  // Model trying to pass path or tenant or sql
  assert.throws(
    () =>
      SearchArgsSchema.parse({
        query: "test",
        path: "/etc/passwd",
      }),
    /unrecognized_keys/i
  );

  assert.throws(
    () =>
      SearchArgsSchema.parse({
        query: "test",
        tenant_id: "0193a5b0-7000-7000-8000-000000000001",
      }),
    /unrecognized_keys/i
  );

  assert.throws(
    () =>
      SearchArgsSchema.parse({
        query: "test",
        sql: "SELECT * FROM users",
      }),
    /unrecognized_keys/i
  );
});

test("SearchArgsSchema - enforces query length and limit bounds", () => {
  // Empty query
  assert.throws(() => SearchArgsSchema.parse({ query: "" }), /query must not be empty/i);
  assert.throws(() => SearchArgsSchema.parse({ query: "   " }), /query must not be empty/i);

  // Negative limit
  assert.throws(() => SearchArgsSchema.parse({ query: "ok", limit: -1 }), /limit must be at least 1/i);
  assert.throws(() => SearchArgsSchema.parse({ query: "ok", limit: 0 }), /limit must be at least 1/i);

  // Exceeding max limit (10)
  assert.throws(() => SearchArgsSchema.parse({ query: "ok", limit: 11 }), /limit must not exceed 10/i);
  assert.throws(() => SearchArgsSchema.parse({ query: "ok", limit: 100 }), /limit must not exceed 10/i);
});

test("ReadArgsSchema - requires expected_hash and validates hex SHA-256", () => {
  const validHash = "71604f0fc3d42635ec9d4c8545813ce70e08e6ab4fb0a91ef5e016dd0493f99e";
  const parsed = ReadArgsSchema.parse({
    doc_id: "01a1900e-ab0b-749c-9808-af69babace6c",
    expected_hash: validHash,
  });
  assert.equal(parsed.doc_id, "01a1900e-ab0b-749c-9808-af69babace6c");
  assert.equal(parsed.expected_hash, validHash);

  // Missing expected_hash is strictly rejected
  assert.throws(
    () =>
      ReadArgsSchema.parse({
        doc_id: "01a1900e-ab0b-749c-9808-af69babace6c",
      }),
    /expected_hash/i
  );

  // Invalid hash length or characters
  assert.throws(
    () =>
      ReadArgsSchema.parse({
        doc_id: "01a1900e-ab0b-749c-9808-af69babace6c",
        expected_hash: "invalid-hash",
      }),
    /64-char hex/i
  );

  // Extra field rejected
  assert.throws(
    () =>
      ReadArgsSchema.parse({
        doc_id: "01a1900e-ab0b-749c-9808-af69babace6c",
        expected_hash: validHash,
        file_path: "/data/secret",
      }),
    /unrecognized_keys/i
  );
});

test("Envelopes validate contract version and structure", () => {
  const searchSuccess = {
    ok: true,
    contract_version: 1,
    source_checked_at: new Date().toISOString(),
    results: [
      {
        doc_id: "doc-1",
        title: "Quy trình kho",
        logical_path: "warehouse-knowledge-base/00-TONG-QUAN.md",
        content_hash: "71604f0fc3d42635ec9d4c8545813ce70e08e6ab4fb0a91ef5e016dd0493f99e",
        updated_at: "2026-10-31T13:55:10Z",
        line_start: 1,
        line_end: 10,
        snippet: "Nhân sự gồm Anh Khoa, Chị Phượng...",
      },
    ],
  };
  assert.doesNotThrow(() => SearchSuccessSchema.parse(searchSuccess));

  const readSuccess = {
    ok: true,
    contract_version: 1,
    source_checked_at: new Date().toISOString(),
    doc_id: "doc-1",
    title: "Quy trình kho",
    logical_path: "warehouse-knowledge-base/00-TONG-QUAN.md",
    content_hash: "71604f0fc3d42635ec9d4c8545813ce70e08e6ab4fb0a91ef5e016dd0493f99e",
    updated_at: "2026-10-31T13:55:10Z",
    line_start: 1,
    line_end: 15,
    text: "# Nội quy kho...",
    has_more: false,
    next_cursor: null,
  };
  assert.doesNotThrow(() => ReadSuccessSchema.parse(readSuccess));
});

test("Error envelopes and string formatting for MCP text content", () => {
  const errJson = formatMcpErrorContent(
    ERROR_CODES.SOURCE_NOT_READY,
    "Nguồn tài liệu đang cập nhật, chưa thể xác nhận tính hiện hành."
  );

  const parsed = JSON.parse(errJson);
  assert.equal(parsed.ok, false);
  assert.equal(parsed.code, "SOURCE_NOT_READY");
  assert.equal(typeof parsed.message, "string");

  assert.doesNotThrow(() => ErrorEnvelopeSchema.parse(parsed));
});
