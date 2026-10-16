package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// FriendInfo is a minimal friend/contact record for the picker UI.
type FriendInfo struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	ZaloName    string `json:"zaloName,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
}

// GroupListInfo is a minimal group record for the picker UI.
type GroupListInfo struct {
	GroupID     string `json:"groupId"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar,omitempty"`
	TotalMember int    `json:"totalMember"`
}

// GroupMemberInfo is a verified member record returned as part of Zalo group
// information. It contains only what is needed for safe outbound mentions.
type GroupMemberInfo struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
}

// FetchFriends fetches the authenticated user's friend list from Zalo.
func FetchFriends(ctx context.Context, sess *Session) ([]FriendInfo, error) {
	baseURL := getServiceURL(sess, "profile")
	if baseURL == "" {
		return nil, fmt.Errorf("zalo_personal: no profile service URL")
	}

	payload := map[string]any{
		"page":        1,
		"count":       20000,
		"incInvalid":  1,
		"avatar_size": 120,
		"actiontime":  0,
		"imei":        sess.IMEI,
	}

	encData, err := encryptPayload(sess, payload)
	if err != nil {
		return nil, fmt.Errorf("zalo_personal: encrypt friends payload: %w", err)
	}

	reqURL := makeURL(sess, baseURL+"/api/social/friend/getfriends",
		map[string]any{"params": encData}, true)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	setDefaultHeaders(req, sess)

	resp, err := sess.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zalo_personal: fetch friends: %w", err)
	}
	defer resp.Body.Close()

	// Response envelope: {"error_code":0, "data":"<encrypted_base64>"}
	var envelope Response[*string]
	if err := readJSON(resp, &envelope); err != nil {
		return nil, fmt.Errorf("zalo_personal: parse friends response: %w", err)
	}
	if envelope.ErrorCode != 0 {
		return nil, fmt.Errorf("zalo_personal: friends error code %d: %s", envelope.ErrorCode, envelope.ErrorMessage)
	}
	if envelope.Data == nil {
		return nil, fmt.Errorf("zalo_personal: empty friends data")
	}

	plain, err := decryptDataField(sess, *envelope.Data)
	if err != nil {
		return nil, fmt.Errorf("zalo_personal: decrypt friends: %w", err)
	}

	var friends []FriendInfo
	if err := json.Unmarshal(plain, &friends); err != nil {
		return nil, fmt.Errorf("zalo_personal: parse friends list: %w", err)
	}
	return friends, nil
}

// FetchGroups fetches the authenticated user's group list from Zalo (two-step).
func FetchGroups(ctx context.Context, sess *Session) ([]GroupListInfo, error) {
	// Step 1: Get group IDs from group_poll service
	gridVerMap, err := fetchGroupIDs(ctx, sess)
	if err != nil {
		return nil, err
	}
	if len(gridVerMap) == 0 {
		return nil, nil
	}

	// Step 2: Get group details in batches (Zalo rejects large payloads)
	const batchSize = 50
	ids := make([]string, 0, len(gridVerMap))
	for id := range gridVerMap {
		ids = append(ids, id)
	}

	var allGroups []GroupListInfo
	for i := 0; i < len(ids); i += batchSize {
		end := min(i+batchSize, len(ids))
		batch := make(map[string]string, end-i)
		for _, id := range ids[i:end] {
			batch[id] = gridVerMap[id]
		}
		groups, err := fetchGroupDetails(ctx, sess, batch)
		if err != nil {
			return nil, err
		}
		allGroups = append(allGroups, groups...)
	}

	sort.Slice(allGroups, func(i, j int) bool { return allGroups[i].Name < allGroups[j].Name })
	return allGroups, nil
}

