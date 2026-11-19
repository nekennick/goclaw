export interface VaultDocMetadata {
  id: string;
  tenant_id: string;
  scope: string;
  agent_id: string | null;
  team_id: string | null;
  chat_id?: string | null;
  custom_scope?: string | null;
  path: string;
  title: string;
  doc_type: string;
  content_hash: string;
  updated_at: string;
}

export interface VaultListResponse {
  documents: VaultDocMetadata[];
  total: number;
}

export type FetchFn = typeof globalThis.fetch;

export class GoclawVaultClient {
  constructor(
    private readonly origin: string,
    private readonly apiKey: string,
    private readonly timeoutMs: number = 5000,
    private readonly customFetch: FetchFn = globalThis.fetch
  ) {}

  /**
   * Fetches all registered vault documents by paging through GET /v1/vault/documents
   */
  async listAllDocuments(): Promise<VaultDocMetadata[]> {
    const allDocs: VaultDocMetadata[] = [];
    const limit = 50;
    let offset = 0;
    let hasMore = true;

    while (hasMore) {
      const url = new URL("/v1/vault/documents", this.origin);
      url.searchParams.set("limit", String(limit));
      url.searchParams.set("offset", String(offset));

      const res = await this.doGet(url.toString());
      if (!res.ok) {
        throw new Error(`GoClaw GET /v1/vault/documents failed with status ${res.status}`);
      }

      const body = (await res.json()) as VaultListResponse;
      if (!body || !Array.isArray(body.documents)) {
        throw new Error("Invalid response format from GoClaw /v1/vault/documents");
      }

      allDocs.push(...body.documents);
      offset += body.documents.length;

      if (body.documents.length === 0 || allDocs.length >= body.total) {
        hasMore = false;
      }
    }

    return allDocs;
  }

  /**
   * Fetches a single document's metadata by docID
   */
  async getDocument(docId: string): Promise<VaultDocMetadata | null> {
    const encodedId = encodeURIComponent(docId);
    const url = new URL(`/v1/vault/documents/${encodedId}`, this.origin);

    const res = await this.doGet(url.toString());
    if (res.status === 404) {
      return null;
    }
    if (!res.ok) {
      throw new Error(`GoClaw GET /v1/vault/documents/${docId} failed with status ${res.status}`);
    }

    const doc = (await res.json()) as VaultDocMetadata;
    return doc;
  }

  private async doGet(urlString: string): Promise<Response> {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), this.timeoutMs);

    try {
      const headers: Record<string, string> = {
        Accept: "application/json",
      };
      if (this.apiKey) {
        headers["Authorization"] = `Bearer ${this.apiKey}`;
      }

      return await this.customFetch(urlString, {
        method: "GET",
        headers,
        signal: controller.signal,
        redirect: "error", // Strictly disallow redirects (prevent SSRF)
      });
    } finally {
      clearTimeout(timeout);
    }
  }
}
