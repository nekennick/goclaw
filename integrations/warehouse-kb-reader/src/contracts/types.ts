import type { ErrorEnvelope } from "./error-codes.js";

export const CONTRACT_VERSION = 1;

export interface SearchResultItem {
  doc_id: string;
  title: string;
  logical_path: string;
  content_hash: string;
  updated_at: string;
  line_start: number;
  line_end: number;
  snippet: string;
}

export interface SearchSuccessEnvelope {
  ok: true;
  contract_version: typeof CONTRACT_VERSION;
  source_checked_at: string; // ISO-8601 UTC
  results: SearchResultItem[];
}

export type SearchEnvelope = SearchSuccessEnvelope | ErrorEnvelope;

export interface ReadSuccessEnvelope {
  ok: true;
  contract_version: typeof CONTRACT_VERSION;
  source_checked_at: string; // ISO-8601 UTC
  doc_id: string;
  title: string;
  logical_path: string;
  content_hash: string;
  updated_at: string;
  line_start: number;
  line_end: number;
  text: string;
  has_more: boolean;
  next_cursor: string | null;
}

export type ReadEnvelope = ReadSuccessEnvelope | ErrorEnvelope;