// FetchGroupMembers retrieves the current member roster from Zalo's group-info
// response. Only UIDs returned by this response are eligible for native
// mentions; no UID or member API shape is guessed.
func FetchGroupMembers(ctx context.Context, sess *Session, groupID string) ([]GroupMemberInfo, error) {
	if groupID == "" {
		return nil, fmt.Errorf("zalo_personal: group ID cannot be empty")
	}
	baseURL := getServiceURL(sess, "group")
	if baseURL == "" {
		return nil, fmt.Errorf("zalo_personal: no group service URL")
	}

	gridVerJSON, err := json.Marshal(map[string]int{groupID: 0})
	if err != nil {
		return nil, err
	}
	encData, err := encryptPayload(sess, map[string]any{"gridVerMap": string(gridVerJSON)})
	if err != nil {
		return nil, fmt.Errorf("zalo_personal: encrypt group members payload: %w", err)
	}
	reqURL := makeURL(sess, baseURL+"/api/group/getmg-v2", nil, true)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, buildFormBody(map[string]string{"params": encData}))
	if err != nil {
		return nil, err
	}
	setDefaultHeaders(req, sess)

	resp, err := sess.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zalo_personal: fetch group members: %w", err)
	}
	defer resp.Body.Close()
	var envelope Response[*string]
	if err := readJSON(resp, &envelope); err != nil {
		return nil, fmt.Errorf("zalo_personal: parse group members response: %w", err)
	}
	if envelope.ErrorCode != 0 {
		return nil, fmt.Errorf("zalo_personal: group members error code %d: %s", envelope.ErrorCode, envelope.ErrorMessage)
	}
	if envelope.Data == nil {
		return nil, nil
	}
	plain, err := decryptDataField(sess, *envelope.Data)
	if err != nil {
		return nil, fmt.Errorf("zalo_personal: decrypt group members: %w", err)
	}
	var result struct {
		GridInfoMap map[string]struct {
			MemberIDs   []string `json:"memberIds"`
			MemVerList  []string `json:"memVerList"`
			CurrentMems []struct {
				ID       string `json:"id"`
				DName    string `json:"dName"`
				ZaloName string `json:"zaloName"`
			} `json:"currentMems"`
		} `json:"gridInfoMap"`
	}
	if err := json.Unmarshal(plain, &result); err != nil {
		return nil, fmt.Errorf("zalo_personal: parse group members: %w", err)
	}
	info, ok := result.GridInfoMap[groupID]
	if !ok {
		return nil, nil
	}
	memberByID := make(map[string]string, len(info.MemberIDs)+len(info.MemVerList)+len(info.CurrentMems))
	for _, member := range info.CurrentMems {
		if member.ID == "" || (member.DName == "" && member.ZaloName == "") {
			continue
		}
		name := member.DName
		if name == "" {
			name = member.ZaloName
		}
		memberByID[member.ID] = name
	}
	// Some groups return only memberIds or memVerList (without currentMems).
	// Resolve only those server-provided UIDs; names are never inferred from an
	// agent-supplied marker.
	memberIDs := make([]string, 0, len(info.MemberIDs)+len(info.MemVerList))
	seenIDs := make(map[string]struct{}, cap(memberIDs))
	for _, ids := range [][]string{info.MemberIDs, info.MemVerList} {
		for _, id := range ids {
			if id == "" {
				continue
			}
			if _, seen := seenIDs[id]; seen {
				continue
			}
			seenIDs[id] = struct{}{}
			memberIDs = append(memberIDs, id)
		}
	}
	missingIDs := make([]string, 0, len(memberIDs))
	for _, id := range memberIDs {
		if _, known := memberByID[id]; !known {
			missingIDs = append(missingIDs, id)
		}
	}
	if len(missingIDs) > 0 {
		profiles, err := fetchGroupMemberProfiles(ctx, sess, missingIDs)
		if err != nil {
			return nil, err
		}
		for id, name := range profiles {
			if name != "" {
				memberByID[id] = name
			}
		}
	}
	members := make([]GroupMemberInfo, 0, len(memberByID))
	for _, id := range memberIDs {
		if name := memberByID[id]; name != "" {
			members = append(members, GroupMemberInfo{UserID: id, DisplayName: name})
			delete(memberByID, id)
		}
	}
	// currentMems is still useful on responses that omit memberIds.
	for id, name := range memberByID {
		members = append(members, GroupMemberInfo{UserID: id, DisplayName: name})
	}
	return members, nil
}

