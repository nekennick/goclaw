package personal

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/nextlevelbuilder/goclaw/internal/channels/zalo/personal/protocol"
)

// 1. Tiêu đề đánh số có dấu chấm.
func TestChunking_1_NumberedHeadingWithDot(t *testing.T) {
	input := "## 1. Báo cáo kiểm tra định dạng Zalo\n\nNội dung chi tiết của phần một.\n\n## 2. Kế hoạch tiếp theo\n\nNội dung chi tiết của phần hai."
	res := renderZaloMessage(input, nil)

	// Verify structure captured headings
	if len(res.structure.Sections) < 2 {
		t.Fatalf("expected at least 2 sections, got %d", len(res.structure.Sections))
	}
	sec1 := res.structure.Sections[0]
	if sec1.HeadingBlock == nil || sec1.HeadingBlock.Kind != BlockHeadingMain {
		t.Fatalf("sec1 heading block missing or wrong kind: %#v", sec1.HeadingBlock)
	}

	chunks := splitMentionResolution(res, 2000)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk since total fits in 2000 bytes, got %d", len(chunks))
	}

	// Split with smaller limit to force division between sections
	chunksSplit := splitMentionResolution(res, len("1. Báo cáo kiểm tra định dạng Zalo\n\nNội dung chi tiết của phần một.")+10)
	if len(chunksSplit) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunksSplit))
	}
	if !strings.Contains(chunksSplit[0].text, "1. Báo cáo kiểm tra định dạng Zalo") {
		t.Fatalf("chunk 0 missing first heading: %q", chunksSplit[0].text)
	}
	if !strings.Contains(chunksSplit[1].text, "2. Kế hoạch tiếp theo") {
		t.Fatalf("chunk 1 missing second heading: %q", chunksSplit[1].text)
	}
}

// 2. Tiêu đề tiếng Việt dài hơn 35 byte.
func TestChunking_2_VietnameseHeadingLongerThan35Bytes(t *testing.T) {
	heading := "Quy định kiểm kho — phạm vi thí điểm kho Cao Lãnh"
	if len(heading) <= 35 {
		t.Fatalf("heading byte len must be > 35, got %d", len(heading))
	}
	input := "## " + heading + "\n\nNội dung hướng dẫn chi tiết quy trình kiểm kê kho."
	res := renderZaloMessage(input, nil)

	if len(res.structure.Sections) == 0 || res.structure.Sections[0].HeadingBlock == nil {
		t.Fatalf("heading was not recognized as a Section heading in structure: %#v", res.structure)
	}
	hBlock := res.structure.Sections[0].HeadingBlock
	if hBlock.Kind != BlockHeadingMain {
		t.Fatalf("expected BlockHeadingMain, got %v", hBlock.Kind)
	}
	hText := res.text[hBlock.StartByte:hBlock.EndByte]
	if !strings.Contains(hText, heading) {
		t.Fatalf("heading text mismatch: %q vs %q", hText, heading)
	}
}

// 3. Tiêu đề sát ranh giới tin phải chuyển cùng nội dung.
func TestChunking_3_HeadingNearBoundaryMovesWithContent(t *testing.T) {
	// Build Section 1 of ~1850 bytes
	sec1Body := strings.Repeat("Nội dung phần một kiểm tra định dạng. ", 50)
	sec1 := "## Phần mở đầu\n\n" + sec1Body

	// Section 2 starts near the 2000 byte boundary
	sec2Heading := "## Phần tiếp theo rất quan trọng"
	sec2Body := "Nội dung của phần tiếp theo được giữ trọn vẹn cùng tiêu đề."
	sec2 := sec2Heading + "\n\n" + sec2Body

	input := sec1 + "\n\n" + sec2
	res := renderZaloMessage(input, nil)

	chunks := splitMentionResolution(res, 2000)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// Chunk 1 MUST NOT end with the heading of Section 2!
	if strings.Contains(chunks[0].text, "Phần tiếp theo rất quan trọng") {
		t.Fatalf("chunk 0 illegally contains the heading of Section 2: %q", chunks[0].text)
	}

	// Chunk 2 MUST start with or contain the heading AND its content
	if !strings.Contains(chunks[1].text, "Phần tiếp theo rất quan trọng") {
		t.Fatalf("chunk 1 missing Section 2 heading: %q", chunks[1].text)
	}
	if !strings.Contains(chunks[1].text, sec2Body) {
		t.Fatalf("chunk 1 missing Section 2 body: %q", chunks[1].text)
	}
}

