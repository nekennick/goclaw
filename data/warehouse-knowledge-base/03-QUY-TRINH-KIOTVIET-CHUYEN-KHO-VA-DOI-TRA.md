# 03. THAO TÁC KIOTVIET, CHUYỂN KHO & XỬ LÝ HÀNG ĐỔI TRẢ

> Phạm vi: **thí điểm tại kho Cao Lãnh** (SCOPE-01). Mô tả bên Bình Dương trong luồng chuyển kho là thông tin liên quan hai đầu; không coi bộ quy trình đã được triển khai tại Bình Dương.

> Rà soát 30/10/2026: phần 4 đã cập nhật theo RET-01 trong thí điểm Cao Lãnh: nhập lại trên ứng dụng tự cập nhật KiotViet; Giám đốc duyệt hủy hàng trả hỏng, không nhập tồn trước hủy. Tên chứng từ, kiểm tra độc lập và nhóm đích còn cần chốt. Xem TRANS-01, ORDER-01 trong tệp 07.

## 1. Nguồn Đơn Hàng & Thao tác KiotViet
- Chi nhánh đặt hàng thông qua App/Web nội bộ của công ty (kết nối trực tiếp qua API KiotViet tạo thành Phiếu Tạm).
- **Định nghĩa hoàn tất xử lý (Anh Khoa xác nhận 30/10/2026):** Phải xuất hóa đơn từ những đơn đặt (phiếu tạm). Tất cả đơn khách đặt trong ngày phải được xử lý đến bước này trước **16:00**. Đây là tiêu chí xử lý chứng từ, không phải giờ kiểm tồn hoặc xác nhận khách đã nhận hàng.
- **Trong buổi kiểm kho thứ Tư/thứ Bảy:** Kế toán vẫn xuất hóa đơn cho hàng đã đi buổi sáng đến trước 16:00. Kiệt dùng tồn KiotViet lấy sau 16:00 làm mốc đối chiếu số thực tế từ từng khu (INV-02); không diễn giải dừng xuất hàng lúc kiểm thành dừng việc xuất hóa đơn này.
- Thời điểm chuyển từ "Phiếu Tạm" thành "Hóa Đơn":
  + **Buổi sáng**: Chỉ xuất hóa đơn ngay cho 2 nhóm:
    1. Đơn gửi xe ngoài (chành xe) — để có chứng từ kèm theo hàng ra chành.
    2. Đơn xuất bù.
  + **Buổi chiều (từ 13:30)**: Kế toán vào văn phòng xử lý đơn đặt sáng, đối soát các phát sinh thực tế khi đóng hàng rồi mới bấm xuất hóa đơn chính thức cho tất cả các đơn xe nội bộ.

## 2. Quy trình Chuyển Kho (Tổng kho Cao Lãnh ➔ Kho Bình Dương)
- **Chứng từ**: Sử dụng "Phiếu Chuyển Hàng" trên KiotViet.
- **Trách nhiệm 2 đầu kho**:
  + **Kế toán Tổng kho Cao Lãnh**: Tạo phiếu chuyển kho khi hàng (chủ yếu là súp) được bốc lên chuyến xe Chủ Nhật.
  + **Kế toán Kho Bình Dương**: Tiếp nhận hàng thực tế, kiểm đếm đủ số lượng và bấm nút xác nhận "Nhận Hàng" trên KiotViet để hoàn tất.

## 3. Xử lý Hủy đơn hoặc Giảm số lượng sau Giờ Cắt Đơn (07:00)
- Mọi phát sinh xin hủy hoặc bớt hàng phải được **Chị Phượng (KTT)** duyệt.
- Nếu hàng đã soạn ra khu tập kết: Nhân viên kho mang trả hàng về đúng vị trí cũ trong kho sau khi kết thúc ca đóng hàng sáng.
- Kế toán kho ghi chú lại các món giảm/hủy để khi xuất hóa đơn lúc 13:30 sẽ trừ món trực tiếp (đơn hủy thì không xuất hóa đơn và không giao).