// fetchGroupMemberProfiles resolves names only for UIDs already returned by a
// group's memberIds/memVerList fields. The endpoint and payload mirror zca-js
// GetGroupMembersInfo; this is not a guessed member API.
func fetchGroupMemberProfiles(ctx context.Context, sess *Session, memberIDs []string) (map[string]string, error) {
	baseURL := getServiceURL(sess, "profile")
	if baseURL == "" {
		return nil, fmt.Errorf("zalo_personal: no profile service URL")
	}

	profiles := make(map[string]string, len(memberIDs))
	const batchSize = 50
	for start := 0; start < len(memberIDs); start += batchSize {
		end := min(start+batchSize, len(memberIDs))
		versions := make([]string, 0, end-start)
		for _, id := range memberIDs[start:end] {
			version := id
			if !strings.Contains(version, "_") {
				version += "_0"
			}
			versions = append(versions, version)
		}
		encData, err := encryptPayload(sess, map[string]any{"friend_pversion_map": versions})
		if err != nil {
			return nil, fmt.Errorf("zalo_personal: encrypt group member profiles payload: %w", err)
		}
		reqURL := makeURL(sess, baseURL+"/api/social/group/members", map[string]any{"params": encData}, true)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}
		setDefaultHeaders(req, sess)
		resp, err := sess.Client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("zalo_personal: fetch group member profiles: %w", err)
		}
		var envelope Response[*string]
		readErr := readJSON(resp, &envelope)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("zalo_personal: parse group member profiles response: %w", readErr)
		}
		if envelope.ErrorCode != 0 {
			return nil, fmt.Errorf("zalo_personal: group member profiles error code %d: %s", envelope.ErrorCode, envelope.ErrorMessage)
		}
		if envelope.Data == nil {
			continue
		}
		plain, err := decryptDataField(sess, *envelope.Data)
		if err != nil {
			return nil, fmt.Errorf("zalo_personal: decrypt group member profiles: %w", err)
		}
		var result struct {
			Profiles map[string]struct {
				UserID      string `json:"id"`
				DisplayName string `json:"displayName"`
				ZaloName    string `json:"zaloName"`
			} `json:"profiles"`
		}
		if err := json.Unmarshal(plain, &result); err != nil {
			return nil, fmt.Errorf("zalo_personal: parse group member profiles: %w", err)
		}
		for rawID, profile := range result.Profiles {
			id := profile.UserID
			if id == "" {
				id = strings.TrimSuffix(rawID, "_0")
			}
			name := profile.DisplayName
			if name == "" {
				name = profile.ZaloName
			}
			if id != "" && name != "" {
				profiles[id] = name
			}
		}
	}
	return profiles, nil
}

// fetchGroupIDs gets group ID -> version map from group_poll service.
func fetchGroupIDs(ctx context.Context, sess *Session) (map[string]string, error) {
	baseURL := getServiceURL(sess, "group_poll")
	if baseURL == "" {
		return nil, fmt.Errorf("zalo_personal: no group_poll service URL")
	}

	reqURL := makeURL(sess, baseURL+"/api/group/getlg/v4", nil, true)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	setDefaultHeaders(req, sess)

	resp, err := sess.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zalo_personal: fetch group IDs: %w", err)
	}
	defer resp.Body.Close()

	var envelope Response[*string]
	if err := readJSON(resp, &envelope); err != nil {
		return nil, fmt.Errorf("zalo_personal: parse group IDs response: %w", err)
	}
	if envelope.ErrorCode != 0 {
		return nil, fmt.Errorf("zalo_personal: group IDs error code %d: %s", envelope.ErrorCode, envelope.ErrorMessage)
	}
	if envelope.Data == nil {
		return nil, nil
	}

	plain, err := decryptDataField(sess, *envelope.Data)
	if err != nil {
		return nil, fmt.Errorf("zalo_personal: decrypt group IDs: %w", err)
	}
	var result struct {
		GridVerMap map[string]string `json:"gridVerMap"`
	}
	if err := json.Unmarshal(plain, &result); err != nil {
		return nil, fmt.Errorf("zalo_personal: parse group IDs: %w", err)
	}
	return result.GridVerMap, nil
}