// 4. Mục vừa giới hạn được giữ nguyên. Nếu phần trống không đủ, chuyển cả mục sang tin kế tiếp.
func TestChunking_4_SectionWithinLimitPreservedWhole(t *testing.T) {
	sec1 := "## Mục 1\n\n" + strings.Repeat("Nội dung mục một vừa vặn. ", 20)
	sec2 := "## Mục 2\n\n" + strings.Repeat("Nội dung mục hai cũng vừa vặn. ", 20)
	input := sec1 + "\n\n" + sec2
	res := renderZaloMessage(input, nil)

	// Set limit such that each section fits individually, but sec1 + sec2 > limit
	limit := max(len(sec1), len(sec2)) + 50
	chunks := splitMentionResolution(res, limit)

	if len(chunks) != 2 {
		for i, c := range chunks {
			t.Logf("chunk %d (len %d): %q", i, len(c.text), c.text)
		}
		t.Fatalf("expected exactly 2 chunks, got %d", len(chunks))
	}
	if strings.Contains(chunks[0].text, "Mục 2") {
		t.Fatalf("Section 2 was split into chunk 0 instead of moving wholly to chunk 1: %q", chunks[0].text)
	}
	if !strings.Contains(chunks[1].text, "Mục 2") {
		t.Fatalf("chunk 1 does not contain Section 2: %q", chunks[1].text)
	}
}

// 5. Mục quá dài chia thành các đoạn hợp lý.
func TestChunking_5_OversizedSectionSplitsAtReasonableBlocks(t *testing.T) {
	sub1 := "### Mục con 1\n\n" + strings.Repeat("Đoạn văn mục con 1. ", 40)
	sub2 := "### Mục con 2\n\n" + strings.Repeat("Đoạn văn mục con 2. ", 40)
	sub3 := "### Mục con 3\n\n" + strings.Repeat("Đoạn văn mục con 3. ", 40)
	input := "## Mục lớn quá dài\n\n" + sub1 + "\n\n" + sub2 + "\n\n" + sub3
	res := renderZaloMessage(input, nil)

	// maxBytes = 1500 forces splitting across subsections
	chunks := splitMentionResolution(res, 1500)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	for i, chunk := range chunks {
		// Check that no chunk ends abruptly immediately after a heading
		lines := strings.Split(strings.TrimSpace(chunk.text), "\n")
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		if strings.HasPrefix(lastLine, "Mục con") || strings.HasPrefix(lastLine, "Mục lớn") {
			t.Fatalf("chunk %d ends with orphaned heading line %q", i, lastLine)
		}
	}
}

// 6. Cặp câu hỏi/đáp án không bị tách khi cả cặp vẫn vừa giới hạn.
func TestChunking_6_QAPairNotSplitWhenBothFit(t *testing.T) {
	intro := strings.Repeat("Giới thiệu quy trình kho bãi. ", 40)
	qaQuestion := "Câu hỏi: Khi hàng rách bao bì tài xế mang về kho xử lý thế nào?"
	qaAnswer := "Trả lời: Tài xế bàn giao trực tiếp cho thủ kho tại khu vực chờ phân loại."
	input := intro + "\n\n" + qaQuestion + "\n" + qaAnswer

	res := renderZaloMessage(input, nil)

	// Set limit so intro fits, but intro + QA does not fit entirely in chunk 0
	limit := len(intro) + 20
	chunks := splitMentionResolution(res, limit)

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// Verify that QA pair was moved together into chunk 1, not split in half
	if strings.Contains(chunks[0].text, "Câu hỏi:") {
		t.Fatalf("chunk 0 prematurely split question away from answer: %q", chunks[0].text)
	}
	if !strings.Contains(chunks[1].text, "Câu hỏi:") || !strings.Contains(chunks[1].text, "Trả lời:") {
		t.Fatalf("chunk 1 does not contain both question and answer: %q", chunks[1].text)
	}
}

