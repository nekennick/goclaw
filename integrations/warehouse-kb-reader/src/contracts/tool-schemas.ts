import { z } from "zod";
import { CONTRACT_VERSION } from "./types.js";
import { ERROR_CODES } from "./error-codes.js";

export const LIMIT_DEFAULT = 5;
export const LIMIT_MAX = 10;
export const QUERY_MAX_LENGTH = 500;
export const SNIPPET_MAX_LENGTH = 1000;
export const PAGE_TEXT_MAX_LENGTH = 12000;

/**
 * Strict schema for warehouse_kb_search tool arguments.
 * Rejects any extra fields (path, sql, shell, tenant, scope, headers).
 */
export const SearchArgsSchema = z
  .object({
    query: z.string().trim().min(1, "query must not be empty").max(QUERY_MAX_LENGTH),
    limit: z.coerce
      .number()
      .int("limit must be integer")
      .min(1, "limit must be at least 1")
      .max(LIMIT_MAX, `limit must not exceed ${LIMIT_MAX}`)
      .optional()
      .default(LIMIT_DEFAULT),
  })
  .strict();

export type SearchArgs = z.infer<typeof SearchArgsSchema>;

/**
 * Strict schema for warehouse_kb_read tool arguments.
 * Rejects any extra fields. expected_hash is mandatory.
 */
export const ReadArgsSchema = z
  .object({
    doc_id: z.string().trim().min(1, "doc_id is required"),
    expected_hash: z
      .string()
      .trim()
      .regex(/^[a-f0-9]{64}$/i, "expected_hash must be a 64-char hex SHA-256 string"),
    cursor: z.string().trim().min(1).optional(),
  })
  .strict();

export type ReadArgs = z.infer<typeof ReadArgsSchema>;

/**
 * Zod schema validating successful search response envelope.
 */
export const SearchSuccessSchema = z
  .object({
    ok: z.literal(true),
    contract_version: z.literal(CONTRACT_VERSION),
    source_checked_at: z.string().datetime({ offset: true }),
    results: z.array(
      z.object({
        doc_id: z.string().min(1),
        title: z.string(),
        logical_path: z.string().min(1),
        content_hash: z.string().regex(/^[a-f0-9]{64}$/i),
        updated_at: z.string(),
        line_start: z.number().int().min(1),
        line_end: z.number().int().min(1),
        snippet: z.string().max(SNIPPET_MAX_LENGTH + 100),
      })
    ),
  })
  .strict();

/**
 * Zod schema validating successful read response envelope.
 */
export const ReadSuccessSchema = z
  .object({
    ok: z.literal(true),
    contract_version: z.literal(CONTRACT_VERSION),
    source_checked_at: z.string().datetime({ offset: true }),
    doc_id: z.string().min(1),
    title: z.string(),
    logical_path: z.string().min(1),
    content_hash: z.string().regex(/^[a-f0-9]{64}$/i),
    updated_at: z.string(),
    line_start: z.number().int().min(1),
    line_end: z.number().int().min(1),
    text: z.string().max(PAGE_TEXT_MAX_LENGTH + 500),
    has_more: z.boolean(),
    next_cursor: z.string().nullable(),
  })
  .strict();

/**
 * Error envelope validation.
 */
export const ErrorEnvelopeSchema = z
  .object({
    ok: z.literal(false),
    code: z.enum([
      ERROR_CODES.UNAUTHORIZED,
      ERROR_CODES.DOC_NOT_ALLOWED,
      ERROR_CODES.SOURCE_UNAVAILABLE,
      ERROR_CODES.SOURCE_NOT_READY,
      ERROR_CODES.SOURCE_CHANGED,
      ERROR_CODES.SOURCE_TOO_LARGE,
      ERROR_CODES.INVALID_INPUT,
    ]),
    message: z.string().min(1),
  })
  .strict();
