# 06. Quản trị dữ liệu và từ điển nghiệp vụ

Ngày rà soát: 30/10/2026. Phần này hướng dẫn tổ chức và diễn giải dữ liệu; không ban hành thêm thẩm quyền hay thay đổi quy trình vận hành.

**Phạm vi vận hành đã xác nhận (SCOPE-01):** Quy trình đang thí điểm tại **kho Cao Lãnh**. Khi AI trích quy tắc kiểm kho, xử lý đơn, hàng trả hoặc thẩm quyền, phải giữ phạm vi thí điểm này. Không tự suy rộng sang Bình Dương; nhắc kho đó trong luồng chuyển hàng không phải bằng chứng đã áp dụng quy trình. Ngày xác nhận phạm vi không phải ngày bắt đầu thí điểm; ngày bắt đầu chưa được cung cấp.

## 1. Trạng thái và nguồn xác nhận

| Trạng thái | Ý nghĩa | Cách sử dụng |
|---|---|---|
| GHI_NHAN_TU_BO_GOC | Có trong tệp 00–05, chưa có hồ sơ xác nhận chi tiết trong lần rà soát này | Trích rõ nguồn; không gọi là vừa được chủ nghiệp vụ duyệt |
| DA_XAC_NHAN | Có câu trả lời trực tiếp của chủ nghiệp vụ, ghi lại phạm vi xác nhận | Chỉ áp dụng đúng phần đã xác nhận |
| CAN_XAC_NHAN | Thiếu thông tin, có mâu thuẫn hoặc cần bằng chứng | Hỏi lại trước khi kết luận hoặc viết SOP chính thức |
| DE_XUAT | Phương án cải thiện, chưa được duyệt | Trình bày như đề xuất |
| NGUNG_AP_DUNG | Có quyết định thay thế và mốc hiệu lực | Giữ lịch sử, không áp dụng cho thời điểm mới |

Một vấn đề có thể được xác nhận một phần. Ví dụ biết cách xử lý công nợ không có nghĩa đã biết chứng từ xử lý tồn. Xem `RULE_REGISTER.json` để biết trạng thái từng phần.

Thông tin một quy tắc cần có: mã quy tắc, phát biểu rõ, kho áp dụng, vai trò chịu trách nhiệm, nguồn/bằng chứng, người xác nhận, ngày xác nhận, ngày hiệu lực và quy tắc được thay thế (nếu có). Chưa biết thì ghi `null` hoặc “chưa xác nhận”; không lấy ngày tạo tệp làm ngày có hiệu lực.

## 2. Định danh và liên kết

| Đối tượng | Khóa nhận diện | Điểm cần tránh |
|---|---|---|
| Khách hàng/chi nhánh nhận hàng | `customerCode`, đối chiếu thêm ID hệ thống nếu có | Không gộp theo tên gần giống; đổi tên phải có ánh xạ và bằng chứng |
| Kho xuất/nhận | Mã/ID kho đã đối chiếu với danh mục KiotViet và tên kho vận hành | Không dùng `branchId` của hồ sơ khách hàng để suy ra tuyến/kho phục vụ |
| Sản phẩm | Mã hàng và ID sản phẩm; kèm đơn vị tính | Không gộp mã cũ/mới chỉ vì tên giống nhau; không coi hoa/thường là hai mặt hàng khi quy tắc đã cho phép chuẩn hóa |
| Đơn đặt hàng và hóa đơn | ID/mã chứng từ; quan hệ hóa đơn–đơn theo dữ liệu thực | Không coi đơn đặt hàng là hàng đã giao; không chỉ ghép bằng tên khách/ngày |
| Phiếu chuyển kho | Mã phiếu, kho nguồn, kho đích, trạng thái và mốc giao/nhận | Không cộng hàng đang đi đường vào cả hai đầu như tồn sẵn sàng |
| Lô/hạn dùng | Mã hàng + lô + NSX/HSD nếu được ghi nhận | Không tự tính lại HSD khi chưa đối chiếu nhãn đúng sản phẩm/lô |

Nguồn lịch giao hiện được script đọc: `../yagami-knowledge-base/yagami-branch-warehouse-map.json`. File này đang ghi version 11, `verifiedAt=2026-09-24`; manifest cạnh nó đang ghi version 10 và mốc khác. Đây là phát hiện lệch metadata tại lần đọc 30/10, không phải xác nhận lịch vẫn đúng cho mọi ngày sau đó. Cần lưu lịch có hiệu lực theo ngày để phân tích lịch sử; không áp lịch hiện tại ngược về quá khứ.

## 3. Phân biệt số lượng và bằng chứng

