# Yagami Knowledge Base

This folder is the portable knowledge pack for Yagami Dashboard agents.

## How To Use

Copy the whole `docs/knowledge` folder to another agent or project. Ask the agent to read this file first, then read `KNOWLEDGE_MANIFEST.json`.

The canonical source of truth is:

- `yagami-branch-warehouse-map.json`

Convenience exports are generated from that JSON:

- `yagami-branches.jsonl`: one branch per line, best for RAG/vector indexing.
- `yagami-branches.csv`: spreadsheet-friendly branch table.
- `yagami-branch-warehouse-map.md`: human-readable summary.

## Import Instructions For AI Agents

1. Read `KNOWLEDGE_MANIFEST.json`.
2. Treat `yagami-branch-warehouse-map.json` as the authoritative file.
3. Use `customerCode` as the strongest join key when matching KiotViet data.
4. Use `canonicalName` only as a display name or fallback matching key.
5. Respect `status`; do not count inactive/deleted entries as active branches.
6. Respect `warehouse`; valid values are `CAO_LANH` and `BINH_DUONG`.
7. When answering analysis questions, cite the exact file version and `verifiedAt` date.

## Data Contract

Each branch record has:

| Field | Meaning |
| --- | --- |
| `warehouse` | Owning warehouse, usually `CAO_LANH` or `BINH_DUONG`. |
| `day` | Delivery/route day from the source route sheet, if known. |
| `sourceCell` | Original workbook cell, if available. |
| `rawName` | Raw route/customer label from source data. |
| `canonicalName` | Clean branch/customer name for reporting. |
| `customerCode` | KiotViet customer code. Use this for joins. |
| `status` | `ACTIVE`, `PLANNED`, `DELETED`, or other explicit state. |
| `routeType` | Route classification such as `SCHEDULED_TRUCK` or `EXTERNAL_VEHICLE`. |
| `notes` | Manual confirmation, alias, or caveat. |

## Sharing Rule

Do not put secrets, API keys, database files, or `.env` values in this folder. This folder is meant to be safe to copy and share with other AI agents.
