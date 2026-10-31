package personal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
	"github.com/nextlevelbuilder/goclaw/internal/channels"
	"github.com/nextlevelbuilder/goclaw/internal/channels/zalo/personal/protocol"
)

func TestChannelSendConvertsInternalSyntaxToNativeMention(t *testing.T) {
	key := []byte("0123456789ABCDEF")
	var gotPath string
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/group/getmg-v2":
			writePersonalEncryptedResponse(t, w, key, `{"gridInfoMap":{"g1":{"currentMems":[{"id":"u1","dName":"Tran Anh Khoa"}]}}}`)
		case "/api/group/mention":
			gotPath = r.URL.Path
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			plain, err := protocol.DecodeAESCBC(key, r.Form.Get("params"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(plain, &gotPayload); err != nil {
				t.Fatal(err)
			}
			writePersonalEncryptedResponse(t, w, key, `{"msgId":"100"}`)
		default:
			t.Fatalf("unexpected endpoint %s", r.URL.Path)
		}
	}))
	defer server.Close()

	sess := &protocol.Session{
		SecretKey: base64.StdEncoding.EncodeToString(key), IMEI: "imei", UserAgent: "test", Client: server.Client(),
		LoginInfo: &protocol.LoginInfo{ZpwServiceMapV3: protocol.ZpwServiceMapV3{Group: []string{server.URL}}},
	}
	channel := &Channel{
		BaseChannel:  channels.NewBaseChannel(channels.TypeZaloPersonal, nil, nil),
		sess:         sess,
		groupMembers: make(map[string]groupMembersCacheEntry),
	}
	channel.SetRunning(true)
	channel.MarkGroupApproved("g1")

	err := channel.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "g1",
		Content: "Bao cao @[u1|fallback] da san sang.",
	})
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if gotPath != "/api/group/mention" {
		t.Fatalf("path = %q, want /api/group/mention", gotPath)
	}
	if gotPayload["message"] != "Bao cao @Tran Anh Khoa da san sang." {
		t.Fatalf("message = %#v", gotPayload["message"])
	}
	mentionInfo, ok := gotPayload["mentionInfo"].(string)
	if !ok {
		t.Fatalf("mentionInfo = %#v", gotPayload["mentionInfo"])
	}
	var mentions []protocol.TMention
	if err := json.Unmarshal([]byte(mentionInfo), &mentions); err != nil {
		t.Fatal(err)
	}
	if len(mentions) != 1 || mentions[0].UID != "u1" || mentions[0].Pos != 8 || mentions[0].Len != 14 || mentions[0].Type != protocol.MentionEach {
		t.Fatalf("mentions = %#v", mentions)
	}
}

func TestSendChunkedTextFetchesRosterAndSendsMention(t *testing.T) {
	key := []byte("0123456789ABCDEF")
	var gotPath string
	var gotMentionInfo string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/group/getmg-v2":
			writePersonalEncryptedResponse(t, w, key, `{"gridInfoMap":{"g1":{"currentMems":[{"id":"u1","dName":"Nguyễn A"}]}}}`)
		case "/api/group/mention":
			gotPath = r.URL.Path
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			plain, err := protocol.DecodeAESCBC(key, r.Form.Get("params"))
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if err := json.Unmarshal(plain, &payload); err != nil {
				t.Fatal(err)
			}
			gotMentionInfo, _ = payload["mentionInfo"].(string)
			writePersonalEncryptedResponse(t, w, key, `{"msgId":"99"}`)
		default:
			t.Fatalf("unexpected endpoint %s", r.URL.Path)
		}
	}))
	defer server.Close()

	sess := &protocol.Session{
		SecretKey: base64.StdEncoding.EncodeToString(key), IMEI: "imei", UserAgent: "test", Client: server.Client(),
		LoginInfo: &protocol.LoginInfo{ZpwServiceMapV3: protocol.ZpwServiceMapV3{Group: []string{server.URL}}},
	}
	channel := &Channel{sess: sess, groupMembers: make(map[string]groupMembersCacheEntry)}
	if err := channel.sendChunkedText(context.Background(), sess, "g1", protocol.ThreadTypeGroup, "Chào @[u1|bất kỳ]"); err != nil {
		t.Fatalf("sendChunkedText() error: %v", err)
	}
	if gotPath != "/api/group/mention" {
		t.Fatalf("path = %q", gotPath)
	}
	var mentions []protocol.TMention
	if err := json.Unmarshal([]byte(gotMentionInfo), &mentions); err != nil {
		t.Fatal(err)
	}
	if len(mentions) != 1 || mentions[0].UID != "u1" || mentions[0].Pos != 5 || mentions[0].Len != utf16Length("@Nguyễn A") {
		t.Fatalf("mentions = %#v", mentions)
	}
}