// 7. Đoạn đơn vượt giới hạn: fallback theo câu, rồi từ; bảo toàn UTF-8.
func TestChunking_7_SingleParagraphExceedingLimit(t *testing.T) {
	// A giant single paragraph with 5 distinct sentences
	s1 := "Câu thứ nhất nói về quy trình kiểm kê kho vào thứ Tư và thứ Bảy hàng tuần lúc 13h30 đến 17h00. "
	s2 := "Câu thứ hai quy định việc kế toán chỉ xuất hóa đơn sau khi đã kiểm tra đầy đủ danh mục hàng hóa. "
	s3 := "Câu thứ ba yêu cầu tất cả nhân viên phải tuân thủ nghiêm ngặt quy tắc an toàn phòng cháy chữa cháy. "
	s4 := "Câu thứ tư nghiêm cấm hút thuốc và sử dụng lửa tự do trong toàn bộ khuôn viên tổng kho. "
	s5 := "Câu thứ năm chỉ định rõ người có thẩm quyền phê duyệt điều chỉnh số liệu tồn kho là Anh Khoa. "
	paragraph := s1 + s2 + s3 + s4 + s5

	res := renderZaloMessage(paragraph, nil)

	// Limit smaller than entire paragraph, e.g. enough for ~2 sentences
	limit := len(s1) + len(s2) + 20
	chunks := splitMentionResolution(res, limit)

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// Verify UTF-8 integrity of every chunk
	for i, c := range chunks {
		if !utf8.ValidString(c.text) {
			t.Fatalf("chunk %d contains invalid UTF-8 bytes", i)
		}
	}

	// Verify fallback split at sentence boundary: chunk 0 should end with sentence 2
	if !strings.Contains(chunks[0].text, "hàng hóa.") {
		t.Fatalf("chunk 0 was not cut at sentence boundary: %q", chunks[0].text)
	}
	if !strings.Contains(chunks[1].text, "Câu thứ ba") {
		t.Fatalf("chunk 1 does not begin with sentence 3: %q", chunks[1].text)
	}
}

// 8. Tiếng Việt, emoji, nhiều mention và style đi qua ranh giới.
func TestChunking_8_VietnameseEmojiMentionsStylesCrossingBoundary(t *testing.T) {
	members := []protocol.GroupMemberInfo{
		{UserID: "u1", DisplayName: "Trần Anh Khoa"},
		{UserID: "u2", DisplayName: "Nguyễn Thị Hoa"},
	}

	input := "Xin chào @[u1|Trần Anh Khoa] 🌟! " +
		"Thông báo quan trọng về đợt kiểm kho tổng hợp: ***khẩn cấp*** cần đối chiếu tồn kho ngay hôm nay. " +
		"Đồng chí @[u2|Nguyễn Thị Hoa] 🚀 sẽ phối hợp cùng tổ kiểm kê hoàn thành trước 17:00."

	res := renderZaloMessage(input, members)
	if len(res.mentions) != 2 {
		t.Fatalf("expected 2 mentions, got %d", len(res.mentions))
	}

	// Split near the middle
	limit := len(res.text) / 2
	chunks := splitMentionResolution(res, limit)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	totalMentions := 0
	for i, chunk := range chunks {
		if !utf8.ValidString(chunk.text) {
			t.Fatalf("chunk %d contains invalid UTF-8: %q", i, chunk.text)
		}
		totalMentions += len(chunk.mentions)
		for _, m := range chunk.mentions {
			start, end := byteRangeForUTF16(chunk.text, m.Pos, m.Len)
			if start < 0 || end > len(chunk.text) {
				t.Fatalf("mention out of range in chunk %d: %#v", i, m)
			}
			mText := chunk.text[start:end]
			if !strings.HasPrefix(mText, "@") {
				t.Fatalf("mention target in chunk %d does not start with @: %q", i, mText)
			}
		}
		for _, s := range chunk.styles {
			start, end := byteRangeForUTF16(chunk.text, s.Start, s.Len)
			if start < 0 || end > len(chunk.text) {
				t.Fatalf("style out of range in chunk %d: %#v", i, s)
			}
		}
	}

	if totalMentions != 2 {
		t.Fatalf("expected 2 mentions preserved across chunks, got %d", totalMentions)
	}
}

