package personal

import (
	"strings"
	"testing"

	"github.com/nextlevelbuilder/goclaw/internal/channels/zalo/personal/protocol"
)

func TestRenderZaloMessage_CaseA_MentionPrecededByMarkdown(t *testing.T) {
	members := []protocol.GroupMemberInfo{
		{UserID: "u1", DisplayName: "Tran Anh Khoa"},
	}
	input := "**Lưu ý** @[u1|Khoa] đã nhận báo cáo."
	res := renderZaloMessage(input, members)

	expectedText := "Lưu ý @Tran Anh Khoa đã nhận báo cáo."
	if res.text != expectedText {
		t.Fatalf("text = %q, want %q", res.text, expectedText)
	}

	if len(res.mentions) != 1 {
		t.Fatalf("mentions count = %d, want 1", len(res.mentions))
	}
	m := res.mentions[0]
	mStart, mEnd := byteRangeForUTF16(res.text, m.Pos, m.Len)
	subMention := res.text[mStart:mEnd]
	if subMention != "@Tran Anh Khoa" {
		t.Fatalf("mention target = %q, want '@Tran Anh Khoa'", subMention)
	}

	// Verify bold style points to "Lưu ý"
	foundBold := false
	for _, st := range res.styles {
		if st.Style == "b" {
			start, end := byteRangeForUTF16(res.text, st.Start, st.Len)
			sub := res.text[start:end]
			if sub == "Lưu ý" {
				foundBold = true
			}
		}
	}
	if !foundBold {
		t.Fatalf("did not find exact bold style for 'Lưu ý': %#v", res.styles)
	}
}

func TestRenderZaloMessage_CaseB_RepeatedWords(t *testing.T) {
	input := "Cần kiểm kho.\n\n- **kiểm kho**: đối chiếu."
	res := renderZaloMessage(input, nil)

	if !strings.Contains(res.text, "Cần kiểm kho.\n\n• kiểm kho: đối chiếu.") {
		t.Fatalf("text = %q", res.text)
	}

	// The bold style MUST point to the second "kiểm kho", NOT the first!
	foundSecondBold := false
	for _, st := range res.styles {
		if st.Style == "b" {
			start, end := byteRangeForUTF16(res.text, st.Start, st.Len)
			sub := res.text[start:end]
			if sub == "kiểm kho" {
				if st.Start >= 17 {
					foundSecondBold = true
				} else {
					t.Fatalf("bold style falsely matched the first occurrence: Start = %d", st.Start)
				}
			}
		}
	}
	if !foundSecondBold {
		t.Fatalf("did not find bold style on second 'kiểm kho': %#v", res.styles)
	}
}

func TestRenderZaloMessage_CaseC_LongVietnameseLabelAndPartialBold(t *testing.T) {
	input := "- **Người được phép phê duyệt Cân bằng kho**: Anh Khoa\n- Mốc **đối chiếu**: sau 16h"
	res := renderZaloMessage(input, nil)

	// Verify "Người được phép phê duyệt Cân bằng kho" is fully bold
	foundLongLabel := false
	foundPartialMerged := false
	for _, st := range res.styles {
		if st.Style == "b" {
			start, end := byteRangeForUTF16(res.text, st.Start, st.Len)
			sub := res.text[start:end]
			if sub == "Người được phép phê duyệt Cân bằng kho" {
				foundLongLabel = true
			}
			if sub == "Mốc đối chiếu" {
				foundPartialMerged = true
			}
			// Colon must never be bolded!
			if strings.HasSuffix(sub, ":") {
				t.Fatalf("colon was included in bold style: %q", sub)
			}
		}
	}

	if !foundLongLabel {
		t.Fatalf("long label was not bolded or missed: %#v", res.styles)
	}
	if !foundPartialMerged {
		t.Fatalf("partial bold was not merged into full label 'Mốc đối chiếu': %#v", res.styles)
	}
}

func TestRenderZaloMessage_CaseD_StyleCrossingChunkBoundary(t *testing.T) {
	// Create a message with a bold block that will cross a 100-byte chunk boundary
	prefix := strings.Repeat("a", 80)
	boldContent := strings.Repeat("b", 40)
	suffix := strings.Repeat("c", 80)
	input := prefix + "**" + boldContent + "**" + suffix

	res := renderZaloMessage(input, nil)
	chunks := splitMentionResolution(res, 100)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// Verify Chunk 1 has a bold style for the first part of boldContent
	hasBoldChunk1 := false
	for _, st := range chunks[0].styles {
		if st.Style == "b" && st.Len > 0 {
			hasBoldChunk1 = true
		}
	}
	// Verify Chunk 2 has a bold style for the remaining part of boldContent
	hasBoldChunk2 := false
	for _, st := range chunks[1].styles {
		if st.Style == "b" && st.Len > 0 {
			hasBoldChunk2 = true
		}
	}

	if !hasBoldChunk1 {
		t.Fatalf("chunk 1 lost bold style on boundary: %#v", chunks[0].styles)
	}
	if !hasBoldChunk2 {
		t.Fatalf("chunk 2 lost bold style on boundary: %#v", chunks[1].styles)
	}
}