func TestSendChunkedTextWithMentionAndMarkdownStylesAlignsExactOffsets(t *testing.T) {
	key := []byte("0123456789ABCDEF")
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/group/getmg-v2":
			writePersonalEncryptedResponse(t, w, key, `{"gridInfoMap":{"g1":{"currentMems":[{"id":"1282516087395509750","dName":"Tran Anh Khoa"}]}}}`)
		case "/api/group/mention":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			plain, err := protocol.DecodeAESCBC(key, r.Form.Get("params"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(plain, &gotPayload); err != nil {
				t.Fatal(err)
			}
			writePersonalEncryptedResponse(t, w, key, `{"msgId":"101"}`)
		default:
			t.Fatalf("unexpected endpoint %s", r.URL.Path)
		}
	}))
	defer server.Close()

	sess := &protocol.Session{
		SecretKey: base64.StdEncoding.EncodeToString(key), IMEI: "imei", UserAgent: "test", Client: server.Client(),
		LoginInfo: &protocol.LoginInfo{ZpwServiceMapV3: protocol.ZpwServiceMapV3{Group: []string{server.URL}}},
	}
	channel := &Channel{sess: sess, groupMembers: make(map[string]groupMembersCacheEntry)}

	input := "@[1282516087395509750|Trần Anh Khoa] Đã đọc trực tiếp hai bộ dữ liệu.\n\n## Quy định kiểm kho\n\n- **Lịch kiểm kho**: Toàn bộ kho"
	if err := channel.sendChunkedText(context.Background(), sess, "g1", protocol.ThreadTypeGroup, input); err != nil {
		t.Fatalf("sendChunkedText() error: %v", err)
	}

	msg, _ := gotPayload["message"].(string)
	textPropsRaw, _ := gotPayload["textProperties"].(string)
	var textProps struct {
		Styles []protocol.TextStyle `json:"styles"`
	}
	if err := json.Unmarshal([]byte(textPropsRaw), &textProps); err != nil {
		t.Fatalf("unmarshal textProperties: %v", err)
	}

	foundHeading := false
	foundLabel := false
	for _, st := range textProps.Styles {
		start, end := byteRangeForUTF16(msg, st.Start, st.Len)
		if start >= 0 && end <= len(msg) {
			sub := msg[start:end]
			if sub == "Quy định kiểm kho" && st.Style == "c_15a85f" {
				foundHeading = true
			}
			if sub == "Lịch kiểm kho" && st.Style == "b" {
				foundLabel = true
			}
		}
	}
	if !foundHeading {
		t.Fatalf("did not find exact green style for 'Quy định kiểm kho', styles = %#v, message = %q", textProps.Styles, msg)
	}
	if !foundLabel {
		t.Fatalf("did not find exact bold style for 'Lịch kiểm kho', styles = %#v, message = %q", textProps.Styles, msg)
	}
}

func writePersonalEncryptedResponse(t *testing.T, w http.ResponseWriter, key []byte, data string) {
	t.Helper()
	inner := `{"error_code":0,"data":` + data + `}`
	encrypted, err := protocol.EncodeAESCBC(key, inner, false)
	if err != nil {
		t.Fatal(err)
	}
	response, err := json.Marshal(map[string]any{"error_code": 0, "data": url.PathEscape(encrypted)})
	if err != nil {
		t.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, string(response))
}
