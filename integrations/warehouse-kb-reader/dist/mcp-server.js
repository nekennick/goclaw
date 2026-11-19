import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { LIMIT_MAX, QUERY_MAX_LENGTH } from "./contracts/tool-schemas.js";
export function createWarehouseKbMcpServer(service) {
    const server = new McpServer({
        name: "warehouse-kb-reader",
        version: "1.0.0",
    });
    // Tool 1: warehouse_kb_search
    server.tool("warehouse_kb_search", "Tra cứu nội dung quy trình vận hành kho Cao Lãnh và sơ đồ chi nhánh Yagami hiện hành", {
        query: z
            .string()
            .min(1, "query must not be empty")
            .max(QUERY_MAX_LENGTH, `query must not exceed ${QUERY_MAX_LENGTH} characters`)
            .describe("Từ khóa hoặc cụm từ tìm kiếm trong quy trình kho"),
        limit: z
            .number()
            .int()
            .min(1)
            .max(LIMIT_MAX)
            .optional()
            .describe("Số lượng kết quả tối đa cần lấy (mặc định 5, tối đa 10)"),
    }, async ({ query, limit }) => {
        const result = await service.search(query, limit ?? 5);
        return {
            isError: !result.ok,
            content: [
                {
                    type: "text",
                    text: JSON.stringify(result),
                },
            ],
        };
    });
    // Tool 2: warehouse_kb_read
    server.tool("warehouse_kb_read", "Đọc toàn văn hoặc phân trang tài liệu quy trình kho bằng doc_id và expected_hash nhận từ search", {
        doc_id: z.string().min(1).describe("ID tài liệu cần đọc (lấy từ kết quả warehouse_kb_search)"),
        expected_hash: z
            .string()
            .regex(/^[a-f0-9]{64}$/i, "expected_hash must be a 64-char hex SHA-256 string")
            .describe("Mã SHA-256 hash của tài liệu nhận từ warehouse_kb_search"),
        cursor: z.string().optional().describe("Con trỏ để đọc trang tiếp theo nếu has_more là true"),
    }, async ({ doc_id, expected_hash, cursor }) => {
        const result = await service.read(doc_id, expected_hash, cursor);
        return {
            isError: !result.ok,
            content: [
                {
                    type: "text",
                    text: JSON.stringify(result),
                },
            ],
        };
    });
    return server;
}
