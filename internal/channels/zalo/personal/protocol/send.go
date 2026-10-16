package protocol

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SendMessage sends a text message to a user or group.
// threadID: user UID (DM) or group ID (group).
func SendMessage(ctx context.Context, sess *Session, threadID string, threadType ThreadType, text string) (string, error) {
	return SendMessageWithMentions(ctx, sess, threadID, threadType, text, nil)
}

// SendMessageWithMentions sends a text message and, for group messages only,
// emits native Zalo mentions. Mention metadata is ignored for DMs because Zalo
// accepts native mentions through a group-only endpoint.
func SendMessageWithMentions(ctx context.Context, sess *Session, threadID string, threadType ThreadType, text string, mentions []*TMention) (string, error) {
	return SendMessageWithFormatting(ctx, sess, threadID, threadType, text, mentions, nil)
}

// TextStyle applies a native Zalo text style. Start and Len are UTF-16 code
// units, matching Zalo Web's JavaScript string offsets.
type TextStyle struct {
	Start int    `json:"start"`
	Len   int    `json:"len"`
	Style string `json:"st"`
}

// SendMessageWithFormatting sends text with optional native mentions and rich
// text styles. Zalo receives styles in the textProperties JSON string.
func SendMessageWithFormatting(ctx context.Context, sess *Session, threadID string, threadType ThreadType, text string, mentions []*TMention, styles []TextStyle) (string, error) {
	if text == "" {
		return "", fmt.Errorf("zalo_personal: message text cannot be empty")
	}

	serviceKey := "chat"
	apiPath := "/api/message/sms"
	if threadType == ThreadTypeGroup {
		serviceKey = "group"
		apiPath = "/api/group/sendmsg"
	}

	baseURL := getServiceURL(sess, serviceKey)
	if baseURL == "" {
		return "", fmt.Errorf("zalo_personal: no service URL for %s", serviceKey)
	}

	// Build payload
	payload := map[string]any{
		"message":  text,
		"clientId": time.Now().UnixMilli(),
		"ttl":      0,
	}
	if threadType == ThreadTypeGroup {
		payload["grid"] = threadID
		payload["visibility"] = 0
		if len(mentions) > 0 {
			mentionInfo, err := json.Marshal(mentions)
			if err != nil {
				return "", fmt.Errorf("zalo_personal: encode mention info: %w", err)
			}
			payload["mentionInfo"] = string(mentionInfo)
			apiPath = "/api/group/mention"
		}
	} else {
		payload["toid"] = threadID
		payload["imei"] = sess.IMEI
	}
	if len(styles) > 0 {
		textProperties, err := json.Marshal(map[string]any{"styles": styles, "ver": 0})
		if err != nil {
			return "", fmt.Errorf("zalo_personal: encode text properties: %w", err)
		}
		payload["textProperties"] = string(textProperties)
	}

	// Encrypt payload with session secret key
	encData, err := encryptPayload(sess, payload)
	if err != nil {
		return "", fmt.Errorf("zalo_personal: encrypt send payload: %w", err)
	}

	// Build URL with standard params
	sendURL := makeURL(sess, baseURL+apiPath, map[string]any{"nretry": 0}, true)

	// POST form-encoded
	form := buildFormBody(map[string]string{"params": encData})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, form)
	if err != nil {
		return "", err
	}
	setDefaultHeaders(req, sess)

	resp, err := sess.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("zalo_personal: send message: %w", err)
	}
	defer resp.Body.Close()

	// Send response is encrypted: {"error_code":0, "data":"<encrypted>"}
	var envelope Response[*string]
	if err := readJSON(resp, &envelope); err != nil {
		return "", fmt.Errorf("zalo_personal: parse send response: %w", err)
	}
	if envelope.ErrorCode != 0 {
		return "", fmt.Errorf("zalo_personal: send error code %d", envelope.ErrorCode)
	}
	if envelope.Data == nil {
		return "", nil
	}

	plain, err := decryptDataField(sess, *envelope.Data)
	if err != nil {
		return "", fmt.Errorf("zalo_personal: decrypt send response: %w", err)
	}

	var result struct {
		MsgID json.Number `json:"msgId"`
	}
	if err := json.Unmarshal(plain, &result); err != nil {
		return "", fmt.Errorf("zalo_personal: parse send result: %w", err)
	}

	return result.MsgID.String(), nil
}

// SendTypingEvent sends a typing indicator to a user or group.
// Zalo typing lasts ~5s server-side; callers should re-send every 3-4s.
func SendTypingEvent(ctx context.Context, sess *Session, threadID string, threadType ThreadType) error {
	serviceKey := "chat"
	apiPath := "/api/message/typing"
	if threadType == ThreadTypeGroup {
		serviceKey = "group"
		apiPath = "/api/group/typing"
	}

	baseURL := getServiceURL(sess, serviceKey)
	if baseURL == "" {
		return fmt.Errorf("zalo_personal: no service URL for %s", serviceKey)
	}

	payload := map[string]any{"imei": sess.IMEI}
	if threadType == ThreadTypeGroup {
		payload["grid"] = threadID
	} else {
		payload["toid"] = threadID
		payload["destType"] = 3 // DestTypeUser
	}

	encData, err := encryptPayload(sess, payload)
	if err != nil {
		return fmt.Errorf("zalo_personal: encrypt typing payload: %w", err)
	}

	typingURL := makeURL(sess, baseURL+apiPath, nil, true)
	form := buildFormBody(map[string]string{"params": encData})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, typingURL, form)
	if err != nil {
		return err
	}
	setDefaultHeaders(req, sess)

	resp, err := sess.Client.Do(req)
	if err != nil {
		return fmt.Errorf("zalo_personal: send typing: %w", err)
	}
	defer resp.Body.Close()

	var envelope Response[any]
	if err := readJSON(resp, &envelope); err != nil {
		return fmt.Errorf("zalo_personal: parse typing response: %w", err)
	}
	if envelope.ErrorCode != 0 {
		return fmt.Errorf("zalo_personal: typing error code %d", envelope.ErrorCode)
	}
	return nil
}

// getServiceURL extracts a service base URL from LoginInfo.
func getServiceURL(sess *Session, service string) string {
	if sess.LoginInfo == nil {
		return ""
	}
	var urls []string
	switch service {
	case "chat":
		urls = sess.LoginInfo.ZpwServiceMapV3.Chat
	case "group":
		urls = sess.LoginInfo.ZpwServiceMapV3.Group
	case "file":
		urls = sess.LoginInfo.ZpwServiceMapV3.File
	case "profile":
		urls = sess.LoginInfo.ZpwServiceMapV3.Profile
	case "reaction":
		urls = sess.LoginInfo.ZpwServiceMapV3.Reaction
	case "group_poll":
		urls = sess.LoginInfo.ZpwServiceMapV3.GroupPoll
	}
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

// encryptPayload encrypts a JSON payload with the session's secret key via AES-CBC.
func encryptPayload(sess *Session, payload map[string]any) (string, error) {
	blob, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	key, err := base64.StdEncoding.DecodeString(sess.SecretKey)
	if err != nil {
		return "", fmt.Errorf("decode secret key: %w", err)
	}
	return EncodeAESCBC(key, string(blob), false)
}
