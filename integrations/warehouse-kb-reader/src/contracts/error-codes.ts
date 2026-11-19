/**
 * Error codes defined by the warehouse-kb-reader contract.
 * See: docs/warehouse-kb-readonly-handoff.md § 5.2
 */
export const ERROR_CODES = {
  UNAUTHORIZED: "UNAUTHORIZED",
  DOC_NOT_ALLOWED: "DOC_NOT_ALLOWED",
  SOURCE_UNAVAILABLE: "SOURCE_UNAVAILABLE",
  SOURCE_NOT_READY: "SOURCE_NOT_READY",
  SOURCE_CHANGED: "SOURCE_CHANGED",
  SOURCE_TOO_LARGE: "SOURCE_TOO_LARGE",
  INVALID_INPUT: "INVALID_INPUT",
} as const;

export type ErrorCode = (typeof ERROR_CODES)[keyof typeof ERROR_CODES];

export interface ErrorEnvelope {
  ok: false;
  code: ErrorCode;
  message: string;
}

export function createErrorEnvelope(code: ErrorCode, message: string): ErrorEnvelope {
  return { ok: false, code, message };
}

export function formatMcpErrorContent(code: ErrorCode, message: string): string {
  return JSON.stringify(createErrorEnvelope(code, message));
}
