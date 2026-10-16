# Yagami Morning Dashboard

Dashboard portable cho báo cáo đóng hàng sáng của Yagami. Ứng dụng không dùng PostgreSQL và không cần GoClaw để xem lịch sử đã có.

## Chạy

- Windows: nhấp đúp `start-dashboard.bat`.
- macOS/Linux: chạy `./start-dashboard.sh`.
- Mở `http://127.0.0.1:8765`.

Máy cần Python 3.10+; không cần cài thư viện ngoài.

## Mở trong mạng LAN và gửi Zalo

Trong dashboard, bật **Cho phép truy cập trong LAN** rồi bấm Lưu và khởi động lại. Xem địa chỉ IPv4 của máy chủ bằng lệnh `ipconfig`; các máy cùng mạng mở `http://<IPv4-máy-chủ>:8765`. Khi Windows hỏi Firewall, cho phép Python trên mạng Private.

Nút **Chụp & gửi Zalo** tạo ảnh PNG báo cáo. Trên điện thoại, trình duyệt sẽ mở bảng Chia sẻ để chọn Zalo; trên máy tính, ảnh được tải xuống để đính kèm vào Zalo Desktop.

## Đồng bộ KiotViet

1. Copy `config/.env.kiotviet.template` thành `config/.env.kiotviet`.
2. Điền credential KiotViet trên máy đang chạy dashboard. File credential được `.gitignore` và không nên chia sẻ.
3. Trong dashboard bấm **Đồng bộ ngay**, hoặc bật **Tự động đồng bộ** để chạy theo giờ trong `settings.json` (mặc định 07:30, múi giờ GMT+7).

Mỗi lần đồng bộ, dashboard lưu snapshot JSON tại `data/reports/`. Chỉ cần chép nguyên thư mục này sang máy khác để giữ nguyên lịch sử báo cáo.

## Thành phần dữ liệu

- `scripts/kv_morning_report.py`: logic báo cáo KiotViet đang dùng.
- `data/yagami-knowledge-base/`: lịch giao và mapping kho/chi nhánh.
- `outputs/morning-report.txt`: báo cáo nguồn mới nhất.
- `data/reports/latest.json`: bản dashboard đang hiển thị.

## Hai báo cáo phân tích

Dashboard có thêm hai tab:

- **Không đặt đúng lịch**: đọc các lần chạy thành công của cron `bao-cao-dong-hang-sang`, chỉ chọn ngày có rõ mục `Chi nhánh chưa đặt` và loại các lần chạy lỗi/trùng.
- **Mặt hàng ngừng mua**: đối chiếu mỗi cặp khách hàng/chi nhánh–mặt hàng từ 01/01/2026 đến hết ngày dữ liệu và 30 ngày gần nhất. Các tên khác nhau nhưng cùng `customerId` được gom về tên hiện tại; các tên gần giống nhưng khác `customerId` không tự động gộp.

Sau khi thay file dữ liệu trong `data/analytics/`, chạy:

```bash
python scripts/build_analytics_snapshot.py
```

Dashboard LAN chỉ cần chạy trên máy chủ; các máy khác mở `http://IP-MAY-CHU:8765`. Nút **Đồng bộ ngay** vẫn dùng cho báo cáo sáng KiotViet. Hai báo cáo phân tích chỉ thay đổi khi snapshot analytics được build lại, tránh tính toán lại mỗi lần mở trang.
