# 08. Nhật ký rà soát và câu hỏi hoàn thiện

Đây là nhật ký theo từng lượt xác nhận. Các câu “chưa chốt” trong mục phiên bản cũ phản ánh thời điểm đó; trạng thái hiện tại nằm tại tệp 07, RULE_REGISTER và phần câu hỏi hiện hành bên dưới. Phạm vi hiện tại đã xác nhận: thí điểm kho Cao Lãnh (SCOPE-01).

## Lần rà soát 30/10/2026

- Đã đọc toàn bộ 6 tài liệu 00–05 và MANIFEST v1.0.0 từ workspace nhóm của Quản lý Kim.
- Đã đọc metadata bộ lịch giao cùng workspace và đối chiếu phần chọn đơn/tính tồn của `kv_morning_report.py`.
- Đã kiểm tra bộ gốc nằm trong volume `goclaw_goclaw-workspace` và tạo bản làm việc tại `D:\Github\goclaw\data\warehouse-knowledge-base\`.
- Bản gốc trước rà soát được giữ tại `../.warehouse-kb-backups/v1.0.0-before-review-20261030/` trong cùng workspace Docker.
- Đã bổ sung README, từ điển/quản trị dữ liệu, danh mục 15 vấn đề và sổ trạng thái quy tắc. Không viết SOP hoàn chỉnh trong lần này.
- Đã gắn lưu ý rà soát vào đầu tệp 00–05 để tránh sử dụng riêng một tệp mà bỏ qua vấn đề chưa chốt.
- Tệp 02 và đáp án câu 2 tệp 05: làm rõ câu chữ FEFO là HSD gần nhất xuất trước; chuẩn hóa MYG ở tệp 02. Không thay đổi định mức HSD trong nguồn hay công thức báo cáo.
- Tệp 03: bổ sung xác nhận trực tiếp RET-01 về công nợ/tồn hàng trả, chỉ rõ phần còn thiếu.
- MANIFEST: giữ `verifiedAt` nguồn, bổ sung phạm vi xác nhận và chuyển bản rà soát sang `1.0.1-review.1`; `approvedAt/approvedBy` còn trống.
- Chưa kiểm tra thực địa, nhãn hàng, hồ sơ phê duyệt nhân sự, cấu hình giữ tồn KiotViet hoặc tính đúng đắn của số liệu giao dịch. Không coi ngày rà soát là ngày duyệt nghiệp vụ.

## Xác nhận trực tiếp đã nhận

### RET-01 — Công nợ và tồn hàng trả

- Người xác nhận: Anh Khoa, trong hội thoại rà soát ngày 30/10/2026.
- Nội dung nguyên văn: “Chỉ điều chỉnh công nợ, tồn được xử lý riêng”.
- Xác nhận bổ sung nguyên văn: “Là bước thủ kho nhận hàng từ tài xế trả về, nếu đủ điều kiện bán tiếp thì tiến hành nhập kho lại trên ứng dụng và ghi lí do là nhập hàng từ chi nhánh … trả về với lí do … đã trừ công nợ. Nếu hàng hỏng không thể sử dụng thì tiêu huỷ, nhắn lên nhóm huỷ hàng”.
- Phạm vi được xác nhận: cách làm hiện tại khi xử lý hàng trả về đã xuất hóa đơn, theo câu hỏi đã trao đổi.
- Đã rõ thêm người thực hiện nhận và nhập lại là thủ kho; luồng hàng còn bán được qua ứng dụng; luồng hàng hỏng tiêu hủy và báo nhóm. Không tự nâng toàn bộ RET-01 thành hoàn tất.
- Xác nhận tiếp theo nguyên văn: “Nhập lại trên ứng dụng sẽ tự cập nhật tồn kiotviet. Hàng hỏng giám đốc sẽ duyệt huỷ, không nhập tồn trước khi huỷ”.
- Đã rõ thêm: ứng dụng tự cập nhật tồn KiotViet; người duyệt hủy hàng trả hỏng là Giám đốc; không nhập tồn hàng trả hỏng trước khi hủy. Đây là xác nhận nghiệp vụ, chưa kiểm thử kỹ thuật giao dịch đồng bộ.
- Chưa được xác nhận: tên ứng dụng/màn hình và chứng từ KiotViet, người kiểm tra nhập lại, hình thức lưu phê duyệt hủy, tên/ID nhóm hủy và phạm vi từng kho.
- Chưa có ngày hiệu lực chính sách riêng. Không suy ra đây là yêu cầu phải bỏ phiếu trả hàng hay tự cấp quyền điều chỉnh kho.

### Cập nhật v1.0.1-review.2

- Cập nhật tệp 03 theo hai nhánh hàng trả vừa được Anh Khoa xác nhận; tách phần đã biết khỏi những bước hạch toán/thẩm quyền chưa rõ.
- Thêm câu 5b vào tệp 05 để đáp án đào tạo bám đúng xác nhận, chưa mở rộng sang thao tác KiotViet hoặc quyền duyệt hủy.
- Cập nhật RET-01 ở tệp 07 và RULE_REGISTER, giữ trạng thái cần xác nhận cho các điểm còn thiếu.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.1-before-returns-update/`.

