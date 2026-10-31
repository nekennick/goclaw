# Knowledge Base chuẩn nghiệp vụ kho Yagami

## Trạng thái bộ dữ liệu

**Phạm vi đã xác nhận:** Quy trình đang **thí điểm tại kho Cao Lãnh**, theo Anh Khoa xác nhận ngày 30/10/2026 (SCOPE-01). Không tự áp dụng bộ quy tắc này cho kho Bình Dương hoặc coi đã ban hành toàn hệ thống. Những chỗ nhắc Bình Dương là bối cảnh chuyển kho/đối chiếu nguồn; quy trình vận hành tại kho đó cần xác nhận và ban hành riêng. Trạng thái thí điểm không đồng nghĩa mọi mục còn chờ trong bản rà soát đã được duyệt.

Bản rà soát ngày 30/10/2026, dựa trên bộ gốc v1.0.0 đang có trong workspace nhóm của Quản lý Kim. Đây là bộ dữ liệu nghiệp vụ để hoàn thiện trước khi viết SOP. Việc tài liệu tồn tại hoặc có trường `verifiedAt` không tự chứng minh mọi nội dung đã được chủ nghiệp vụ duyệt.

Sáu tệp 00–05 giữ nội dung nghiệp vụ nền; các chỉnh lý câu chữ và bổ sung xác nhận được ghi ở tệp 08, bản gốc vẫn có bản sao lưu. Các bổ sung 06–08 ghi quy tắc quản trị dữ liệu, vấn đề phát hiện và câu hỏi cần xác nhận. Không dùng đề xuất hoặc mục đang chờ xác nhận làm quy định đã ban hành, đáp án sát hạch chính thức hay lệnh cập nhật KiotViet.

## Thứ tự đọc

1. `MANIFEST.json`: phiên bản, trạng thái và danh mục.
2. Tệp 06: cách dùng dữ liệu, định danh, mốc thời gian và nguồn chứng cứ.
3. Tệp 07: vấn đề cần xử lý, đọc cùng tệp nghiệp vụ liên quan.
4. Tệp 00–04: nội dung nghiệp vụ gốc.
5. Tệp 05: câu hỏi đào tạo hiện có; chỉ dùng chính thức sau khi quy tắc tương ứng được xác nhận.
6. Tệp 08: nhật ký rà soát và bộ câu hỏi cho Anh Khoa.

## Vị trí thực tế

- Trong container `goclaw-goclaw-1`: `/app/workspace/quan-ly-kim/zlo-ql-kim/group_zlo-ql-kim_6244108299423959453/data/warehouse-knowledge-base/`.
- Volume chứa workspace: `goclaw_goclaw-workspace`.
- Bản làm việc trên Windows: `D:\Github\goclaw\data\warehouse-knowledge-base\`.

Bản Windows là bản phục vụ rà soát/sao lưu; không có cơ chế tự đồng bộ hai chiều. Bộ dữ liệu trong thư mục không đồng nghĩa đã được index vào bộ nhớ, được tất cả agent đọc hoặc được các cron dùng. Những điều này cần kiểm tra riêng nếu triển khai tích hợp.

## Nguồn liên quan

- `../yagami-knowledge-base/yagami-branch-warehouse-map.json`: lịch giao và ánh xạ khách hàng/kho mà script báo cáo sáng hiện đọc. Dùng bản trong cùng workspace để đối chiếu; không chép lại toàn bộ lịch vào nhiều tệp.
- `../../scripts/kv_morning_report.py`: cách tính báo cáo hiện tại, là bằng chứng phần mềm đang chạy; không mặc nhiên là định nghĩa nghiệp vụ đúng.
- Dữ liệu KiotViet đồng bộ: số liệu theo từng thời điểm, không phải nơi chứa quy tắc vận hành.

## Cách hoàn thiện

Mỗi câu trả lời của Anh Khoa cần được gắn mã vấn đề ở tệp 07, ghi ngày xác nhận, người xác nhận, phạm vi kho và ngày hiệu lực. Sau đó mới sửa nội dung liên quan và đáp án đào tạo. Chưa có câu trả lời thì giữ trạng thái chờ, không tự điền người duyệt, định mức, hạn dùng hoặc ngưỡng xử lý.
