package protocol

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSendMessageWithMentionsUsesGroupMentionEndpoint(t *testing.T) {
	var gotPath string
	var gotPayload map[string]any
	sess := newMentionTestSession(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotPayload = decryptMentionRequest(t, sessSecretKey(t), r)
		writeMentionSuccess(t, w, sessSecretKey(t))
	})

	_, err := SendMessageWithMentions(context.Background(), sess, "g1", ThreadTypeGroup, "Chào @An", []*TMention{{UID: "u1", Pos: 5, Len: 3, Type: MentionEach}})
	if err != nil {
		t.Fatalf("SendMessageWithMentions() error: %v", err)
	}
	if gotPath != "/api/group/mention" {
		t.Fatalf("path = %q", gotPath)
	}
	mentionInfo, ok := gotPayload["mentionInfo"].(string)
	if !ok {
		t.Fatalf("mentionInfo = %#v", gotPayload["mentionInfo"])
	}
	var mentions []TMention
	if err := json.Unmarshal([]byte(mentionInfo), &mentions); err != nil {
		t.Fatal(err)
	}
	if len(mentions) != 1 || mentions[0].UID != "u1" || mentions[0].Pos != 5 || mentions[0].Len != 3 {
		t.Fatalf("mentions = %#v", mentions)
	}
}

func TestSendMessageWithMentionsKeepsPlainGroupAndDMPaths(t *testing.T) {
	paths := make([]string, 0, 2)
	sess := newMentionTestSession(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		writeMentionSuccess(t, w, sessSecretKey(t))
	})
	if _, err := SendMessageWithMentions(context.Background(), sess, "g1", ThreadTypeGroup, "plain", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := SendMessageWithMentions(context.Background(), sess, "u1", ThreadTypeUser, "DM", []*TMention{{UID: "u2", Pos: 0, Len: 2}}); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || paths[0] != "/api/group/sendmsg" || paths[1] != "/api/message/sms" {
		t.Fatalf("paths = %v", paths)
	}
}

func TestSendMessageWithMentionsReportsAPIError(t *testing.T) {
	sess := newMentionTestSession(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"error_code":114,"error_message":"invalid mention"}`)
	})
	if _, err := SendMessageWithMentions(context.Background(), sess, "g1", ThreadTypeGroup, "x", []*TMention{{UID: "u1", Pos: 0, Len: 1}}); err == nil {
		t.Fatal("expected API error")
	}
}

func TestSendMessageWithFormattingSendsNativeStyles(t *testing.T) {
	var gotPayload map[string]any
	sess := newMentionTestSession(t, func(w http.ResponseWriter, r *http.Request) {
		gotPayload = decryptMentionRequest(t, sessSecretKey(t), r)
		writeMentionSuccess(t, w, sessSecretKey(t))
	})
	_, err := SendMessageWithFormatting(context.Background(), sess, "g1", ThreadTypeGroup, "1. Tieu de", nil, []TextStyle{{Start: 0, Len: 10, Style: "c_15a85f"}, {Start: 0, Len: 10, Style: "b"}})
	if err != nil {
		t.Fatal(err)
	}
	properties, ok := gotPayload["textProperties"].(string)
	if !ok {
		t.Fatalf("textProperties = %#v", gotPayload["textProperties"])
	}
	var got struct {
		Styles []TextStyle `json:"styles"`
		Ver    int         `json:"ver"`
	}
	if err := json.Unmarshal([]byte(properties), &got); err != nil {
		t.Fatal(err)
	}
	if got.Ver != 0 || len(got.Styles) != 2 || got.Styles[0].Style != "c_15a85f" || got.Styles[1].Style != "b" {
		t.Fatalf("textProperties = %#v", got)
	}
}

func TestAddHeartReactionUsesNativeGroupEndpoint(t *testing.T) {
	var gotPayload map[string]any
	sess := newMentionTestSession(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/group/reaction" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		gotPayload = decryptMentionRequest(t, sessSecretKey(t), r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"error_code":0,"data":null}`)
	})
	sess.LoginInfo.ZpwServiceMapV3.Reaction = sess.LoginInfo.ZpwServiceMapV3.Group
	if err := AddHeartReaction(context.Background(), sess, "g1", ThreadTypeGroup, "123", "456"); err != nil {
		t.Fatal(err)
	}
	list, ok := gotPayload["react_list"].([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("react_list = %#v", gotPayload["react_list"])
	}
	entry := list[0].(map[string]any)
	var message struct {
		RMsg []struct {
			GMsgID int `json:"gMsgID"`
			CMsgID int `json:"cMsgID"`
		} `json:"rMsg"`
		RIcon string `json:"rIcon"`
		RType int    `json:"rType"`
	}
	if err := json.Unmarshal([]byte(entry["message"].(string)), &message); err != nil {
		t.Fatal(err)
	}
	if message.RIcon != "HEART" || message.RType != 5 || len(message.RMsg) != 1 || message.RMsg[0].GMsgID != 123 || message.RMsg[0].CMsgID != 456 {
		t.Fatalf("reaction message = %#v", message)
	}
}

