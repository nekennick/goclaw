# 07. Danh mục vấn đề cần xác nhận

Mốc rà soát: 31/10/2026. Ưu tiên P0: có thể dẫn đến sai tồn/kết luận/chứng từ; P1: cần trước khi viết SOP; P2: chuẩn hóa đào tạo. Đây là ưu tiên rà soát dữ liệu, không phải phân công mới cho nhân sự.

| Mã | Ưu tiên | Tệp nguồn | Nội dung còn thiếu hoặc cần làm rõ |
|---|---|---|---|
| SCOPE-01 | P1 | Xác nhận Anh Khoa; 00, 01 | ĐÃ XÁC NHẬN: quy trình đang thí điểm ở kho Cao Lãnh. Không tự áp sang Bình Dương/toàn hệ thống. Ngày bắt đầu thí điểm chưa được cung cấp; không lấy ngày xác nhận làm ngày triển khai. |
| AUTH-01 | P1 | 00, 01, 03, 04 | Đã chốt Giám đốc duyệt hủy hàng trả hỏng (RET-01); Anh Khoa **nói trực tiếp** để duyệt, Kiệt dùng chức năng Kiểm kho và chỉ **Cân bằng kho** sau khi nghe đồng ý (INV-02). Cách hiện hành không có dấu vết phê duyệt riêng. Còn thiếu người thay thế/ủy quyền khi Phượng/Kiệt/thủ kho vắng; việc bổ sung dấu vết chỉ là đề xuất, chưa phải quy định. |
| TIME-01 | P0 | 00, 01, 03, 04; xác nhận Anh Khoa | ĐÃ XÁC NHẬN: tất cả đơn khách đặt trong ngày phải được xuất hóa đơn từ đơn đặt (phiếu tạm) trước 16:00; đó là hoàn tất xử lý đơn, không phải giờ kiểm tồn hay xác nhận đã giao. Lịch đếm theo INV-02; ngoại lệ theo ORDER-01, REPORT-01. |
| INV-01 | P0 | 02; script sáng | `need - (onHand - reserved)` có đếm trùng nhu cầu? `reserved` bao gồm đơn nào, cảnh báo muốn nói thiếu giao hay thiếu nhận thêm đơn? |
| RET-01 | P0 | 03 §4 | Trong thí điểm Cao Lãnh: thủ kho nhận hàng trả; hàng bán được nhập lại trên ứng dụng tự cập nhật KiotViet với lý do đã xác nhận; hàng trả hỏng do Giám đốc duyệt hủy, không nhập tồn trước hủy và báo nhóm. Còn thiếu ứng dụng/chứng từ, người kiểm tra nhập lại, nơi lưu phê duyệt và nhóm đích. |
| INV-02 | P0 | 00, 01, 04 §3; xác nhận Anh Khoa | Thí điểm Cao Lãnh: đã chốt lịch/phân công/mốc sau 16:00, bổ sung nhập; **nhân viên kho đếm lại khi có chênh lệch**; Anh Khoa nói trực tiếp để duyệt; Kiệt dùng chức năng Kiểm kho, chỉ Cân bằng kho sau khi được đồng ý. Cách duyệt hiện không có dấu vết riêng. Còn thiếu nơi lưu mốc kiểm, nhập sau mốc lấy tồn, người xác nhận kết quả và số lần đếm lại bắt buộc. |
| TRANS-01 | P0 | 03 §2; lịch giao JSON | Tệp 03 chỉ mô tả chuyến Chủ Nhật; lịch JSON còn có chỉ dẫn THỨ 7 “KHO BD (HÀNG ĐÓNG)”. Cần xác nhận đầy đủ ngày/loại chuyến, thiếu/thừa/hỏng và hàng đang đi đường. |
| ORDER-01 | P1 | 01, 03 | Đã chốt hoàn tất = xuất hóa đơn từ phiếu tạm trước 16:00. Còn thiếu nơi lưu phê duyệt tăng/giảm/hủy sau 07:00, người thông báo phiên bản cuối và cách xử lý phát sinh sau hạn, xuất một phần/gộp/xuất lại hoặc hủy. |
| PACK-01 | P1 | 02 §1, §3 | Súp đơn vị “bịch” nhưng quy cách “bốc nguyên bao”: bao và bịch có là một? Bao/thùng có bao nhiêu đơn vị, có mã quy đổi trên KiotViet không? |
| EXP-01 | P1 | 02 §2; 05 câu 2 | HSD 6/8 tháng cần đối chiếu nhãn/mã/lô đang dùng. FEFO cần diễn đạt “HSD gần nhất xuất trước”; ngưỡng cận date, nhãn mờ và ngoại lệ chưa rõ. |
| COLD-01 | P1 | 02 §3; 05 câu 3 | Thùng xốp chưa đủ làm bằng chứng giữ nhiệt. Chưa có yêu cầu theo sản phẩm, thời gian vận chuyển và cách kiểm tra; không tự đặt nhiệt độ. |
| HAND-01 | P1 | 01, 04 | Thiếu hồ sơ kiểm/bàn giao và người quyết định khi giao thiếu/sai/khách từ chối; “uyển chuyển” cần phạm vi phê duyệt cụ thể. |
| REPORT-01 | P0 | 01, 03; script sáng | Tiêu chí nghiệp vụ đã rõ: phiếu tạm có hóa đơn tương ứng trước 16:00. Còn thiếu cách đối soát ngoại lệ, lưu mốc xuất hóa đơn để xác định đúng hạn, snapshot sáng–chiều, trạng thái đồng bộ/gửi, chạy bù và lịch có hiệu lực. Không mặc nhiên giới hạn tất cả đơn trong ngày thành chỉ đơn theo lịch. Số tồn báo cáo không phải kết quả kiểm đếm lúc 16:00. |
| PEOPLE-01 | P2 | 00 §4; 05 câu 7 | “3 tháng/7 trên 10”: cần người quản lý xác nhận phạm vi áp dụng, cách chấm, điều kiện sai nghiêm trọng, tái kiểm tra và phiên bản đáp án. Không tự coi là chính sách nhân sự đã rà soát. |
| DATA-01 | P1 | MANIFEST; bộ lịch giao | Manifest ghi `verifiedAt` nhưng chưa có bằng chứng từng quy tắc/ngày hiệu lực. Manifest lịch giao và JSON chính khác version. Cần quản lý thay đổi có nguồn. |

## Những chỉnh lý có thể làm ngay ở tầng dữ liệu

- Dùng cùng một mã chuẩn MYG khi tra cứu 4 mã; giữ nguyên `myg` trong nguồn gốc để truy vết, đối chiếu mã thật trước thao tác.
- Phân biệt danh mục khách nhận hàng với kho quản lý; liên kết bằng mã thay vì tên gần giống.
- Không dùng `onHand` như tên khác của “số đếm thực tế”.
- Phân biệt “đã có hóa đơn”, “đã ra xe”, “đã giao” và “đã nhận”; chưa có bằng chứng thì không tự nâng trạng thái.
- Tách “nguồn gốc ghi như vậy”, “anh vừa xác nhận”, “đề xuất”, “chưa biết”.

Các điểm này là hướng dẫn đọc và tổ chức dữ liệu trong tệp 06. Chưa có thay đổi script, lịch cron hoặc thao tác KiotViet trong đợt rà soát này.
