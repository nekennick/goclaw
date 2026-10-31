# 00. TỔNG QUAN NHÂN SỰ, NỘI QUY & MẶT BẰNG KHO YAGAMI

> Phạm vi: **đang thí điểm tại kho Cao Lãnh**, theo Anh Khoa xác nhận 30/10/2026 (SCOPE-01). Người thay thế và tiêu chí đánh giá nhân sự còn cần chốt tại AUTH-01, PEOPLE-01; phạm vi thí điểm không tự xác nhận các nội dung này. Đọc README và tệp 06 trước khi dùng làm SOP chính thức.

## 1. Bản đồ & Mặt bằng Kho (Warehouse Layout)
- **Sơ đồ kho số hoá trực quan**: Truy cập tại [http://ygm-so-so-kho.vercel.app](http://ygm-so-so-kho.vercel.app).
- Mọi nhân viên mới bắt buộc phải truy cập sơ đồ trên để nhận diện vị trí từng phân khu:
  + **Khu vực hàng bán**: Bột súp, Mì Yagami, gia vị, bao bì.
  + **Khu vực tập kết đóng hàng**: Phân theo từng tuyến xe tải nội bộ và xe ngoài.
  + **Khu vực Hàng hoá chờ phân loại**: Khu vực chuyên biệt để tiếp nhận hàng đổi trả, hàng lỗi móp méo từ tài xế mang về chờ Thủ kho thẩm định.
  + **Khu vực hàng chờ tiêu huỷ / phế phẩm**.

## 2. Phân công Trách nhiệm & Nhân sự Chủ chốt
- **Anh Khoa**: Quản lý điều hành cao nhất.
  + Tiếp nhận & giải quyết các trường hợp đặc biệt, sai lệch số tồn lớn không rõ nguyên nhân.
  + **Phê duyệt điều chỉnh tồn KiotViet** khi kiểm kho vẫn lệch sau đối soát và bổ sung hàng nhập; **Anh Khoa nói trực tiếp với Kiệt để duyệt, Kiệt mới được dùng chức năng Kiểm kho và thực hiện “Cân bằng kho”**, theo xác nhận ngày 31/10/2026 (INV-02/AUTH-01). Cách hiện hành không tạo dấu vết phê duyệt riêng; chưa tự đặt thêm yêu cầu ký hoặc nhắn nhóm.
  + Trực tiếp chấm điểm sát hạch nhân viên mới sau thời gian thử việc.
  + **Giám đốc duyệt hủy hàng trả về bị hỏng, không thể sử dụng**, theo xác nhận Anh Khoa ngày 30/10/2026 (RET-01). Hàng này không nhập tồn trước khi hủy; thẩm quyền cho các loại hủy/điều chỉnh khác cần xác nhận riêng.
- **Chị Phượng (Kế toán trưởng)**:
  + Thẩm quyền duy nhất phê duyệt các đơn hàng phát sinh hoặc xin thay đổi số lượng sau 07:00 sáng.
  + Điều động tài xế kho chạy xe sang xưởng lấy hàng bù khi có cảnh báo thiếu khả dụng súp.
  + Cùng Anh Khoa trực tiếp chấm điểm thử việc nhân viên mới.
- **Anh Kiệt**:
  + Chốt chặn kiểm tra chứng từ nhập hàng khớp giữa App nội bộ với KiotViet trước khi bấm xác nhận nhập hàng.
  + Chủ trì cùng kế toán rà soát chứng từ nhập - xuất khi phát hiện lệch số 4 mã siết tồn.
  + Nhận số đếm thực tế do nhân viên phụ trách từng khu gửi; lấy tồn KiotViet **sau 16:00** làm mốc đối chiếu vì kế toán hoàn tất xuất hóa đơn hàng đã đi buổi sáng trước 16:00 (INV-02, Anh Khoa xác nhận 30/10/2026).
  + Khi chênh lệch lớn khớp thông báo nhận hàng của thủ kho trên nhóm nhận hàng, Kiệt được tự bổ sung lượng hàng nhập tương ứng vào **biên bản kiểm kho**. Việc này không tự xác lập quyền điều chỉnh tồn KiotViet.
  + **Dùng chức năng Kiểm kho của KiotViet và chỉ thực hiện “Cân bằng kho” sau khi Anh Khoa nói trực tiếp để đồng ý**, theo xác nhận 31/10/2026. Không tự duyệt hoặc cân bằng trước sự đồng ý của Anh Khoa. Phê duyệt bằng lời nói hiện không có dấu vết riêng để truy vết sau này.
- **Thủ kho Tuyền & Thủ kho Nhân**:
  + Trực tiếp tiếp nhận hàng nhập từ xưởng sản xuất và nhà cung cấp (13:30 - 17:00).
  + Kiểm đếm số lượng, kiểm tra Date/Hạn sử dụng, độ nguyên vẹn bao bì.
  + Nhập liệu vào App nội bộ và chụp hình báo lên nhóm "Quản lý Hàng Nhập".
  + Đánh giá hàng trả về: hàng bán tiếp được thì nhập lại trên ứng dụng; hàng hỏng thì Giám đốc duyệt hủy, không nhập tồn trước hủy và báo nhóm hủy hàng (chi tiết RET-01 tại tệp 03).
  + Kiêm nhiệm chở hàng ra chành xe giao xe ngoài khi cần.
- **Nhân viên Kho**:
  + Bốc dỡ, soạn hàng, đóng gói theo phiếu đặt hàng do kế toán in.
  + Bốc hàng lên xe tải theo nguyên tắc thứ tự tuyến đường và quy chuẩn tải trọng.
  + Kiểm đếm **toàn bộ kho thứ Tư và thứ Bảy hằng tuần, từ 13:30 đến 17:00**, bao gồm 4 mã siết tồn. Trong lúc kiểm **vẫn cho nhập hàng, không cho xuất hàng**; số đếm phải đủ lượng thực tế và khớp số liệu kho mới nhất (INV-02, Anh Khoa xác nhận 30/10/2026). **16:00 là hạn xuất hóa đơn từ phiếu tạm của tất cả đơn khách đặt trong ngày**, tách biệt với lịch kiểm kho.
  + Từ 13:30, nhân viên phụ trách khu nào đếm số lượng thực tế khu đó rồi gửi Kiệt để đối chiếu với tồn KiotViet lấy sau 16:00.
  + Nếu có hàng nhập sau khi đã đếm, cộng bổ sung theo số nhận thực tế do thủ kho báo rồi gửi số liệu cho Kiệt.
  + Khi đối chiếu phát hiện chênh lệch, **nhân viên kho kiểm đếm lại hàng thực tế** trước khi trình Anh Khoa duyệt Cân bằng kho. Chưa quy định một cá nhân cố định hoặc số lần đếm lại bắt buộc.
  + Dọn dẹp vệ sinh kho bãi cuối ca.
- **Đội ngũ ngủ lại giữ kho (Tín, Tình, Tuyền)**:
  + Trực tiếp ngủ lại tại kho, chịu trách nhiệm bảo vệ an ninh trật tự và tài sản kho bãi.
  + Kiểm tra tắt toàn bộ thiết bị điện không sử dụng trước khi nghỉ đêm.
- **Kế toán kho**:
  + Hoàn tất xử lý đơn khách đặt trong ngày **trước 16:00**, nghĩa là đã **xuất hóa đơn từ những đơn đặt (phiếu tạm)**, theo xác nhận Anh Khoa ngày 30/10/2026 (TIME-01).
  + In phiếu đặt hàng lúc 07:30 giao cho kho soạn hàng.
  + Trực tiếp ra kho kiểm đếm độc lập đối soát số lượng trước khi bốc hàng lên xe.
  + 13:30 vào văn phòng xử lý hóa đơn chính thức trên KiotViet và gửi ảnh vào nhóm Zalo chi nhánh.
  + Tạo phiếu chuyển kho Cao Lãnh -> Bình Dương.
- **Tài xế kho nội bộ**:
  + Chuyên trách chở hàng hoá giữa xưởng sản xuất và kho, trung chuyển nguyên liệu.
  + Chở hàng ra chành xe gửi các đơn xe ngoài và chụp biên nhận gửi lên nhóm tài xế.

## 3. Nội quy & Kỷ luật Kho bắt buộc
- **CẤM HÚT THUỐC, CẤM LỬA 100%**: Tuyệt đối nghiêm cấm hút thuốc, bật lửa trong toàn bộ khuôn viên kho.
- **Ăn uống & Sinh hoạt**: Ăn uống đúng nơi quy định, không mang đồ ăn/nước uống vào khu vực kệ hàng và đóng gói.
- **Tư trang cá nhân**: Tự bảo quản, để ngăn nắp đúng nơi quy định của kho.
- **An toàn điện & Khoá cửa**: Bộ ba Tín, Tình, Tuyền kiểm tra tắt thiết bị điện khi không sử dụng và đảm bảo chốt khoá cửa an toàn.

## 4. Tiêu chí Đánh giá Thử việc Nhân viên Mới
- Thời gian thử việc: **3 tháng**.
- Điều kiện đạt chuẩn làm việc chính thức: Được **Anh Khoa** và **Chị Phượng** chấm thang điểm từ **7/10 điểm trở lên** dựa trên:
  + Kỷ luật giờ giấc và chấp hành nội quy an toàn kho bãi.
  + Độ chính xác khi soạn hàng (không soạn nhầm mã, không làm rách hỏng bao bì).
  + Khả năng nắm vững và thực thi thuần thục quy trình chuẩn (SOP).