func TestFetchGroupMembersUsesMemberIDsProfileFallback(t *testing.T) {
	paths := make([]string, 0, 2)
	sess := newMentionTestSession(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/group/getmg-v2":
			writeMentionData(t, w, sessSecretKey(t), `{"gridInfoMap":{"g1":{"memberIds":["u1"],"memVerList":["u1","u2"]}}}`)
		case "/api/social/group/members":
			writeMentionData(t, w, sessSecretKey(t), `{"profiles":{"u1":{"id":"u1","displayName":"Nguyễn A"},"u2":{"id":"u2","zaloName":"Bình"}}}`)
		default:
			t.Fatalf("unexpected endpoint %q", r.URL.Path)
		}
	})
	sess.LoginInfo.ZpwServiceMapV3.Profile = sess.LoginInfo.ZpwServiceMapV3.Group

	members, err := FetchGroupMembers(context.Background(), sess, "g1")
	if err != nil {
		t.Fatalf("FetchGroupMembers() error: %v", err)
	}
	if len(members) != 2 || members[0] != (GroupMemberInfo{UserID: "u1", DisplayName: "Nguyễn A"}) || members[1] != (GroupMemberInfo{UserID: "u2", DisplayName: "Bình"}) {
		t.Fatalf("members = %#v", members)
	}
	if len(paths) != 2 || paths[0] != "/api/group/getmg-v2" || paths[1] != "/api/social/group/members" {
		t.Fatalf("paths = %v", paths)
	}
}

func newMentionTestSession(t *testing.T, handler http.HandlerFunc) *Session {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	key := base64.StdEncoding.EncodeToString([]byte("0123456789ABCDEF"))
	return &Session{
		IMEI: "imei", UserAgent: "test", SecretKey: key, Client: server.Client(),
		LoginInfo: &LoginInfo{ZpwServiceMapV3: ZpwServiceMapV3{Chat: []string{server.URL}, Group: []string{server.URL}}},
	}
}

func sessSecretKey(t *testing.T) []byte {
	t.Helper()
	return []byte("0123456789ABCDEF")
}

func decryptMentionRequest(t *testing.T, key []byte, r *http.Request) map[string]any {
	t.Helper()
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	encoded := r.Form.Get("params")
	plain, err := DecodeAESCBC(key, encoded)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(plain, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func writeMentionSuccess(t *testing.T, w http.ResponseWriter, key []byte) {
	t.Helper()
	writeMentionData(t, w, key, `{"msgId":"42"}`)
}

func writeMentionData(t *testing.T, w http.ResponseWriter, key []byte, data string) {
	t.Helper()
	inner := `{"error_code":0,"data":` + data + `}`
	encrypted, err := EncodeAESCBC(key, inner, false)
	if err != nil {
		t.Fatal(err)
	}
	response, err := json.Marshal(map[string]any{"error_code": 0, "data": url.PathEscape(encrypted)})
	if err != nil {
		t.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(response)
}