func TestRenderZaloMessage_CaseE_HeadingMarkdownAndNumberingOnlyOneColor(t *testing.T) {
	input := "### 1. Kiểm kho"
	res := renderZaloMessage(input, nil)

	// Heading 3 is Orange Bold (c_f27806). It must NOT also have Green (c_15a85f).
	hasOrange := false
	hasGreen := false
	for _, st := range res.styles {
		if st.Style == "c_f27806" {
			hasOrange = true
		}
		if st.Style == "c_15a85f" {
			hasGreen = true
		}
	}

	if !hasOrange {
		t.Fatalf("expected orange style for H3, got styles: %#v", res.styles)
	}
	if hasGreen {
		t.Fatalf("conflicting green style was erroneously added to H3: %#v", res.styles)
	}
}

func TestRenderZaloMessage_CaseF_CodeBlocksAreVerbatim(t *testing.T) {
	members := []protocol.GroupMemberInfo{
		{UserID: "u1", DisplayName: "Tran Anh Khoa"},
	}
	input := "```go\n**not bold** @[u1|Khoa]\n```"
	res := renderZaloMessage(input, members)

	if !strings.Contains(res.text, "**not bold** @[u1|Khoa]") {
		t.Fatalf("code block text was altered: %q", res.text)
	}
	if len(res.mentions) != 0 {
		t.Fatalf("mention was erroneously resolved inside code block: %#v", res.mentions)
	}
	if len(res.styles) != 0 {
		t.Fatalf("style was erroneously applied inside code block: %#v", res.styles)
	}
}

func TestRenderZaloMessage_CaseG_MultipleMentionsWithVietnameseAndEmoji(t *testing.T) {
	members := []protocol.GroupMemberInfo{
		{UserID: "u1", DisplayName: "Tran Anh Khoa"},
		{UserID: "u2", DisplayName: "Nguyen Thi Hoa"},
	}
	input := "Chào @[u1|Khoa] 🎉 và @[u2|Hoa] 🚀!"
	res := renderZaloMessage(input, members)

	expectedText := "Chào @Tran Anh Khoa 🎉 và @Nguyen Thi Hoa 🚀!"
	if res.text != expectedText {
		t.Fatalf("text = %q, want %q", res.text, expectedText)
	}
	if len(res.mentions) != 2 {
		t.Fatalf("mentions count = %d, want 2", len(res.mentions))
	}

	start1, end1 := byteRangeForUTF16(res.text, res.mentions[0].Pos, res.mentions[0].Len)
	if res.text[start1:end1] != "@Tran Anh Khoa" {
		t.Fatalf("first mention = %q", res.text[start1:end1])
	}

	start2, end2 := byteRangeForUTF16(res.text, res.mentions[1].Pos, res.mentions[1].Len)
	if res.text[start2:end2] != "@Nguyen Thi Hoa" {
		t.Fatalf("second mention = %q", res.text[start2:end2])
	}
}

func TestRenderZaloMessage_MentionFollowedByHeadingOnSameLine(t *testing.T) {
	members := []protocol.GroupMemberInfo{
		{UserID: "260898693251376548", DisplayName: "Trần Anh Khoa"},
	}
	input := "@[260898693251376548|Trần Anh Khoa] ## 1. Báo cáo kiểm tra định dạng Zalo\n\n### Tình hình kiểm kho"
	res := renderZaloMessage(input, members)

	expectedPrefix := "@Trần Anh Khoa\n\n1. Báo cáo kiểm tra định dạng Zalo\n\nTình hình kiểm kho"
	if res.text != expectedPrefix {
		t.Fatalf("text = %q, want %q", res.text, expectedPrefix)
	}

	if len(res.mentions) != 1 {
		t.Fatalf("mentions count = %d, want 1", len(res.mentions))
	}
	if res.mentions[0].Pos != 0 {
		t.Fatalf("mention pos = %d, want 0", res.mentions[0].Pos)
	}

	foundGreen := false
	for _, st := range res.styles {
		if st.Style == "c_15a85f" {
			start, end := byteRangeForUTF16(res.text, st.Start, st.Len)
			sub := res.text[start:end]
			if sub == "1. Báo cáo kiểm tra định dạng Zalo" {
				foundGreen = true
			}
		}
	}
	if !foundGreen {
		t.Fatalf("heading '1. Báo cáo kiểm tra định dạng Zalo' did not receive green style: %#v", res.styles)
	}
}

