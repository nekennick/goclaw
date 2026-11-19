import { createHash } from "node:crypto";

export function sha256(content: string | Buffer): string {
  return createHash("sha256").update(content).digest("hex");
}

/**
 * Verified sample UTF-8 Vietnamese content with diacritics, emoji, headings, and lists.
 */
export const SAMPLE_DOC_00_CONTENT = `# TỔNG QUAN NHÂN SỰ VÀ NỘI QUY KHO CAO LÃNH 🏭

## 1. Cơ cấu nhân sự chủ chốt
- Giám đốc điều hành: **Anh Khoa**
- Kế toán trưởng: **Chị Phượng**
- Kiểm soát & Kiểm kê tồn: **Anh Kiệt**
- Trưởng kho ca ngày: **Tuyền**, **Nhân**
- Trực đêm & bảo vệ: **Tín**, **Tình**

## 2. Quy định kiểm kê định kỳ
Kiểm kê toàn bộ diễn ra định kỳ vào **Thứ 4** và **Thứ 7** hàng tuần.
Khung giờ kiểm đếm vật lý: từ **13:30** đến **17:00**.

⚠️ **Lưu ý quan trọng**:
- Trong thời gian đếm kiểm, không xuất hàng khỏi kho.
- 16:00 là hạn chót kế toán hoàn tất xuất hóa đơn từ đơn hàng tạm của ngày, KHÔNG phải giờ đếm kho.
`;

export const SAMPLE_DOC_00_HASH = sha256(Buffer.from(SAMPLE_DOC_00_CONTENT, "utf-8"));

/**
 * Large document fixture to test pagination, cursor bounds, line counting.
 */
export function generateLargeDocContent(lineCount: number): { content: string; hash: string } {
  const lines: string[] = ["# TÀI LIỆU DÀI ĐỂ KIỂM THỬ PHÂN TRANG", ""];
  for (let i = 1; i <= lineCount; i++) {
    lines.push(`Dòng thứ ${i}: Quy trình kiểm tra thùng hàng số ${i} đảm bảo nguyên vẹn tem niêm phong.`);
  }
  const content = lines.join("\n");
  const hash = sha256(Buffer.from(content, "utf-8"));
  return { content, hash };
}