| Thuật ngữ | Diễn giải khi đọc dữ liệu | Điều kiện trước khi kết luận |
|---|---|---|
| `onHand` | Trường tồn trên phần mềm tại mốc đồng bộ | Không gọi là số đếm vật lý nếu chưa có biên bản kiểm đếm |
| Tồn đếm thực tế | Số đếm tại từng vị trí và thời điểm cụ thể | Có mã hàng, đơn vị, người đếm, giờ đếm, phạm vi và phát sinh quanh mốc |
| `reserved` | Trường “giữ hàng” nguồn trả về | Cần xác minh loại phiếu và phạm vi nhu cầu được bao gồm |
| `available` | Script hiện tính `onHand - reserved` | Không mặc nhiên là hàng trống trên kệ; không trừ lại đơn đã nằm trong `reserved` |
| “Cần” | Tổng nhu cầu của tập đơn được chọn | Ghi rõ đơn theo lịch/xe ngoài/ngoài lịch, trạng thái và đơn vị |
| “Thiếu” | Kết luận cần định nghĩa và bằng chứng | Phân biệt thiếu giao đơn hiện tại với không còn hàng nhận thêm đơn |
| Hóa đơn hoàn thành | Bằng chứng trạng thái chứng từ trên hệ thống | Không đồng nghĩa đã có xác nhận người nhận tại chi nhánh |
| Hoàn tất xử lý đơn khách | Đã xuất hóa đơn từ những đơn đặt (phiếu tạm), theo xác nhận Anh Khoa 30/10/2026 | Đối chiếu đúng đơn–hóa đơn; không dùng hóa đơn không liên quan làm bằng chứng và không gọi đây là đã giao hàng |
| Điều chỉnh công nợ | Ghi nhận tài chính | Không suy ra tồn đã cập nhật hoặc hàng đã được phân loại |

**Ví dụ kiểm tra nguy cơ tính hai lần (chưa chốt công thức nghiệp vụ):** tồn 100, giữ 80 (giả sử đã gồm nhu cầu hôm nay 60), khả dụng 20. Phép `60 - 20 = 40` không đủ để khẳng định đơn hôm nay thiếu 40. Cần biết chính xác 80 đang giữ cho những đơn nào. Script hiện có phép `need - available`; chờ xác nhận INV-01 trước khi đổi báo cáo.

## 4. Mốc dữ liệu tối thiểu

**Mốc nghiệp vụ đã xác nhận (TIME-01):** Tất cả đơn khách đặt trong ngày phải được xuất hóa đơn từ đơn đặt (phiếu tạm) trước 16:00; đó là hoàn tất xử lý đơn. Không sử dụng mốc này làm thời điểm kiểm đếm vật lý. Các ngoại lệ/đối chiếu chứng từ và lịch kiểm tồn theo dõi riêng tại ORDER-01/REPORT-01 và INV-02.

**Lịch kiểm kho đã xác nhận (INV-02):** Trong thí điểm **kho Cao Lãnh**, thứ Tư và thứ Bảy, 13:30–17:00, kiểm toàn bộ kho. Vẫn nhập, không xuất hàng thực tế; nhân viên đếm theo khu gửi Kiệt. Kế toán xuất hóa đơn hàng đi sáng trước 16:00; **Kiệt lấy tồn KiotViet sau 16:00 làm mốc đối chiếu số thực tế**. Phạm vi toàn kho là toàn bộ hàng tại kho thí điểm, không có nghĩa cả hai địa điểm kho.

**Hai thời điểm cần phân biệt:** Giờ nhân viên đếm khu mình và giờ Kiệt lấy tồn sau 16:00 có thể khác nhau. Mốc đối chiếu phải là số KiotViet Kiệt lấy sau 16:00, không phải số phần mềm lấy trước khi kế toán hoàn tất xuất hóa đơn. Chưa có giờ lấy tồn chính xác nên không quy định cứng 16:00, 16:01 hoặc 17:00. Đề xuất lưu ảnh/tệp số liệu và giờ lấy tồn để truy vết; hình thức lưu này còn chờ duyệt.

**Bổ sung hàng nhập sau đếm đã xác nhận:** Nhân viên cộng theo số nhận thực tế do thủ kho báo và gửi Kiệt. Nếu chênh lệch lớn khớp thông báo nhận hàng trên nhóm nhận hàng, Kiệt tự bổ sung số nhập vào biên bản kiểm kho. Không tự gán ngưỡng định lượng cho “chênh lệch lớn”.

**Toàn vẹn dữ liệu:** Số đếm gốc, phần cộng do hàng nhập và số tổng sau bổ sung là các giá trị khác nhau cần phân biệt. Cùng một lượng nhận thực tế chỉ cộng một lần, không cộng đồng thời cả phần nhân viên đã gửi và phần Kiệt nhận diện lại. Việc bổ sung biên bản không phải bằng chứng đã điều chỉnh tồn KiotViet.