### Cập nhật v1.0.1-review.3

- Ghi nhận tự cập nhật tồn KiotViet khi nhập hàng bán tiếp được trên ứng dụng; tránh hướng dẫn nhập cùng hàng thêm lần nữa trên KiotViet.
- Ghi nhận Giám đốc duyệt hủy hàng trả hỏng và không nhập tồn trước hủy; giữ bước nhắn nhóm hủy theo xác nhận trước.
- Đồng bộ nội dung ở tệp 00, 03, 05, 07, RULE_REGISTER và manifest. Những chi tiết kiểm tra/chứng từ/nhóm/phạm vi còn thiếu giữ trạng thái chờ xác nhận.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.2-before-sync-disposal-confirmation/`.

### TIME-01 — Đính chính mốc 16:00, v1.0.1-review.4

- Người xác nhận: Anh Khoa, hội thoại ngày 30/10/2026.
- Nguyên văn: “Lúc 16:00 không phải là kiểm tồn, mà trong ngày tất cả đơn đặt của khách phải được xử lý hoàn tất trước giờ đó.”
- Đã sửa tệp 00, 01, 04 để bỏ lịch đếm tồn 16:00; lịch đếm thực tế cần xác nhận riêng (INV-02).
- Thêm câu đào tạo 8; cập nhật từ điển, bảng rà soát và RULE_REGISTER. TIME-01 xác nhận riêng ý nghĩa/hạn giờ; không tự chốt định nghĩa hoàn tất, phạm vi kho hoặc ngoại lệ sau hạn.
- Đây là đính chính nội dung gốc và giả định kiểm tồn trong câu hỏi rà soát trước, không phải thay đổi lịch cron hoặc script báo cáo.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.3-before-1600-correction/`.

### TIME-01 — Định nghĩa hoàn tất xử lý, v1.0.1-review.5

- Anh Khoa xác nhận nguyên văn ngày 30/10/2026: “Xử lý hoàn tất là phải xuất hoá đơn của những đơn đặt (phiếu tạm)”.
- Ghép với xác nhận hạn trước đó: tất cả đơn khách đặt trong ngày phải được xuất hóa đơn từ phiếu tạm trước 16:00. Không đồng nhất với kiểm đếm tồn hoặc xác nhận khách đã nhận hàng.
- Cập nhật tài liệu trách nhiệm, timeline, thao tác KiotViet, đáp án câu 8, từ điển, bảng rà soát và sổ quy tắc; bỏ các câu hỏi còn treo về định nghĩa cơ bản đã được trả lời.
- Chưa tự chốt cách phân loại đơn hủy, xuất một phần/gộp/xuất lại, đơn phát sinh sau hạn hoặc trường kỹ thuật dùng xác định giờ xuất hóa đơn. Chưa sửa cron hay script phân tích.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.4-before-completion-definition/`.

### INV-02 — Lịch kiểm toàn bộ kho, v1.0.1-review.6

- Anh Khoa xác nhận nguyên văn ngày 30/10/2026: “Theo lịch định kỳ thứ 4 và thứ 7 sẽ kiểm toàn bộ kho”.
- Ghi nhận ngày thứ Tư/thứ Bảy hằng tuần và phạm vi toàn bộ kho, không giới hạn 4 mã siết tồn. Bỏ các câu khẳng định chưa biết tần suất/lịch ngày kiểm.
- Cập nhật trách nhiệm, timeline, cẩm nang, câu đào tạo 8–9, từ điển, bảng rà soát và sổ quy tắc. Chi tiết lịch kiểm được lưu tại INV-02; TIME-01 chỉ xác nhận hạn xử lý đơn, không dùng để định giờ kiểm.
- Chưa xác nhận giờ đếm/chốt số, nhập/xuất trong khi kiểm, phân công người đếm/đối soát, phê duyệt chênh lệch hoặc lịch áp dụng cho địa điểm kho nào. Không tự bổ sung lịch đếm 4 mã hằng ngày.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.5-before-count-schedule/`.

