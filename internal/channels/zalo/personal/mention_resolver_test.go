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
	plain, styles := stripMarkdownWithStyles("Lưu ý **Mặt sau** và __mã hàng__.")
	if plain != "Lưu ý Mặt sau và mã hàng." {
		t.Fatalf("plain = %q", plain)
	}
	if len(styles) != 4 {
		t.Fatalf("styles = %#v, want two styles per bold span", styles)
	}
	if styles[0].Style != "c_f27806" || styles[1].Style != "b" {
		t.Fatalf("first styles = %#v", styles[:2])
	}
	if styles[2].Start <= styles[0].Start {
		t.Fatalf("second span did not advance: %#v", styles)
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
