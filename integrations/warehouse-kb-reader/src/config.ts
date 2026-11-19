export interface ServiceConfig {
  port: number;
  goclawOrigin: string;
  goclawApiKey: string;
  adapterBearerToken: string;
  cursorSecret: string;
  tenantId: string;
  warehouseKbRoot: string;
  yagamiKbRoot: string;
  upstreamTimeoutMs: number;
}

export function loadConfig(env: NodeJS.ProcessEnv = process.env): ServiceConfig {
  const port = parseInt(env.PORT || "8080", 10);
  const goclawOrigin = env.GOCLAW_ORIGIN || "http://host.docker.internal:18790";
  const goclawApiKey = env.GOCLAW_API_KEY || "";
  const adapterBearerToken = env.ADAPTER_BEARER_TOKEN || "";
  const cursorSecret = env.CURSOR_SECRET || "default-warehouse-cursor-secret-key-change-in-prod";
  const tenantId = env.TENANT_ID || "0193a5b0-7000-7000-8000-000000000001";
  const warehouseKbRoot = env.WAREHOUSE_KB_ROOT || "/sources/warehouse-knowledge-base";
  const yagamiKbRoot = env.YAGAMI_KB_ROOT || "/sources/yagami-knowledge-base";
  const upstreamTimeoutMs = parseInt(env.UPSTREAM_TIMEOUT_MS || "5000", 10);

  return {
    port,
    goclawOrigin,
    goclawApiKey,
    adapterBearerToken,
    cursorSecret,
    tenantId,
    warehouseKbRoot,
    yagamiKbRoot,
    upstreamTimeoutMs,
  };
}