### INV-02 — Khung giờ và nhập/xuất trong khi kiểm, v1.0.1-review.7

- Anh Khoa xác nhận nguyên văn ngày 30/10/2026: “Không cần tạm dừng nhập hàng vì chỉ cần nhân viên đếm đủ lượng hàng thực tế khớp với số liệu mới nhất trên kho là được. Xuất thì không cho xuất hàng lúc kiểm kho. Thời gian kiểm kho là 13h30 đến 17h”.
- Ghép với lịch đã chốt: thứ Tư/thứ Bảy hằng tuần, 13:30–17:00, kiểm toàn kho; nhập tiếp tục, xuất hàng dừng, đối chiếu đủ hàng thực tế với số kho mới nhất.
- Đã cập nhật tài liệu trách nhiệm, timeline, cẩm nang, đào tạo, từ điển, bảng rà soát và sổ quy tắc. Không áp dụng cơ chế khóa toàn bộ nhập/xuất hoặc snapshot cố định đầu buổi khi chưa được xác nhận.
- Còn thiếu người đếm/đối soát, cách cập nhật số đếm khi nhập hàng sau khi đã đếm mã đó, mốc so sánh, kiểm lại/duyệt chênh lệch và phạm vi địa điểm kho. Xuất hóa đơn trong khi kiểm có thể thay đổi tồn phần mềm nên cần xác nhận quy tắc riêng, không tự diễn giải từ việc dừng xuất hàng.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.6-before-count-window/`.

### INV-02 — Đếm theo khu, Kiệt đối chiếu sau 16:00, v1.0.1-review.8

- Anh Khoa xác nhận nguyên văn ngày 30/10/2026: “Đúng kế toán vẫn xuất hoá đơn đến trước 16h vì hàng đã đi buổi sáng rồi. Kiểm kho từ 13h30 là nhân viên phụ trách sẽ đếm số lượng thực tế ở khu của mình sau đó gửi số liệu thực tế cho Kiệt, Kiệt sẽ lấy tồn kho kiotviet sau 16h để đảm bảo dữ liệu xuất khớp với hàng thực tế đi. Số liệu sau 16h sẽ làm mốc để đối chiếu”.
- Chốt nhân viên đếm khu mình từ 13:30 rồi gửi Kiệt; Kiệt lấy tồn KiotViet sau 16:00 đối chiếu. Kế toán vẫn xuất hóa đơn hàng đã đi sáng đến trước 16:00, phù hợp việc dừng xuất hàng thực tế trong lúc kiểm.
- Cập nhật vai trò, timeline, thao tác chứng từ, truy vết, câu đào tạo 9–10, từ điển, bảng rà soát và RULE_REGISTER. Đã sửa dòng mô tả hàng trả của thủ kho trong tệp 00 cho khớp RET-01 đã xác nhận ở tệp 03.
- Còn thiếu cách bổ sung hàng nhập sau đếm hoặc sau mốc Kiệt lấy tồn, lưu mốc số liệu chính xác, đếm lại/duyệt chênh lệch và phạm vi địa điểm kho. Không tự chuyển “sau 16:00” thành một giờ lấy tồn cố định.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.7-before-kiet-reference/`.

