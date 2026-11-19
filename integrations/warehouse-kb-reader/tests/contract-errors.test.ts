import test from "node:test";
import assert from "node:assert/strict";
import { ERROR_CODES, formatMcpErrorContent, createErrorEnvelope } from "../src/contracts/error-codes.js";
import { ErrorEnvelopeSchema } from "../src/contracts/tool-schemas.js";

test("Contract errors - all error codes are valid and strictly schema-checked", () => {
  const codes = [
    ERROR_CODES.UNAUTHORIZED,
    ERROR_CODES.DOC_NOT_ALLOWED,
    ERROR_CODES.SOURCE_UNAVAILABLE,
    ERROR_CODES.SOURCE_NOT_READY,
    ERROR_CODES.SOURCE_CHANGED,
    ERROR_CODES.SOURCE_TOO_LARGE,
    ERROR_CODES.INVALID_INPUT,
  ];

  for (const code of codes) {
    const envelope = createErrorEnvelope(code, `Test message for ${code}`);
    assert.doesNotThrow(() => ErrorEnvelopeSchema.parse(envelope));
    assert.equal(envelope.ok, false);
    assert.equal(envelope.code, code);

    // MCP text content format
    const formatted = formatMcpErrorContent(code, `Formatted error for ${code}`);
    const parsed = JSON.parse(formatted);
    assert.equal(parsed.ok, false);
    assert.equal(parsed.code, code);
  }
});

test("Contract errors - ensures error envelope has no leakage of internal host paths or secrets", () => {
  const msg = "Tài liệu doc-123 không được phép truy cập hoặc đã bị thu hồi quyền.";
  const envelope = createErrorEnvelope(ERROR_CODES.DOC_NOT_ALLOWED, msg);

  const serialized = JSON.stringify(envelope);
  // Ensure no host path or sensitive env leaks
  assert.ok(!serialized.includes("/app/workspace"));
  assert.ok(!serialized.includes("/var/lib/docker"));
  assert.ok(!serialized.includes("postgres"));
});