// 9. Ghép các chunk khôi phục đúng văn bản đầu vào của bộ chia.
func TestChunking_9_ReconstructionExactMatch(t *testing.T) {
	members := []protocol.GroupMemberInfo{
		{UserID: "u1", DisplayName: "Khoa"},
	}
	input := "## 1. Báo cáo kiểm kho\n\n" +
		"Kính gửi @[u1|Khoa] và các bạn thủ kho:\n\n" +
		"### Tình hình kiểm kê\n" +
		"• Mã hàng 01: Đạt tiêu chuẩn chất lượng 100%.\n" +
		"• Mã hàng 02: Cần kiểm đếm lại lúc ***16:00***.\n\n" +
		"Câu hỏi: Ai sẽ phê duyệt cân bằng kho?\n" +
		"Trả lời: Anh Khoa trực tiếp phê duyệt.\n\n" +
		"```text\nCODE_BLOCK_VERBATIM_123\n```\n\n" +
		"Chúc mọi người làm việc hiệu quả! 🎯"

	res := renderZaloMessage(input, members)

	// Test with several different maxBytes limits
	for _, limit := range []int{50, 100, 200, 350, 500, 1000} {
		chunks := splitMentionResolution(res, limit)
		var reconstructed strings.Builder
		for _, c := range chunks {
			reconstructed.WriteString(c.text)
		}
		if reconstructed.String() != res.text {
			t.Fatalf("reconstruction failed for limit %d:\nGOT: %q\nWANT: %q", limit, reconstructed.String(), res.text)
		}
	}
}

// 10. Không treo/vòng lặp khi gặp khối không thể giữ nguyên.
func TestChunking_10_NoInfiniteLoopOrHang(t *testing.T) {
	// Case A: 3000 bytes unbroken string (no spaces, no punctuation)
	unbroken := strings.Repeat("A", 3000)
	resA := renderZaloMessage(unbroken, nil)
	chunksA := splitMentionResolution(resA, 500)
	if len(chunksA) != 6 {
		t.Fatalf("expected 6 chunks for 3000 bytes with limit 500, got %d", len(chunksA))
	}
	reconstructedA := ""
	for _, c := range chunksA {
		reconstructedA += c.text
	}
	if reconstructedA != unbroken {
		t.Fatalf("unbroken string reconstruction mismatch")
	}

	// Case B: Extremely small limit (maxBytes = 1, 2)
	shortText := "Xin chào thế giới! 🇻🇳"
	resB := renderZaloMessage(shortText, nil)
	chunksB := splitMentionResolution(resB, 2)
	if len(chunksB) == 0 {
		t.Fatalf("expected chunks for limit 2, got 0")
	}
	reconstructedB := ""
	for _, c := range chunksB {
		if !utf8.ValidString(c.text) {
			t.Fatalf("invalid UTF-8 chunk produced: %v", []byte(c.text))
		}
		reconstructedB += c.text
	}
	if reconstructedB != shortText {
		t.Fatalf("limit 2 reconstruction mismatch: %q vs %q", reconstructedB, shortText)
	}

	// Case C: Mention display name exceeds maxBytes limit
	members := []protocol.GroupMemberInfo{
		{UserID: "u1", DisplayName: strings.Repeat("LongName", 20)},
	}
	mentionText := "@[u1|Name] bình thường"
	resC := renderZaloMessage(mentionText, members)
	chunksC := splitMentionResolution(resC, 10)
	if len(chunksC) < 2 {
		t.Fatalf("expected chunks for oversized mention, got %d", len(chunksC))
	}
	reconstructedC := ""
	for _, c := range chunksC {
		reconstructedC += c.text
	}
	if reconstructedC != resC.text {
		t.Fatalf("mention reconstruction mismatch")
	}
}