### INV-02 — Bổ sung hàng nhập vào số kiểm đếm, v1.0.1-review.9

- Anh Khoa xác nhận nguyên văn ngày 30/10/2026: “nhân viên cộng bổ sung theo số nhận thực tế từ thủ kho báo và gửi Kiệt. Hoặc nếu thấy chênh lệch quá lớn và khớp với thông báo nhận hàng của thủ kho trên nhóm nhận hàng, Kiệt sẽ biết và tự bổ sung vào biên bản kiểm kho”.
- Ghi đủ hai cách: nhân viên cộng số thực nhận rồi gửi Kiệt; hoặc Kiệt bổ sung biên bản khi chênh lệch lớn khớp thông báo nhận hàng. Phân biệt cập nhật biên bản kiểm kho với quyền điều chỉnh tồn KiotViet.
- Chuẩn hóa nguyên tắc toàn vẹn: cùng lượng nhập chỉ cộng một lần; số gốc và phần bổ sung cần phân biệt. Đây là yêu cầu giữ kết quả đúng phép cộng, không thêm thẩm quyền nghiệp vụ mới.
- Cập nhật trách nhiệm, timeline, truy vết, đào tạo câu 9/11, từ điển, bảng rà soát và RULE_REGISTER. Ngưỡng “lớn”, hình thức lưu chứng cứ, xử lý nhập sau mốc lấy tồn và phê duyệt chênh lệch vẫn chờ xác nhận.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.8-before-receipt-addendum/`.

### INV-02/AUTH-01 — Người duyệt điều chỉnh tồn, v1.0.1-review.10

- Anh Khoa xác nhận nguyên văn ngày 30/10/2026: “Anh khoa sẽ là người duyệt điều chỉnh tồn kiotviet”.
- Phạm vi theo câu hỏi đã trao đổi: sau đối soát/bổ sung đầy đủ mà kiểm kho vẫn lệch, Anh Khoa là người duyệt điều chỉnh tồn KiotViet.
- Đồng bộ trách nhiệm, timeline, truy vết, câu đào tạo 12, từ điển, bảng rà soát và sổ quy tắc. Tách người duyệt khỏi người thao tác; chưa tự gán Kiệt hay kế toán là người bấm điều chỉnh.
- Còn thiếu nơi lưu phê duyệt, người thao tác và các điểm về bằng chứng/mốc/phạm vi kho. Không thực hiện giao dịch điều chỉnh tồn trong lần cập nhật tài liệu này.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.9-before-adjustment-approval/`.

### SCOPE-01 — Thí điểm tại kho Cao Lãnh, v1.0.1-review.11

- Anh Khoa xác nhận nguyên văn ngày 30/10/2026: “Quy trình đang thí điểm ở kho Cao Lãnh”.
- Chốt phạm vi bộ quy trình đang hoàn thiện là thí điểm tại Cao Lãnh. Không tự áp sang Bình Dương hoặc coi đã ban hành toàn hệ thống; mô tả Bình Dương trong luồng chuyển kho là bối cảnh liên quan hai đầu.
- Bổ sung phạm vi vào README, tệp 00–06, bảng rà soát, manifest và RULE_REGISTER; thêm câu đào tạo 13. Phạm vi được chốt không biến các quy tắc còn thiếu xác nhận thành đã được duyệt.
- Ngày bắt đầu/kết thúc thí điểm chưa được cung cấp; không dùng ngày xác nhận làm ngày triển khai lịch sử.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.10-before-cao-lanh-pilot/`.

### INV-02/AUTH-01 — Kiệt thao tác sau duyệt, v1.0.1-review.12

- Anh Khoa xác nhận nguyên văn ngày 30/10/2026: “Kiệt sẽ thao tác trên kiotviet”.
- Theo câu hỏi trước: Anh Khoa là người phê duyệt, Kiệt trực tiếp thực hiện điều chỉnh tồn trên KiotViet sau khi được duyệt, trong thí điểm Cao Lãnh.
- Đồng bộ trách nhiệm, timeline, truy vết, đáp án câu 12, từ điển, bảng rà soát và sổ quy tắc. Nơi lưu phê duyệt chưa được trả lời, vẫn để chờ; không tự gán nhóm Zalo hoặc chứng từ.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.11-before-kiet-executor/`.

