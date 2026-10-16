package personal

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/nextlevelbuilder/goclaw/internal/channels"
	"github.com/nextlevelbuilder/goclaw/internal/channels/zalo/personal/protocol"
)

// mentionResolution is the rendered Zalo text together with native metadata.
// All offsets in Mentions are UTF-16 code units as required by Zalo clients.
type mentionResolution struct {
	text     string
	mentions []*protocol.TMention
	styles   []protocol.TextStyle
}

// resolveNativeMentions accepts only the internal, explicit UID syntax:
// @[UID|display name]. The UID must be in the verified group roster and its
// resolved display name must be unique in that roster. Invalid or ambiguous
// markers become ordinary visible text and never cause a notification.
func resolveNativeMentions(text string, members []protocol.GroupMemberInfo) mentionResolution {
	byID := make(map[string]string, len(members))
	nameCount := make(map[string]int, len(members))
	for _, member := range members {
		uid, name := strings.TrimSpace(member.UserID), strings.TrimSpace(member.DisplayName)
		if uid == "" || name == "" || uid == protocol.MentionAllUID {
			continue
		}
		byID[uid] = name
		nameCount[name]++
	}

	var out strings.Builder
	mentions := make([]*protocol.TMention, 0)
	for cursor := 0; cursor < len(text); {
		start := strings.Index(text[cursor:], "@[")
		if start < 0 {
			out.WriteString(text[cursor:])
			break
		}
		start += cursor
		out.WriteString(text[cursor:start])
		endRel := strings.IndexByte(text[start+2:], ']')
		if endRel < 0 {
			out.WriteString(text[start:])
			break
		}
		end := start + 2 + endRel
		marker := text[start+2 : end]
		parts := strings.SplitN(marker, "|", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			out.WriteString(text[start : end+1])
			cursor = end + 1
			continue
		}

		uid := strings.TrimSpace(parts[0])
		fallbackName := strings.TrimSpace(parts[1])
		name, known := byID[uid]
		if !known || nameCount[name] != 1 {
			out.WriteByte('@')
			out.WriteString(fallbackName)
			cursor = end + 1
			continue
		}

		position := utf16Length(out.String())
		out.WriteByte('@')
		out.WriteString(name)
		mentions = append(mentions, &protocol.TMention{
			UID:  uid,
			Pos:  position,
			Len:  utf16Length("@" + name),
			Type: protocol.MentionEach,
		})
		cursor = end + 1
	}
	return mentionResolution{text: out.String(), mentions: mentions}
}