**Kiểm lại chênh lệch đã xác nhận:** Khi có chênh lệch, nhân viên kho kiểm đếm lại hàng thực tế. Cần giữ riêng số đếm ban đầu và số đếm lại để truy vết, không ghi đè làm mất lịch sử. Chưa xác nhận người cụ thể chịu trách nhiệm từng lần, người xác nhận kết quả hoặc số lần đếm lại bắt buộc nên không tự đặt các điều kiện này.

**Đề xuất cách lưu bằng chứng, còn chờ chốt:** Ghi mã hàng/đơn vị, số đếm gốc, lượng nhập bổ sung, người bổ sung, thông báo thủ kho và mốc lấy tồn Kiệt đang dùng. Khi phát sinh nhập sau mốc lấy tồn cần chốt cách cập nhật mốc đối chiếu để hai phía cùng phạm vi; chưa tự áp dụng một công thức cho mọi phát sinh sau mốc.

“Khớp với số liệu kho mới nhất” là mục tiêu đối chiếu. Khi có lệch, hồ sơ phải phản ánh số thực đếm và chênh lệch để truy vết; không diễn giải thành sửa số đếm cho bằng phần mềm hoặc tự cấp quyền điều chỉnh tồn.

**Thẩm quyền, chức năng và cách duyệt đã xác nhận:** Trong thí điểm Cao Lãnh, Anh Khoa phê duyệt điều chỉnh tồn KiotViet khi đối soát vẫn lệch bằng cách **nói trực tiếp với Kiệt**. Kiệt dùng chức năng **Kiểm kho** của KiotViet và chỉ thực hiện **“Cân bằng kho” sau khi Anh Khoa đồng ý**, theo xác nhận cập nhật 31/10/2026. Phân biệt người bổ sung biên bản, người duyệt và người thao tác; không suy ra quyền tự duyệt hoặc tự cân bằng kho chỉ từ quyền đối chiếu hay sửa biên bản.

**Giới hạn truy vết của cách hiện hành:** Phê duyệt bằng lời nói trực tiếp không tạo bản ghi bền vững để kiểm tra ai duyệt, lúc nào và nội dung điều chỉnh nào. Knowledge Base ghi nhận đúng hiện trạng, chưa biến đề xuất lưu dấu vết thành quy định bắt buộc. Nếu sau này cần kiểm soát chặt hơn, chủ quy trình phải duyệt riêng phương thức ghi nhận.

Mỗi snapshot/báo cáo cần nêu: thời điểm chạy, thời điểm bắt đầu/kết thúc đồng bộ, dữ liệu mới nhất đã nhận, kỳ lọc chứng từ, phiên bản lịch giao và múi giờ. Với đếm kho cần thêm giờ đếm và trạng thái xử lý phát sinh. Không có snapshot sáng thì không dùng dữ liệu chiều để giả lập “dự kiến lúc sáng”.

Thiếu giá trị, API lỗi, danh mục chưa khớp hoặc đồng bộ dở dang phải ghi là chưa đủ dữ liệu; không thay bằng 0. Một tiến trình chạy xong không chứng minh dữ liệu đúng, cũng không chứng minh báo cáo đã đến nhóm Zalo.

## 5. Mẫu hồ sơ trước khi viết SOP

Đây là các trường đề xuất để thu thập, chưa phải biểu mẫu đang bắt buộc tại kho:

- Nhập hàng: mã chứng từ nguồn, kho, mã hàng/đơn vị, số chứng từ, số đếm, lô/NSX/HSD khi có, chất lượng, chênh lệch, người nhận/kiểm/duyệt, liên kết ảnh.
- Xuất hàng: customerCode, mã đơn/hóa đơn, tuyến xe, số soạn/kiểm/bàn giao, người giao/nhận, phát sinh đã duyệt.
- Hàng trả: liên kết hóa đơn gốc và lần trả, mã hàng/đơn vị/số lượng, tình trạng, nơi cách ly, kết quả thẩm định; tách chứng từ công nợ với chứng từ tồn và người phụ trách từng phần.
- Kiểm tồn: kho/vị trí, mã hàng/đơn vị, mốc đếm, số đếm lần đầu/lần đếm lại, tồn phần mềm cùng mốc, chứng từ lệch, người xác nhận và trạng thái xử lý.
- Thay đổi đơn sau cut-off: mã đơn, dòng hàng trước/sau, lý do, người đề nghị, người duyệt, giờ duyệt, người thực hiện và bằng chứng kho đã nhận thay đổi.

## 6. Giới hạn dùng cho AI

AI được tổng hợp và phát hiện chênh lệch dựa trên dữ liệu đã nhận. Nội dung KB không tự cấp quyền điều chỉnh tồn/công nợ, duyệt hủy hay thay đổi chứng từ. Mục cần xác nhận phải được nêu là chưa chốt. Chỉ đổi quy tắc và đáp án đào tạo sau khi ghi nhận xác nhận có phạm vi rõ ràng.
