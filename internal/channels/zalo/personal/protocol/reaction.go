package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AddHeartReaction adds Zalo's built-in HEART reaction to an inbound message.
// Both message IDs come from the authenticated inbound event and are required
// by Zalo's reaction endpoint.
func AddHeartReaction(ctx context.Context, sess *Session, threadID string, threadType ThreadType, msgID, cliMsgID string) error {
	if msgID == "" || cliMsgID == "" {
		return fmt.Errorf("zalo_personal: reaction requires message and client message IDs")
	}
	baseURL := getServiceURL(sess, "reaction")
	if baseURL == "" {
		return fmt.Errorf("zalo_personal: no service URL for reaction")
	}
	path := "/api/message/reaction"
	if threadType == ThreadTypeGroup {
		path = "/api/group/reaction"
	}
	reactionMessage, err := json.Marshal(map[string]any{
		"rMsg":  []map[string]any{{"gMsgID": json.Number(msgID), "cMsgID": json.Number(cliMsgID), "msgType": 1}},
		"rIcon": "HEART", "rType": 5, "source": 6,
	})
	if err != nil {
		return fmt.Errorf("zalo_personal: encode heart reaction: %w", err)
	}
	payload := map[string]any{
		"react_list": []map[string]any{{"message": string(reactionMessage), "clientId": time.Now().UnixMilli()}},
	}
	if threadType == ThreadTypeGroup {
		payload["grid"] = threadID
		payload["imei"] = sess.IMEI
	} else {
		payload["toid"] = threadID
	}
	encData, err := encryptPayload(sess, payload)
	if err != nil {
		return fmt.Errorf("zalo_personal: encrypt heart reaction: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, makeURL(sess, baseURL+path, nil, true), buildFormBody(map[string]string{"params": encData}))
	if err != nil {
		return err
	}
	setDefaultHeaders(req, sess)
	resp, err := sess.Client.Do(req)
	if err != nil {
		return fmt.Errorf("zalo_personal: add heart reaction: %w", err)
	}
	defer resp.Body.Close()
	var envelope Response[any]
	if err := readJSON(resp, &envelope); err != nil {
		return fmt.Errorf("zalo_personal: parse heart reaction response: %w", err)
	}
	if envelope.ErrorCode != 0 {
		return fmt.Errorf("zalo_personal: heart reaction error code %d", envelope.ErrorCode)
	}
	return nil
}