func utf16Length(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func hasInternalMention(text string) bool { return strings.Contains(text, "@[") }

func (c *Channel) resolveGroupMentions(ctx context.Context, groupID, text string) mentionResolution {
	if !hasInternalMention(text) {
		return mentionResolution{text: text}
	}
	members, err := c.groupMembersForSend(ctx, groupID)
	if err != nil {
		slog.Warn("zalo_personal: unable to verify group members; sending without native mentions", "group_id", groupID, "error", err)
	}
	return resolveNativeMentions(text, members)
}

func (c *Channel) groupMembersForSend(ctx context.Context, groupID string) ([]protocol.GroupMemberInfo, error) {
	c.membersMu.RLock()
	entry, cached := c.groupMembers[groupID]
	c.membersMu.RUnlock()
	if cached && entry.verified && len(entry.members) > 0 && !entry.fetchedAt.IsZero() && entry.fetchedAt.Add(groupMembersCacheTTL).After(now()) {
		return append([]protocol.GroupMemberInfo(nil), entry.members...), nil
	}

	sess := c.session()
	if sess == nil {
		return nil, errNoZaloSession
	}
	members, err := protocol.FetchGroupMembers(ctx, sess, groupID)
	if err != nil {
		return nil, err
	}
	c.cacheGroupMembers(groupID, members)
	return members, nil
}

func (c *Channel) cacheGroupMembers(groupID string, members []protocol.GroupMemberInfo) {
	if groupID == "" || len(members) == 0 {
		return
	}
	copyMembers := append([]protocol.GroupMemberInfo(nil), members...)
	c.membersMu.Lock()
	c.groupMembers[groupID] = groupMembersCacheEntry{members: copyMembers, fetchedAt: now(), verified: true}
	c.membersMu.Unlock()
}

func (c *Channel) rememberGroupMember(groupID, uid, displayName string) {
	if groupID == "" || uid == "" || displayName == "" || uid == protocol.MentionAllUID {
		return
	}
	c.membersMu.Lock()
	entry := c.groupMembers[groupID]
	for i := range entry.members {
		if entry.members[i].UserID == uid {
			entry.members[i].DisplayName = displayName
			entry.fetchedAt = now()
			c.groupMembers[groupID] = entry
			c.membersMu.Unlock()
			return
		}
	}
	entry.members = append(entry.members, protocol.GroupMemberInfo{UserID: uid, DisplayName: displayName})
	entry.fetchedAt = now()
	c.groupMembers[groupID] = entry
	c.membersMu.Unlock()
}

// splitMentionResolution keeps a native mention wholly inside one chunk. The
// transport limit is bytes, matching the existing Zalo Personal chunker.
func splitMentionResolution(in mentionResolution, maxBytes int) []mentionResolution {
	if in.text == "" || maxBytes <= 0 {
		return nil
	}
	if len(in.text) <= maxBytes {
		return []mentionResolution{in}
	}

	byteMentions := make([]struct {
		start, end int
		mention    *protocol.TMention
	}, 0, len(in.mentions))
	for _, mention := range in.mentions {
		start, end := byteRangeForUTF16(in.text, mention.Pos, mention.Len)
		if start >= 0 && end > start {
			byteMentions = append(byteMentions, struct {
				start, end int
				mention    *protocol.TMention
			}{start, end, mention})
		}
	}

	parts := make([]mentionResolution, 0, (len(in.text)/maxBytes)+1)
	for offset := 0; offset < len(in.text); {
		limit := offset + maxBytes
		if limit > len(in.text) {
			limit = len(in.text)
		}
		limit = naturalChunkLimit(in.text, offset, limit)
		for _, span := range byteMentions {
			if span.start < limit && span.end > limit {
				limit = span.start
			}
		}
		if limit <= offset {
			// A pathological display name can exceed the transport limit. Keep it
			// intact in one chunk: correctness of the native mention is safer than
			// silently splitting a person name into two messages.
			for _, span := range byteMentions {
				if span.start == offset && span.end > offset {
					limit = span.end
					break
				}
			}
			if limit <= offset {
				limit = offset + maxBytes
				for limit < len(in.text) && !utf8.RuneStart(in.text[limit]) {
					limit--
				}
			}
		}

		chunk := mentionResolution{text: in.text[offset:limit]}
		for _, span := range byteMentions {
			if span.start >= offset && span.end <= limit {
				chunk.mentions = append(chunk.mentions, &protocol.TMention{
					UID: span.mention.UID, Pos: utf16Length(in.text[offset:span.start]),
					Len: utf16Length(in.text[span.start:span.end]), Type: protocol.MentionEach,
				})
			}
		}
		for _, style := range in.styles {
			start, end := byteRangeForUTF16(in.text, style.Start, style.Len)
			if start >= offset && end <= limit {
				chunk.styles = append(chunk.styles, protocol.TextStyle{
					Start: utf16Length(in.text[offset:start]),
					Len:   utf16Length(in.text[start:end]),
					Style: style.Style,
				})
			}
		}
		parts = append(parts, chunk)
		offset = limit
	}
	return parts
}

// naturalChunkLimit keeps outbound bubbles readable. It prefers a line or
// paragraph boundary, then a word boundary, and only hard-cuts when one line
// itself exceeds the transport limit.
func naturalChunkLimit(text string, offset, hardLimit int) int {
	limit := hardLimit
	for limit < len(text) && !utf8.RuneStart(text[limit]) {
		limit--
	}
	if newline := strings.LastIndexByte(text[offset:limit], '\n'); newline >= 0 {
		candidate := offset + newline + 1
		if candidate > offset {
			return candidate
		}
	}
	for i := limit - 1; i > offset; i-- {
		if text[i] == ' ' || text[i] == '\t' {
			return i + 1
		}
	}
	return limit
}

// addZaloReportStyles adds native rich-text formatting to visible report
// headings. Main numbered sections are green and bold; emoji-led subheadings
// are orange and bold. It leaves ordinary chat untouched.
func addZaloReportStyles(in mentionResolution) mentionResolution {
	lineStart := 0
	for lineStart < len(in.text) {
		lineEnd := strings.IndexByte(in.text[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(in.text)
		} else {
			lineEnd += lineStart
		}
		line := in.text[lineStart:lineEnd]
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			leadingBytes := len(line) - len(strings.TrimLeft(line, " \t"))
			start := utf16Length(in.text[:lineStart+leadingBytes])
			length := utf16Length(trimmed)
			switch {
			case isNumberedReportHeading(trimmed):
				in.styles = append(in.styles,
					protocol.TextStyle{Start: start, Len: length, Style: "c_15a85f"},
					protocol.TextStyle{Start: start, Len: length, Style: "b"},
				)
			case isEmojiReportHeading(trimmed):
				in.styles = append(in.styles,
					protocol.TextStyle{Start: start, Len: length, Style: "c_f27806"},
					protocol.TextStyle{Start: start, Len: length, Style: "b"},
				)
			}
		}
		if lineEnd == len(in.text) {
			break
		}
		lineStart = lineEnd + 1
	}
	return in
}

func isNumberedReportHeading(line string) bool {
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	return i > 0 && i+1 < len(line) && (line[i] == '.' || line[i] == ')') && line[i+1] == ' '
}

func isEmojiReportHeading(line string) bool {
	if line == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(line)
	switch r {
	case '📦', '🧾', '🌿', '⛺', '⚠', '✅', '🔍', '📊', '💡':
		return true
	default:
		return false
	}
}

func byteRangeForUTF16(text string, position, length int) (int, int) {
	if position < 0 || length <= 0 {
		return -1, -1
	}
	units, start, end := 0, -1, -1
	for index, r := range text {
		if units == position {
			start = index
		}
		units += len(utf16.Encode([]rune{r}))
		if units == position+length {
			end = index + utf8.RuneLen(r)
			break
		}
	}
	return start, end
}

// now is replaceable by tests without exposing mutable time on Channel.
var now = func() time.Time { return time.Now() }

var errNoZaloSession = errors.New("zalo_personal: not connected")

var _ channels.GroupMemberProvider = (*Channel)(nil)