// fetchGroupDetails gets group info for given group IDs.
func fetchGroupDetails(ctx context.Context, sess *Session, gridVerMap map[string]string) ([]GroupListInfo, error) {
	baseURL := getServiceURL(sess, "group")
	if baseURL == "" {
		return nil, fmt.Errorf("zalo_personal: no group service URL")
	}

	// Build payload with version 0 to force full data retrieval.
	// Passing the actual version causes the server to return the group
	// in "unchangedsGroup" with no info in "gridInfoMap".
	zeroVerMap := make(map[string]int, len(gridVerMap))
	for id := range gridVerMap {
		zeroVerMap[id] = 0
	}
	gridVerJSON, err := json.Marshal(zeroVerMap)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"gridVerMap": string(gridVerJSON),
	}

	encData, err := encryptPayload(sess, payload)
	if err != nil {
		return nil, fmt.Errorf("zalo_personal: encrypt group details payload: %w", err)
	}

	reqURL := makeURL(sess, baseURL+"/api/group/getmg-v2", nil, true)

	form := buildFormBody(map[string]string{"params": encData})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, form)
	if err != nil {
		return nil, err
	}
	setDefaultHeaders(req, sess)

	resp, err := sess.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zalo_personal: fetch group details: %w", err)
	}
	defer resp.Body.Close()

	var envelope Response[*string]
	if err := readJSON(resp, &envelope); err != nil {
		return nil, fmt.Errorf("zalo_personal: parse group details response: %w", err)
	}
	if envelope.ErrorCode != 0 {
		return nil, fmt.Errorf("zalo_personal: group details error code %d: %s", envelope.ErrorCode, envelope.ErrorMessage)
	}
	if envelope.Data == nil {
		return nil, nil
	}

	plain, err := decryptDataField(sess, *envelope.Data)
	if err != nil {
		return nil, fmt.Errorf("zalo_personal: decrypt group details: %w", err)
	}
	var result struct {
		GridInfoMap map[string]struct {
			Name        string `json:"name"`
			Avatar      string `json:"avt"`
			TotalMember int    `json:"totalMember"`
		} `json:"gridInfoMap"`
	}
	if err := json.Unmarshal(plain, &result); err != nil {
		return nil, fmt.Errorf("zalo_personal: parse group details: %w", err)
	}
	groups := make([]GroupListInfo, 0, len(result.GridInfoMap))
	for id, info := range result.GridInfoMap {
		groups = append(groups, GroupListInfo{
			GroupID:     id,
			Name:        info.Name,
			Avatar:      info.Avatar,
			TotalMember: info.TotalMember,
		})
	}
	return groups, nil
}

// decryptDataField decrypts an encrypted base64 data string from Zalo API response.
// The decrypted payload is itself a Response envelope: {"error_code":0, "data":...},
// so this function unwraps the inner envelope and returns the raw data field.
func decryptDataField(sess *Session, data string) ([]byte, error) {
	key := SecretKey(sess.SecretKey).Bytes()
	if key == nil {
		return nil, fmt.Errorf("zalo_personal: invalid session secret key")
	}
	unescaped, err := url.PathUnescape(data)
	if err != nil {
		return nil, err
	}
	plain, err := DecodeAESCBC(key, unescaped)
	if err != nil {
		return nil, err
	}

	// Unwrap inner response envelope
	var inner Response[json.RawMessage]
	if err := json.Unmarshal(plain, &inner); err != nil {
		return nil, fmt.Errorf("zalo_personal: unwrap inner response: %w", err)
	}
	if inner.ErrorCode != 0 {
		return nil, fmt.Errorf("zalo_personal: inner error code %d: %s", inner.ErrorCode, inner.ErrorMessage)
	}
	return inner.Data, nil
}
