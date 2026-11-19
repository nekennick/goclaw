# warehouse-kb-reader

Dịch vụ MCP (Model Context Protocol) độc lập, chỉ đọc (`read-only`), cung cấp 2 công cụ tra cứu quy trình vận hành kho Cao Lãnh và sơ đồ chi nhánh Yagami từ GoClaw cho Zalo Agent.

## 1. Nguyên tắc kiến trúc & An toàn
- **Chỉ đọc bằng kỹ thuật (Enforced Read-Only)**: Mount ổ đĩa chỉ đọc (`:ro`), chạy non-root user `1000:1000` (`goclaw`), container bật `read_only: true`, `cap_drop: ALL`, `no-new-privileges: true`.
- **Không nhân bản (No Replication, No Cache)**: Đọc trực tiếp từ 2 thư mục gốc trong volume `goclaw_goclaw-workspace`. Khi file gốc sửa đổi, dữ liệu được cập nhật ngay lập tức.
- **Kiểm chứng toàn vẹn (Integrity & Freshness)**: Đối chiếu SHA-256 byte gốc với `content_hash` trong GoClaw Vault trước khi trả dữ liệu. Báo lỗi `SOURCE_NOT_READY` hoặc `SOURCE_CHANGED` nếu phát hiện lệch.
- **Bảo vệ quyền truy cập**: Ràng buộc cố định tenant `0193a5b0-7000-7000-8000-000000000001`, chỉ chấp nhận 17 tài liệu trong allowlist đã được chủ nguồn xác nhận.

## 2. Danh mục 2 công cụ MCP
1. `warehouse_kb_search({ query: string, limit?: number })`
   - Tìm kiếm từ khóa và heading trong nội dung 17 tài liệu quy trình.
   - Trả về danh sách snippets kèm số dòng trích dẫn và SHA-256 hash.
2. `warehouse_kb_read({ doc_id: string, expected_hash: string, cursor?: string })`
   - Đọc nội dung chi tiết theo `doc_id` và `expected_hash`.
   - Hỗ trợ phân trang an toàn qua cursor opaque có chữ ký HMAC chống sửa.

## 3. Lệnh kiểm thử & Build
```bash
# Kiểm thử đơn vị & tích hợp
pnpm test

# Kiểm tra kiểu TypeScript
pnpm typecheck

# Biên dịch ra thư mục dist
pnpm build
```

## 4. Rollback nhanh
Nếu cần thu hồi tích hợp:
1. Gỡ binding server MCP khỏi Agent trên Dashboard Zalo Agent.
2. Dừng container adapter: `docker compose down`.
3. Toàn bộ Zalo Agent và GoClaw gốc không bị ảnh hưởng.
