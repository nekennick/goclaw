package personal

import (
	"strings"
	"testing"

	"github.com/nextlevelbuilder/goclaw/internal/channels/zalo/personal/protocol"
)

func TestResolveNativeMentions(t *testing.T) {
	members := []protocol.GroupMemberInfo{
		{UserID: "u1", DisplayName: "Nguyễn Văn A"},
		{UserID: "u2", DisplayName: "Bình"},
	}
	got := resolveNativeMentions("Chào @[u1|A] và @[u2|Bình] 👋", members)
	if got.text != "Chào @Nguyễn Văn A và @Bình 👋" {
		t.Fatalf("text = %q", got.text)
	}
	if len(got.mentions) != 2 {
		t.Fatalf("mentions = %d, want 2", len(got.mentions))
	}
	if got.mentions[0].UID != "u1" || got.mentions[0].Pos != 5 || got.mentions[0].Len != utf16Length("@Nguyễn Văn A") {
		t.Fatalf("first mention = %#v", got.mentions[0])
	}
	if got.mentions[1].UID != "u2" || got.mentions[1].Pos != utf16Length("Chào @Nguyễn Văn A và ") {
		t.Fatalf("second mention = %#v", got.mentions[1])
	}
}

func TestResolveNativeMentionsFailsClosed(t *testing.T) {
	members := []protocol.GroupMemberInfo{
		{UserID: "u1", DisplayName: "Trùng Tên"},
		{UserID: "u2", DisplayName: "Trùng Tên"},
	}
	got := resolveNativeMentions("@[missing|Giả] @[u1|Trùng Tên] @[ -1|Tất cả]", members)
	if got.text != "@Giả @Trùng Tên @Tất cả" {
		t.Fatalf("text = %q", got.text)
	}
	if len(got.mentions) != 0 {
		t.Fatalf("mentions = %#v, want none", got.mentions)
	}
}

func TestSplitMentionResolutionDoesNotCutMention(t *testing.T) {
	resolved := resolveNativeMentions(strings.Repeat("x", 14)+" @[u1|Nguyễn A] "+strings.Repeat("y", 20), []protocol.GroupMemberInfo{{UserID: "u1", DisplayName: "Nguyễn A"}})
	chunks := splitMentionResolution(resolved, 20)
	if len(chunks) < 2 {
		t.Fatalf("chunks = %d, want multiple", len(chunks))
	}
	for _, chunk := range chunks {
		if strings.Contains(chunk.text, "@Nguyễn A") && len(chunk.mentions) != 1 {
			t.Fatalf("mention was rendered without metadata: %#v", chunk)
		}
		for _, mention := range chunk.mentions {
			start, end := byteRangeForUTF16(chunk.text, mention.Pos, mention.Len)
			if start < 0 || end <= start || chunk.text[start:end] != "@Nguyễn A" {
				t.Fatalf("invalid re-based mention %#v for %q", mention, chunk.text)
			}
		}
	}
}

func TestAddZaloReportStyles(t *testing.T) {
	styled := addZaloReportStyles(mentionResolution{text: "1. Ket qua chinh\nNoi dung\n📦 Chi tiet kho"})
	if len(styled.styles) != 4 {
		t.Fatalf("styles = %#v", styled.styles)
	}
	if styled.styles[0].Style != "c_15a85f" || styled.styles[1].Style != "b" || styled.styles[2].Style != "c_f27806" || styled.styles[3].Style != "b" {
		t.Fatalf("styles = %#v", styled.styles)
	}
}

func TestStripMarkdownWithStyles(t *testing.T) {
	input := "## Thông tin tổng quan\n### Phân cảnh 1\nLưu ý **mặt sau** và ***điểm cốt lõi***."
	plain, styles := stripMarkdownWithStyles(input)
	if !strings.Contains(plain, "Thông tin tổng quan") || !strings.Contains(plain, "mặt sau") || !strings.Contains(plain, "điểm cốt lõi") {
		t.Fatalf("plain = %q", plain)
	}

	// H2 -> green + bold
	// H3 -> orange + bold
	// **mặt sau** -> bold
	// ***điểm cốt lõi*** -> orange + bold
	hasGreen := false
	hasOrange := false
	hasBold := false
	for _, st := range styles {
		if st.Style == "c_15a85f" {
			hasGreen = true
		}
		if st.Style == "c_f27806" {
			hasOrange = true
		}
		if st.Style == "b" {
			hasBold = true
		}
	}
	if !hasGreen || !hasOrange || !hasBold {
		t.Fatalf("missing expected styles: green=%v, orange=%v, bold=%v, styles=%#v", hasGreen, hasOrange, hasBold, styles)
	}
}

func TestSplitMentionResolutionPrefersNaturalBoundaries(t *testing.T) {
	text := "Mot abcde tiep fghij tiep klmno tiep nopqr"
	chunks := splitMentionResolution(mentionResolution{text: text}, 17)
	if len(chunks) < 2 {
		t.Fatalf("chunks = %d, want multiple", len(chunks))
	}
	var rebuilt strings.Builder
	for i, chunk := range chunks {
		rebuilt.WriteString(chunk.text)
		if i < len(chunks)-1 && !strings.HasSuffix(chunk.text, " ") && !strings.HasSuffix(chunk.text, "\n") {
			t.Fatalf("chunk %d ends inside a word: %q", i, chunk.text)
		}
	}
	if rebuilt.String() != text {
		t.Fatalf("rebuilt = %q, want %q", rebuilt.String(), text)
	}
}

func TestSplitMentionResolutionAvoidsTinyFinalBubble(t *testing.T) {
	text := strings.Repeat("dong bao cao co du lieu\n", 24) + "Ket thuc."
	chunks := splitMentionResolution(mentionResolution{text: text}, 500)
	if len(chunks) < 2 {
		t.Fatalf("chunks = %d, want multiple", len(chunks))
	}
	if got := len(chunks[len(chunks)-1].text); got < 80 {
		t.Fatalf("final chunk is an unreadable fragment (%d bytes): %q", got, chunks[len(chunks)-1].text)
	}
	var rebuilt strings.Builder
	for _, chunk := range chunks {
		rebuilt.WriteString(chunk.text)
	}
	if rebuilt.String() != text {
		t.Fatalf("rebuilt = %q, want %q", rebuilt.String(), text)
	}
}
