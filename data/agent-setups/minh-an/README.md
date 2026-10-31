# Minh An · Đào tạo YGM — cấu hình 31/10/2026

## Trạng thái thực tế

- Agent key: `minh-an`; UUID: `01a19038-5f68-7d48-b0cc-5bac240e2a40`.
- Agent `predefined`, trạng thái active, provider/model kế thừa lựa chọn đang chạy của hệ thống, không tạo credentials mới.
- Đã vào Team `YGM-AI-TEAM-CAOLANH` với vai trò member; không đổi Kim làm lead hoặc Helper làm member.
- Có 8 prompt: AGENTS, SOUL, IDENTITY, CAPABILITIES, USER_PREDEFINED, HEARTBEAT, AGENTS_CORE, AGENTS_TASK. Tệp nguồn bằng tiếng Anh cho LLM, đầu ra mặc định tiếng Việt.
- Đã kiểm tra prompt hiệu lực ở full/task/minimal. Không cấp exec, browser, cron, gửi tin chủ động, đọc session khác hoặc quyền gateway operator.
- Không tạo channel Zalo, không gửi tin vào nhóm, không thêm cron/heartbeat job. Thêm vào Team không đồng nghĩa đã kiểm thử toàn luồng giao việc từ Kim.

## Điểm còn chặn

Hai bộ KB đã vào Vault nhưng 17 tài liệu vẫn ở scope personal của Kim. Minh An có vault_search/vault_read nhưng chưa đọc được các tài liệu đó. Bước chuyển 17 đăng ký sang shared đã bị cơ chế duyệt an toàn từ chối vì cần người dùng phê duyệt rõ phạm vi mọi agent trong tenant. Không có thay đổi scope nào được thực hiện.

`share-kb.sql` chỉ là bản chuẩn bị, CHƯA chạy. Không tự chạy trước khi người dùng đồng ý rõ việc chia sẻ hai bộ KB với mọi agent trong tenant. `rollback-kb-scopes.sql` chỉ dùng nếu việc chia sẻ đã được phê duyệt và thực thi sau này; hiện không cần chạy.

## Kết quả kiểm tra

- Một lượt model qua HTTP nội bộ đã phản hồi được, không gửi Zalo.
- Agent báo đúng là chưa tìm/đọc được hai bộ KB, không bịa nguồn hay phiên bản; bỏ qua tài liệu nhân sự không liên quan.
- Câu góp ý học viên có giọng nhẹ nhàng, không kết luận kỷ luật.
- Lượt thử này chưa chứng minh trả lời có nguồn từ KB hoặc khả năng tiếp tục hồ sơ đào tạo qua nhiều phiên.
- Đã đối chiếu fingerprint: cấu hình và prompt Kim/Helper không đổi; scope và hash của 17 tài liệu nguồn không đổi.
- JSON, PowerShell parser và UTF-8 của tệp cấu hình kiểm tra hợp lệ.

## Sao lưu và khôi phục

`before-setup.json` lưu metadata scope/hash tài liệu và fingerprint hai agent cũ, không chứa token/credentials. Prompt nguồn ở cùng thư mục để người dùng xem và chỉnh thủ công.

Nếu cần ngưng dùng Minh An, đổi trạng thái agent sang inactive trong màn hình quản trị; có thể gỡ riêng Minh An khỏi Team. Không xóa database, volume hoặc workspace chung, không cần rollback Kim/Helper. Không chạy lệnh xóa dữ liệu để xử lý một prompt sai.

`admin.ps1` dùng API quản trị HTTP/WebSocket nội bộ; lấy token từ biến môi trường container trong bộ nhớ, không in hoặc lưu token. Nó không phải công cụ cho chính agent Minh An sử dụng. Lệnh Test tiêu thụ một lượt model và tạo phiên HTTP kiểm thử, không phải kiểm thử Zalo.

## Phạm vi kỹ thuật

Không sửa GoClaw codebase, schema, Docker image, cron hoặc giao diện. Surface parity: Gateway/API/Web UI/CLI/desktop N/A cho thay đổi mã sản phẩm; đây là provisioning agent bằng khả năng quản trị có sẵn. Không cần Go build/test vì không sửa Go.