### INV-02/AUTH-01 — Phê duyệt bằng lời nói trực tiếp, v1.0.1-review.13

- Anh Khoa xác nhận ngày 31/10/2026: hình thức phê duyệt điều chỉnh tồn cho Kiệt là “Nói trực tiếp”.
- Ghi nhận đúng luồng hiện hành: Anh Khoa nói trực tiếp để duyệt, Kiệt mới thao tác điều chỉnh tồn trên KiotViet. Không tự biến thành yêu cầu ký biên bản, nhắn Zalo hoặc duyệt trên hệ thống.
- Hình thức này không tạo dấu vết phê duyệt riêng để truy lại người duyệt, thời điểm và phạm vi điều chỉnh. Đây là giới hạn kiểm soát được ghi nhận, không phải kết luận rằng phê duyệt chưa xảy ra.
- Đồng bộ trách nhiệm, timeline, cẩm nang, đào tạo, từ điển, bảng rà soát và sổ quy tắc. Màn hình/chứng từ thao tác, lưu mốc kiểm và phương án tăng khả năng truy vết vẫn chưa chốt.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.12-before-verbal-approval/`.

### INV-02/AUTH-01 — Kiểm kho và Cân bằng kho trên KiotViet, v1.0.1-review.14

- Anh Khoa xác nhận ngày 31/10/2026: “Dùng chức năng kiểm kho của kiotviet, sau khi có sự đồng ý của anh khoa mới được cân bằng kho”.
- Chốt thao tác: Kiệt dùng chức năng **Kiểm kho** của KiotViet; hành động **Cân bằng kho** chỉ được thực hiện sau khi Anh Khoa đồng ý trực tiếp.
- Quyền đối chiếu số liệu hoặc bổ sung biên bản không đồng nghĩa quyền tự Cân bằng kho. Không thực hiện giao dịch KiotViet trong lần cập nhật tài liệu này.
- Đồng bộ vai trò, timeline, cẩm nang, đáp án đào tạo, quản trị dữ liệu, bảng rà soát và sổ quy tắc. Các điểm còn thiếu của INV-02 được thu hẹp còn lưu mốc, phát sinh nhập sau mốc, người đếm lại và dấu vết phê duyệt.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.13-before-kiotviet-stocktake-balance/`.

### INV-02 — Nhân viên kiểm lại khi có chênh lệch, v1.0.1-review.15

- Anh Khoa xác nhận ngày 31/10/2026: “Các bạn nhân viên sẽ kiểm lại khi có chênh lệch”.
- Chốt trách nhiệm ở mức vai trò: nhân viên kho kiểm đếm lại hàng thực tế khi phát hiện chênh lệch, trước luồng trình duyệt và Cân bằng kho.
- Không tự chỉ định một nhân viên cụ thể, người xác nhận cuối hoặc số lần đếm lại bắt buộc. Kết quả ban đầu và kết quả kiểm lại cần được phân biệt để không mất dấu vết số liệu.
- Đồng bộ nội quy, timeline, cẩm nang, đào tạo, quản trị dữ liệu, bảng rà soát và sổ quy tắc. Không thực hiện kiểm đếm hay giao dịch KiotViet trong lần cập nhật tài liệu.
- Sao lưu trước cập nhật tại `../.warehouse-kb-backups/v1.0.1-review.14-before-staff-recount/`.

## Câu hỏi đang trao đổi trước tiên

