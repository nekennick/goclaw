import { createServer, type IncomingMessage, type ServerResponse, type Server } from "node:http";
import { timingSafeEqual } from "node:crypto";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import type { ServiceConfig } from "./config.js";

export function createHttpServer(
  mcpServerOrFactory: McpServer | (() => McpServer),
  config: ServiceConfig
): Server {
  const getMcpServer = typeof mcpServerOrFactory === "function" ? mcpServerOrFactory : () => mcpServerOrFactory;

  const server = createServer(async (req: IncomingMessage, res: ServerResponse) => {
    const url = new URL(req.url || "/", `http://${req.headers.host || "localhost"}`);

    // Health check endpoint
    if (url.pathname === "/health") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ ok: true, version: "1.0.0", name: "warehouse-kb-reader" }));
      return;
    }

    // MCP endpoint
    if (url.pathname === "/mcp" || url.pathname.startsWith("/mcp/")) {
      // 1. Auth check (Bearer token)
      if (config.adapterBearerToken) {
        const authHeader = req.headers["authorization"] || "";
        const expectedHeader = `Bearer ${config.adapterBearerToken}`;

        if (
          !authHeader ||
          authHeader.length !== expectedHeader.length ||
          !timingSafeEqual(Buffer.from(authHeader), Buffer.from(expectedHeader))
        ) {
          res.writeHead(401, { "Content-Type": "application/json" });
          res.end(JSON.stringify({ error: "Unauthorized: Invalid or missing Bearer token" }));
          return;
        }
      }

      // 2. Origin check: Block foreign browser origins
      const origin = req.headers["origin"];
      if (origin && origin !== "null") {
        const host = req.headers["host"] || "";
        if (!origin.includes(host) && !origin.includes("localhost") && !origin.includes("127.0.0.1")) {
          res.writeHead(403, { "Content-Type": "application/json" });
          res.end(JSON.stringify({ error: "Forbidden: Cross-origin browser requests not allowed" }));
          return;
        }
      }

      // 3. Stateless transport per request:
      // Completely eliminates session expiration, stale session IDs, and 'Server not initialized' bugs.
      const transport = new StreamableHTTPServerTransport({
        sessionIdGenerator: undefined,
      });
      const mcpServer = getMcpServer();

      try {
        await mcpServer.connect(transport);
        await transport.handleRequest(req, res);
      } catch (err) {
        console.error("[warehouse-kb-reader] MCP handleRequest error:", err);
        if (!res.headersSent) {
          res.writeHead(500, { "Content-Type": "application/json" });
          res.end(JSON.stringify({ error: `Internal server error: ${err instanceof Error ? err.message : String(err)}` }));
        }
      }
      return;
    }

    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ error: "Not found" }));
  });

  return server;
}
