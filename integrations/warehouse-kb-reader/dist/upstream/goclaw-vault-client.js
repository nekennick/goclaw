export class GoclawVaultClient {
    origin;
    apiKey;
    timeoutMs;
    customFetch;
    constructor(origin, apiKey, timeoutMs = 5000, customFetch = globalThis.fetch) {
        this.origin = origin;
        this.apiKey = apiKey;
        this.timeoutMs = timeoutMs;
        this.customFetch = customFetch;
    }
    /**
     * Fetches all registered vault documents by paging through GET /v1/vault/documents
     */
    async listAllDocuments() {
        const allDocs = [];
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
            const body = (await res.json());
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
    async getDocument(docId) {
        const encodedId = encodeURIComponent(docId);
        const url = new URL(`/v1/vault/documents/${encodedId}`, this.origin);
        const res = await this.doGet(url.toString());
        if (res.status === 404) {
            return null;
        }
        if (!res.ok) {
            throw new Error(`GoClaw GET /v1/vault/documents/${docId} failed with status ${res.status}`);
        }
        const doc = (await res.json());
        return doc;
    }
    async doGet(urlString) {
        const controller = new AbortController();
        const timeout = setTimeout(() => controller.abort(), this.timeoutMs);
        try {
            const headers = {
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
        }
        finally {
            clearTimeout(timeout);
        }
    }
}