1. **Phạm vi đã chốt:** Thí điểm Cao Lãnh. TIME-01/INV-02 đã rõ lịch/phân công/mốc, bổ sung nhập, nhân viên đếm lại khi có chênh lệch, Anh Khoa nói trực tiếp để duyệt; Kiệt dùng chức năng Kiểm kho và chỉ Cân bằng kho sau khi được đồng ý. Cần hỏi tiếp lưu mốc kiểm, xử lý phát sinh sau mốc Kiệt lấy tồn, người xác nhận/số lần đếm lại và có cần tăng khả năng truy vết hay không.
2. **INV-01:** Anh muốn cảnh báo thiếu đáp ứng đơn đã cam kết hay thiếu hàng nhận đơn mới? `reserved` hiện giữ cho những phiếu nào?
3. **RET-01 (phần còn thiếu):** Ứng dụng/màn hình nhập lại tên gì, tạo chứng từ nào? Ai kiểm tra nhập lại, phê duyệt Giám đốc lưu ở đâu, nhóm hủy hàng tên gì? Phạm vi đã chốt là thí điểm Cao Lãnh.

## Các câu hỏi bổ sung, đi theo từng lượt

4. **AUTH-01:** Trong thí điểm Cao Lãnh, ai thay thế khi Phượng/Kiệt/thủ kho vắng và căn cứ ủy quyền được ghi ở đâu? Phạm vi Cao Lãnh đã được xác nhận tại SCOPE-01.
5. **TRANS-01:** Cao Lãnh → Bình Dương có chuyến hàng đóng Thứ Bảy và súp Chủ Nhật không? Lúc hàng đang đi đường, giao thiếu/thừa hoặc chưa xác nhận nhận thì xử lý thế nào?
6. **PACK-01/EXP-01/COLD-01:** Bao/bịch/thùng quy đổi ra sao theo từng mã? HSD theo nhãn nào; ngưỡng cận date và yêu cầu bảo quản hàng đông lạnh hiện do ai quy định?
7. **ORDER-01/HAND-01:** Lệnh thay đổi đơn, người duyệt, kiểm đếm độc lập và xác nhận chi nhánh nhận hàng đang được ghi trên App, phiếu ký hay nhóm Zalo nào?
8. **INV-02/REPORT-01:** Nhân viên đã được xác nhận là người đếm lại khi lệch; ai xác nhận kết quả và cần đếm lại bao nhiêu lần? Phê duyệt hiện bằng lời nói trực tiếp, có cần thêm dấu vết gắn với lần điều chỉnh không? Snapshot sáng có giữ riêng theo ngày không, báo cáo chiều dùng bản nào để so?
9. **PEOPLE-01:** Nội dung đánh giá 3 tháng và đạt 7/10 có phải quy định đã ban hành cho mọi nhân viên kho không, hay đang là đề xuất đào tạo?

## Điều kiện dữ liệu sẵn sàng để viết SOP

Các mục P0 có quyết định rõ; phạm vi từng kho, chứng từ, vai trò và ngày hiệu lực được ghi; các mốc/đơn vị được thống nhất; ví dụ nhập–xuất–trả–chuyển–kiểm tồn đối chiếu được chứng cứ. Những trường hợp còn chờ phải được liệt kê là ngoại lệ chưa chốt. Sau đó cập nhật tệp 00–05, bộ câu hỏi và sổ thay đổi cùng một phiên bản.

## Kiểm tra kỹ thuật của bản rà soát

- 12 tệp: 6 tài liệu nghiệp vụ, MANIFEST, README, 3 tài liệu bổ sung và RULE_REGISTER.
- Đã kiểm tra cú pháp JSON, danh mục/read-order trỏ đến tệp có thật, 15 mã vấn đề không trùng và có trong bảng rà soát; đọc UTF-8 nghiêm ngặt.
- Đối chiếu bản nguồn với sao lưu trước khi cập nhật, rồi so sánh từng tệp triển khai với bản staging. Nội dung quy tắc chưa chốt không được chuyển thành đã duyệt.
- Surface parity: gateway/API contract/web UI/CLI/desktop N/A vì lần này chỉ cập nhật tài liệu dữ liệu nghiệp vụ, không đổi mã nguồn, schema hay giao diện sản phẩm. Không cần build hoặc restart GoClaw.
