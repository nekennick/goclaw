import { createHmac, timingSafeEqual } from "node:crypto";

export interface CursorPayload {
  doc_id: string;
  expected_hash: string;
  line_offset: number; // 1-based start line of next page
}

export class CursorCodec {
  private readonly secret: Buffer;

  constructor(secretKey: string) {
    this.secret = Buffer.from(secretKey, "utf-8");
  }

  encode(payload: CursorPayload): string {
    const data = JSON.stringify({
      d: payload.doc_id,
      h: payload.expected_hash,
      l: payload.line_offset,
    });
    const dataB64 = Buffer.from(data, "utf-8").toString("base64url");
    const sig = createHmac("sha256", this.secret).update(dataB64).digest("base64url");
    return `${dataB64}.${sig}`;
  }

  decode(cursor: string, expectedDocId: string, expectedHash: string): CursorPayload | null {
    if (!cursor || typeof cursor !== "string") return null;
    const parts = cursor.split(".");
    if (parts.length !== 2) return null;

    const [dataB64, providedSig] = parts;
    const expectedSig = createHmac("sha256", this.secret).update(dataB64).digest("base64url");

    if (providedSig.length !== expectedSig.length) return null;
    const sigMatch = timingSafeEqual(
      Buffer.from(providedSig, "utf-8"),
      Buffer.from(expectedSig, "utf-8")
    );
    if (!sigMatch) return null;

    try {
      const json = JSON.parse(Buffer.from(dataB64, "base64url").toString("utf-8"));
      if (
        typeof json.d !== "string" ||
        typeof json.h !== "string" ||
        typeof json.l !== "number" ||
        !Number.isInteger(json.l) ||
        json.l < 1
      ) {
        return null;
      }

      // Bound strictly to doc_id and expected_hash
      if (json.d !== expectedDocId || json.h.toLowerCase() !== expectedHash.toLowerCase()) {
        return null;
      }

      return {
        doc_id: json.d,
        expected_hash: json.h,
        line_offset: json.l,
      };
    } catch {
      return null;
    }
  }
}