## 4. Xử lý Hàng Trả Về (Từ chối nhận / Rách bao / Móp méo)
- **Tiếp nhận hàng tại kho**:
  + Tài xế chở hàng về bàn giao trực tiếp cho Thủ kho.
  + Nếu Thủ kho không có mặt: Đặt hàng vào khu vực **"HÀNG HOÁ CHỜ PHÂN LOẠI"**. Tuyệt đối không để lẫn vào khu vực hàng có thể bán.
- **Đánh giá chất lượng (Thủ kho chủ trì)**:
  + Thủ kho nhận hàng từ tài xế và xác định hàng có đủ điều kiện bán tiếp hay không.
  + **Hàng đủ điều kiện bán tiếp:** Thủ kho tiến hành nhập kho lại trên ứng dụng. Nội dung lý do phải nêu chi nhánh trả, lý do trả và việc đã trừ công nợ, theo mẫu: “Nhập hàng từ chi nhánh [tên chi nhánh] trả về với lý do [lý do trả] — đã trừ công nợ”. Sau phân loại, hàng được đưa về khu vực hàng có thể bán theo nội dung nguồn.
  + **Đồng bộ tồn:** Nhập kho lại trên ứng dụng sẽ tự cập nhật tồn KiotViet, theo xác nhận Anh Khoa. Đây là một lần ghi nhận nhập lại qua ứng dụng; không hướng dẫn tạo thêm một lần nhập độc lập trên KiotViet cho cùng số hàng.
  + **Hàng hỏng, không thể sử dụng:** Giám đốc duyệt hủy, sau đó tiêu hủy và nhắn lên nhóm hủy hàng. **Không nhập tồn trước khi hủy.** Đây là nhánh xử lý hàng trả hỏng đã được Anh Khoa xác nhận; không tự áp dụng phương án trả nhà cung cấp hoặc suy rộng quy tắc này sang các loại hàng hủy khác.
- **Nghiệp vụ tài chính kế toán**:
  + Hóa đơn đã xuất rồi thì KHÔNG sửa lại hóa đơn.
  + Kế toán xử lý bằng cách trừ tiền hàng trả về vào công nợ của khách hàng trên hóa đơn đó.

### Bổ sung xác nhận ngày 30/10/2026 — RET-01

- **Anh Khoa xác nhận cách làm hiện tại:** “Chỉ điều chỉnh công nợ, tồn được xử lý riêng”.
- **Xác nhận bổ sung:** Thủ kho nhận hàng từ tài xế, hàng đủ điều kiện bán tiếp thì nhập lại trên ứng dụng và ghi lý do như trên; hàng hỏng không thể sử dụng thì tiêu hủy, nhắn lên nhóm hủy hàng.
- **Xác nhận tiếp theo:** “Nhập lại trên ứng dụng sẽ tự cập nhật tồn kiotviet. Hàng hỏng giám đốc sẽ duyệt huỷ, không nhập tồn trước khi huỷ”.
- Công nợ và tồn kho là hai phần cần theo dõi tách biệt. Trường lý do “đã trừ công nợ” ghi nhận thông tin nghiệp vụ theo xác nhận của Anh Khoa; không tự nó là chứng từ chứng minh việc giảm công nợ đã hoàn tất. Khi kiểm tra một lần trả cụ thể cần đối chiếu bằng chứng công nợ tương ứng.
- **Phạm vi đã chốt:** Thí điểm tại kho Cao Lãnh. **Chưa chốt:** ứng dụng/màn hình và chứng từ KiotViet; người kiểm tra nhập lại; nơi lưu phê duyệt hủy; tên/ID nhóm hủy hàng. Cơ chế tự cập nhật tồn đã được chủ nghiệp vụ xác nhận, chưa kiểm tra kỹ thuật bằng giao dịch trong lần rà soát này.
- Khi tư vấn hoặc chuẩn bị SOP, được dùng luồng thực tế đã xác nhận nhưng không tự gán chứng từ KiotViet, quyền duyệt hoặc trình tự hạch toán hàng hỏng cho phần chưa được trả lời. Các trường hợp không thuộc hai nhánh trên, gồm đổi/trả nhà cung cấp, cần xác nhận riêng.
