import { loadConfig } from "./config.js";
import { PolicyEngine } from "./policy/policy-engine.js";
import { SafeFileReader } from "./storage/safe-file-reader.js";
import { GoclawVaultClient } from "./upstream/goclaw-vault-client.js";
import { CursorCodec } from "./cursor/cursor-codec.js";
import { WarehouseKbService } from "./service/warehouse-kb-service.js";
import { createWarehouseKbMcpServer } from "./mcp-server.js";
import { createHttpServer } from "./http-transport.js";
async function main() {
    const config = loadConfig();
    const policyEngine = new PolicyEngine(config.tenantId);
    const fileReader = new SafeFileReader({
        warehouseKbRoot: config.warehouseKbRoot,
        yagamiKbRoot: config.yagamiKbRoot,
    });
    const vaultClient = new GoclawVaultClient(config.goclawOrigin, config.goclawApiKey, config.upstreamTimeoutMs);
    const cursorCodec = new CursorCodec(config.cursorSecret);
    const service = new WarehouseKbService({
        policyEngine,
        fileReader,
        vaultClient,
        cursorCodec,
    });
    const httpServer = createHttpServer(() => createWarehouseKbMcpServer(service), config);
    httpServer.listen(config.port, "0.0.0.0", () => {
        console.log(`[warehouse-kb-reader] Listening on port ${config.port}`);
    });
    const shutdown = () => {
        console.log("[warehouse-kb-reader] Shutting down gracefully...");
        httpServer.close(() => {
            process.exit(0);
        });
    };
    process.on("SIGINT", shutdown);
    process.on("SIGTERM", shutdown);
}
main().catch((err) => {
    console.error("[warehouse-kb-reader] Fatal startup error:", err);
    process.exit(1);
});
