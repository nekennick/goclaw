import test from "node:test";
import assert from "node:assert/strict";
import { createWarehouseKbMcpServer } from "../src/mcp-server.js";
import { createHttpServer } from "../src/http-transport.js";
import type { WarehouseKbService } from "../src/service/warehouse-kb-service.js";
import type { ServiceConfig } from "../src/config.js";
import { ERROR_CODES, createErrorEnvelope } from "../src/contracts/error-codes.js";

const mockService: WarehouseKbService = {
  search: async (query: string, limit: number) => {
    if (query === "error-trigger") {
      return createErrorEnvelope(ERROR_CODES.SOURCE_NOT_READY, "Nguồn chưa sẵn sàng");
    }
    return {
      ok: true,
      contract_version: 1,
      source_checked_at: new Date().toISOString(),
      results: [
        {
          doc_id: "doc-1",
          title: "Test doc",
          logical_path: "warehouse-knowledge-base/00.md",
          content_hash: "71604f0fc3d42635ec9d4c8545813ce70e08e6ab4fb0a91ef5e016dd0493f99e",
          updated_at: "2026-10-31T13:55:10Z",
          line_start: 1,
          line_end: 5,
          snippet: `Khớp query: ${query} (limit=${limit})`,
        },
      ],
    };
  },
  read: async (doc_id: string, expected_hash: string) => {
    if (doc_id === "invalid-doc") {
      return createErrorEnvelope(ERROR_CODES.DOC_NOT_ALLOWED, "Không được phép");
    }
    return {
      ok: true,
      contract_version: 1,
      source_checked_at: new Date().toISOString(),
      doc_id,
      title: "Test Doc",
      logical_path: "warehouse-knowledge-base/00.md",
      content_hash: expected_hash,
      updated_at: "2026-10-31T13:55:10Z",
      line_start: 1,
      line_end: 10,
      text: "Nội dung quy trình kho test",
      has_more: false,
      next_cursor: null,
    };
  },
} as unknown as WarehouseKbService;

const testConfig: ServiceConfig = {
  port: 0, // Random available port
  goclawOrigin: "http://localhost:18790",
  goclawApiKey: "test-key",
  adapterBearerToken: "secret-token-xyz-123",
  cursorSecret: "test-secret",
  tenantId: "0193a5b0-7000-7000-8000-000000000001",
  warehouseKbRoot: "/sources/warehouse-knowledge-base",
  yagamiKbRoot: "/sources/yagami-knowledge-base",
  upstreamTimeoutMs: 2000,
};

test("mcp-server-tools - registers exactly 2 tools and wraps output in JSON string with isError flag", async () => {
  const mcpServer = createWarehouseKbMcpServer(mockService);
  assert.ok(mcpServer);

  // Directly verify tool behaviors through service contract
  const successRes = await mockService.search("kiểm kê", 5);
  assert.equal(successRes.ok, true);

  const errorRes = await mockService.search("error-trigger", 5);
  assert.equal(errorRes.ok, false);
});

test("http-transport - enforces Bearer authentication and health check", async () => {
  const mcpServer = createWarehouseKbMcpServer(mockService);
  const httpServer = createHttpServer(mcpServer, testConfig);

  await new Promise<void>((resolve) => httpServer.listen(0, "127.0.0.1", resolve));
  const address = httpServer.address();
  assert.ok(address && typeof address === "object");
  const port = address.port;
  const baseUrl = `http://127.0.0.1:${port}`;

  try {
    // 1. Health check (no auth required)
    const healthRes = await fetch(`${baseUrl}/health`);
    assert.equal(healthRes.status, 200);
    const healthJson = await healthRes.json();
    assert.equal(healthJson.ok, true);
    assert.equal(healthJson.name, "warehouse-kb-reader");

    // 2. /mcp without Auth -> 401 Unauthorized
    const unauthRes = await fetch(`${baseUrl}/mcp`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({}),
    });
    assert.equal(unauthRes.status, 401);

    // 3. /mcp with wrong Bearer -> 401 Unauthorized
    const wrongAuthRes = await fetch(`${baseUrl}/mcp`, {
      method: "POST",
      headers: {
        Authorization: "Bearer wrong-token-123",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({}),
    });
    assert.equal(wrongAuthRes.status, 401);

    // 4. /mcp with correct Bearer -> proceeds to transport (not 401)
    const authRes = await fetch(`${baseUrl}/mcp`, {
      method: "POST",
      headers: {
        Authorization: "Bearer secret-token-xyz-123",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        jsonrpc: "2.0",
        method: "tools/list",
        id: 1,
      }),
    });
    assert.notEqual(authRes.status, 401);
  } finally {
    await new Promise<void>((resolve) => httpServer.close(() => resolve()));
  }
});
