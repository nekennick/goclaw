package personal

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/nextlevelbuilder/goclaw/internal/channels"
	"github.com/nextlevelbuilder/goclaw/internal/channels/zalo/personal/protocol"
)

type BlockKind int

const (
	BlockParagraph BlockKind = iota
	BlockHeadingMain
	BlockHeadingSub
	BlockBullet
	BlockQA
	BlockCodeBlock
)

type ContentBlock struct {
	Kind      BlockKind
	StartByte int
	EndByte   int
}

type SubSection struct {
	HeadingBlock      *ContentBlock
	FirstContentBlock *ContentBlock
	Blocks            []ContentBlock
	StartByte         int
	EndByte           int
}

type Section struct {
	HeadingBlock      *ContentBlock // nil for leading content before first heading
	FirstContentBlock *ContentBlock
	SubSections       []SubSection
	Blocks            []ContentBlock
	StartByte         int
	EndByte           int
}

type DocumentStructure struct {
	Blocks   []ContentBlock
	Sections []Section
}

type byteMentionSpan struct {
	start, end int
	mention    *protocol.TMention
}

// mentionResolution is the rendered Zalo text together with native metadata.
// All offsets in Mentions are UTF-16 code units as required by Zalo clients.
type mentionResolution struct {
	text      string
	mentions  []*protocol.TMention
	styles    []protocol.TextStyle
	structure DocumentStructure
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

// splitMentionResolution splits a rendered Zalo message into chunks that respect
// transport limits (maxBytes), document structure, section boundaries, and native metadata.
func splitMentionResolution(in mentionResolution, maxBytes int) []mentionResolution {
	if in.text == "" || maxBytes <= 0 {
		return nil
	}
	if len(in.text) <= maxBytes {
		return []mentionResolution{in}
	}

	byteMentions := make([]byteMentionSpan, 0, len(in.mentions))
	for _, mention := range in.mentions {
		start, end := byteRangeForUTF16(in.text, mention.Pos, mention.Len)
		if start >= 0 && end > start {
			byteMentions = append(byteMentions, byteMentionSpan{start, end, mention})
		}
	}

	structure := in.structure
	if len(structure.Blocks) == 0 {
		structure = extractDocumentStructure(in.text)
	}

	parts := make([]mentionResolution, 0, (len(in.text)/maxBytes)+1)
	for offset := 0; offset < len(in.text); {
		limit := chooseNextChunkLimit(in.text, offset, maxBytes, structure, byteMentions)
		if limit <= offset {
			// Absolute guarantee against infinite loop: advance by at least 1 rune
			_, size := utf8.DecodeRuneInString(in.text[offset:])
			if size > 0 {
				limit = offset + size
			} else {
				limit = offset + 1
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
			if start < 0 || end <= start {
				continue
			}
			intStart := max(start, offset)
			intEnd := min(end, limit)
			if intStart < intEnd {
				chunk.styles = append(chunk.styles, protocol.TextStyle{
					Start: utf16Length(in.text[offset:intStart]),
					Len:   utf16Length(in.text[intStart:intEnd]),
					Style: style.Style,
				})
			}
		}
		parts = append(parts, chunk)
		offset = limit
	}
	return parts
}

func chooseNextChunkLimit(text string, offset, maxBytes int, doc DocumentStructure, byteMentions []byteMentionSpan) int {
	totalLen := len(text)
	if offset >= totalLen {
		return totalLen
	}
	hardLimit := offset + maxBytes
	if hardLimit >= totalLen {
		return totalLen
	}

	// Avoid leaving an ultra-short final fragment (e.g. < 80 bytes) in its own bubble
	// when possible, by pulling hardLimit back slightly.
	minFinalChunkBytes := 80
	if maxBytes/4 < minFinalChunkBytes {
		minFinalChunkBytes = maxBytes / 4
	}
	if remaining := totalLen - hardLimit; remaining > 0 && remaining < minFinalChunkBytes {
		hardLimit -= minFinalChunkBytes - remaining
	}
	if hardLimit <= offset {
		hardLimit = min(offset+maxBytes, totalLen)
	}

	// 1. Try Section-level boundaries (Requirements 2 & 3)
	currSecIdx := -1
	for i, sec := range doc.Sections {
		if offset >= sec.StartByte && offset < sec.EndByte {
			currSecIdx = i
			break
		}
	}

	if currSecIdx >= 0 {
		currSec := doc.Sections[currSecIdx]

		if offset == currSec.StartByte {
			// Case 1A: Section fits completely in hardLimit
			if currSec.EndByte <= hardLimit {
				bestEnd := currSec.EndByte
				for nextIdx := currSecIdx + 1; nextIdx < len(doc.Sections); nextIdx++ {
					nextSec := doc.Sections[nextIdx]
					if nextSec.EndByte <= hardLimit {
						bestEnd = nextSec.EndByte
					} else {
						// nextSec does not fit in remaining space.
						// Move entire nextSec to next chunk!
						break
					}
				}
				return ensureSafeLimit(text, offset, bestEnd, hardLimit, doc, byteMentions)
			}
		} else {
			// Case 1B: We are in the middle of currSec
			if currSec.EndByte <= hardLimit {
				bestEnd := currSec.EndByte
				for nextIdx := currSecIdx + 1; nextIdx < len(doc.Sections); nextIdx++ {
					nextSec := doc.Sections[nextIdx]
					if nextSec.EndByte <= hardLimit {
						bestEnd = nextSec.EndByte
					} else {
						break
					}
				}
				return ensureSafeLimit(text, offset, bestEnd, hardLimit, doc, byteMentions)
			}
		}

		// 2. Splitting inside currSec: Try SubSection boundaries (Requirement 4)
		if len(currSec.SubSections) > 0 {
			bestSubLimit := -1
			for _, sub := range currSec.SubSections {
				if sub.StartByte > offset && sub.StartByte <= hardLimit {
					// Requirement 3: Ensure not cutting right after main heading
					if currSec.HeadingBlock != nil && sub.StartByte == currSec.HeadingBlock.EndByte {
						continue
					}
					bestSubLimit = sub.StartByte
				}
			}
			if bestSubLimit > offset {
				return ensureSafeLimit(text, offset, bestSubLimit, hardLimit, doc, byteMentions)
			}
		}

		// 3. Splitting inside currSec: Try Block boundaries (Paragraphs, Bullets, Q&A)
		bestBlockLimit := -1
		for _, b := range currSec.Blocks {
			// Block end boundary
			if b.EndByte > offset && b.EndByte <= hardLimit {
				// Requirement 3: Never cut immediately after a heading!
				if b.Kind == BlockHeadingMain || b.Kind == BlockHeadingSub {
					continue
				}
				bestBlockLimit = b.EndByte
			}
			// Cut point BEFORE a heading that starts in this chunk
			if (b.Kind == BlockHeadingMain || b.Kind == BlockHeadingSub) && b.StartByte > offset && b.StartByte <= hardLimit {
				if b.StartByte > bestBlockLimit {
					bestBlockLimit = b.StartByte
				}
			}
		}
		if bestBlockLimit > offset {
			return ensureSafeLimit(text, offset, bestBlockLimit, hardLimit, doc, byteMentions)
		}
	}

	// 4. Try global blocks
	bestBlockLimit := -1
	for _, b := range doc.Blocks {
		if b.EndByte > offset && b.EndByte <= hardLimit {
			if b.Kind == BlockHeadingMain || b.Kind == BlockHeadingSub {
				continue
			}
			bestBlockLimit = b.EndByte
		}
		if (b.Kind == BlockHeadingMain || b.Kind == BlockHeadingSub) && b.StartByte > offset && b.StartByte <= hardLimit {
			if b.StartByte > bestBlockLimit {
				bestBlockLimit = b.StartByte
			}
		}
	}
	if bestBlockLimit > offset {
		return ensureSafeLimit(text, offset, bestBlockLimit, hardLimit, doc, byteMentions)
	}

	// 5. Fallback within a single block (Requirement 5): Sentence -> Word -> Rune
	return fallbackChunkLimit(text, offset, hardLimit, doc, byteMentions)
}

func rawFallbackChunkLimit(text string, offset, hardLimit int, byteMentions []byteMentionSpan) int {
	if offset >= hardLimit {
		return hardLimit
	}
	sub := text[offset:hardLimit]

	// A. Sentence boundary: (. ! ? ;) followed by whitespace or newline
	lastSentenceEnd := -1
	for i := len(sub) - 1; i > 0; i-- {
		ch := sub[i-1]
		if ch == '.' || ch == '!' || ch == '?' || ch == ';' {
			if sub[i] == ' ' || sub[i] == '\t' || sub[i] == '\n' {
				cand := offset + i + 1
				if cand > offset && isMentionSafe(cand, byteMentions) {
					lastSentenceEnd = cand
					break
				}
			}
		}
	}
	if lastSentenceEnd > offset {
		return lastSentenceEnd
	}

	// B. Word boundary: space, tab, newline
	for i := len(sub) - 1; i > 0; i-- {
		if sub[i] == ' ' || sub[i] == '\t' || sub[i] == '\n' {
			cand := offset + i + 1
			if cand > offset && isMentionSafe(cand, byteMentions) {
				return cand
			}
		}
	}

	// C. Rune boundary: valid UTF-8
	limit := hardLimit
	for limit < len(text) && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return limit
}

func fallbackChunkLimit(text string, offset, hardLimit int, doc DocumentStructure, byteMentions []byteMentionSpan) int {
	cand := rawFallbackChunkLimit(text, offset, hardLimit, byteMentions)
	return ensureSafeLimit(text, offset, cand, hardLimit, doc, byteMentions)
}

func ensureSafeLimit(text string, offset, limit, hardLimit int, doc DocumentStructure, byteMentions []byteMentionSpan) int {
	if limit > len(text) {
		limit = len(text)
	}
	for limit < len(text) && !utf8.RuneStart(text[limit]) {
		limit--
	}

	// Requirement 3: Never leave a heading alone at the end of a chunk
	for _, b := range doc.Blocks {
		if b.Kind != BlockHeadingMain && b.Kind != BlockHeadingSub {
			continue
		}
		// Case A: Heading starts in this chunk after offset.
		// If limit falls inside or immediately after the heading, pull back before the heading!
		if b.StartByte > offset && limit >= b.StartByte && limit <= b.EndByte {
			limit = b.StartByte
			break
		}
		// Case B: Chunk starts at this heading. If limit is at or before heading end,
		// advance into the first content block (if hardLimit allows).
		if b.StartByte == offset && limit <= b.EndByte && hardLimit > b.EndByte {
			cand := rawFallbackChunkLimit(text, b.EndByte, hardLimit, byteMentions)
			if cand > b.EndByte {
				limit = cand
			} else {
				_, size := utf8.DecodeRuneInString(text[b.EndByte:])
				if size > 0 {
					limit = b.EndByte + size
				}
			}
			break
		}
	}

	// Check against native mentions: never cut inside a mention
	for _, span := range byteMentions {
		if span.start < limit && span.end > limit {
			limit = span.start
		}
	}

	// Guaranteed progress check (Requirement 10)
	if limit <= offset {
		for _, span := range byteMentions {
			if span.start == offset && span.end > offset {
				return span.end
			}
		}
		_, size := utf8.DecodeRuneInString(text[offset:])
		if size > 0 {
			limit = offset + size
		} else {
			limit = offset + 1
		}
	}
	return limit
}

func isMentionSafe(cutPoint int, byteMentions []byteMentionSpan) bool {
	for _, span := range byteMentions {
		if span.start < cutPoint && span.end > cutPoint {
			return false
		}
	}
	return true
}

func buildSections(blocks []ContentBlock) []Section {
	if len(blocks) == 0 {
		return nil
	}

	var sections []Section
	var currentSection *Section

	for i := range blocks {
		b := blocks[i]
		if b.Kind == BlockHeadingMain || currentSection == nil {
			if currentSection != nil {
				currentSection.EndByte = currentSection.Blocks[len(currentSection.Blocks)-1].EndByte
				sections = append(sections, *currentSection)
			}
			var hBlock *ContentBlock
			if b.Kind == BlockHeadingMain {
				hBlock = &blocks[i]
			}
			currentSection = &Section{
				HeadingBlock: hBlock,
				Blocks:       []ContentBlock{b},
				StartByte:    b.StartByte,
			}
			continue
		}
		currentSection.Blocks = append(currentSection.Blocks, b)
	}

	if currentSection != nil && len(currentSection.Blocks) > 0 {
		currentSection.EndByte = currentSection.Blocks[len(currentSection.Blocks)-1].EndByte
		sections = append(sections, *currentSection)
	}

	for sIdx := range sections {
		s := &sections[sIdx]
		for bIdx := range s.Blocks {
			if s.Blocks[bIdx].Kind != BlockHeadingMain {
				s.FirstContentBlock = &s.Blocks[bIdx]
				break
			}
		}

		var subSections []SubSection
		var currentSub *SubSection

		for bIdx := range s.Blocks {
			blk := s.Blocks[bIdx]
			if blk.Kind == BlockHeadingSub {
				if currentSub != nil {
					currentSub.EndByte = currentSub.Blocks[len(currentSub.Blocks)-1].EndByte
					subSections = append(subSections, *currentSub)
				}
				currentSub = &SubSection{
					HeadingBlock: &s.Blocks[bIdx],
					Blocks:       []ContentBlock{blk},
					StartByte:    blk.StartByte,
				}
				continue
			}
			if currentSub != nil {
				currentSub.Blocks = append(currentSub.Blocks, blk)
			}
		}
		if currentSub != nil && len(currentSub.Blocks) > 0 {
			currentSub.EndByte = currentSub.Blocks[len(currentSub.Blocks)-1].EndByte
			subSections = append(subSections, *currentSub)
		}

		for subIdx := range subSections {
			sub := &subSections[subIdx]
			for bIdx := range sub.Blocks {
				if sub.Blocks[bIdx].Kind != BlockHeadingSub {
					sub.FirstContentBlock = &sub.Blocks[bIdx]
					break
				}
			}
		}
		s.SubSections = subSections
	}

	return sections
}

func makeContiguousBlocks(blocks []ContentBlock, totalLen int) []ContentBlock {
	if totalLen == 0 {
		return nil
	}
	if len(blocks) == 0 {
		return []ContentBlock{{Kind: BlockParagraph, StartByte: 0, EndByte: totalLen}}
	}
	blocks[0].StartByte = 0
	for i := 0; i < len(blocks)-1; i++ {
		blocks[i].EndByte = blocks[i+1].StartByte
	}
	blocks[len(blocks)-1].EndByte = totalLen
	return blocks
}

var (
	reQAQuestionPrefix = regexp.MustCompile(`(?i)^(?:[-*•]\s*)?(?:câu\s*hỏi|hỏi|q)\s*[:.]`)
	reQAAnswerPrefix   = regexp.MustCompile(`(?i)^(?:[-*•]\s*)?(?:câu\s*trả\s*lời|trả\s*lời|đáp|a)\s*[:.]`)
)

func isQuestionLine(s string) bool {
	trimmed := strings.TrimSpace(s)
	if reQAQuestionPrefix.MatchString(trimmed) {
		return true
	}
	if strings.HasSuffix(trimmed, "?") && utf8.RuneCountInString(trimmed) <= 300 {
		return true
	}
	return false
}

func isAnswerLine(s string) bool {
	trimmed := strings.TrimSpace(s)
	return reQAAnswerPrefix.MatchString(trimmed)
}

func isIndentedLine(s string) bool {
	return strings.HasPrefix(s, "  ") || strings.HasPrefix(s, "\t")
}

func extractDocumentStructure(text string) DocumentStructure {
	if text == "" {
		return DocumentStructure{}
	}
	lines := strings.Split(text, "\n")
	var rawBlocks []ContentBlock
	offset := 0
	prevBlockIdx := -1

	for i, rawLine := range lines {
		lineStart := offset
		lineEnd := lineStart + len(rawLine)
		if i < len(lines)-1 {
			lineEnd++
		}
		offset = lineEnd

		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" {
			if prevBlockIdx >= 0 && prevBlockIdx < len(rawBlocks) {
				rawBlocks[prevBlockIdx].EndByte = lineEnd
			}
			continue
		}

		if reHeadingH1H2.MatchString(trimmed) || (reNumberedLine.MatchString(trimmed) && utf8.RuneCountInString(trimmed) <= 60 && !strings.Contains(trimmed, ":")) {
			rawBlocks = append(rawBlocks, ContentBlock{Kind: BlockHeadingMain, StartByte: lineStart, EndByte: lineEnd})
			prevBlockIdx = len(rawBlocks) - 1
			continue
		}

		if reHeadingH3Plus.MatchString(trimmed) || (isEmojiReportHeading(trimmed) && utf8.RuneCountInString(trimmed) <= 60) {
			rawBlocks = append(rawBlocks, ContentBlock{Kind: BlockHeadingSub, StartByte: lineStart, EndByte: lineEnd})
			prevBlockIdx = len(rawBlocks) - 1
			continue
		}

		if reBulletLine.MatchString(trimmed) || (strings.HasSuffix(trimmed, ":") && len(trimmed) <= 40) {
			rawBlocks = append(rawBlocks, ContentBlock{Kind: BlockBullet, StartByte: lineStart, EndByte: lineEnd})
			prevBlockIdx = len(rawBlocks) - 1
			continue
		}

		if isIndentedLine(rawLine) && prevBlockIdx >= 0 && rawBlocks[prevBlockIdx].Kind == BlockBullet {
			rawBlocks[prevBlockIdx].EndByte = lineEnd
			continue
		}

		if isAnswerLine(trimmed) && prevBlockIdx >= 0 && rawBlocks[prevBlockIdx].Kind == BlockQA {
			rawBlocks[prevBlockIdx].EndByte = lineEnd
			continue
		}

		if isQuestionLine(trimmed) {
			rawBlocks = append(rawBlocks, ContentBlock{Kind: BlockQA, StartByte: lineStart, EndByte: lineEnd})
			prevBlockIdx = len(rawBlocks) - 1
			continue
		}

		if isAnswerLine(trimmed) {
			rawBlocks = append(rawBlocks, ContentBlock{Kind: BlockQA, StartByte: lineStart, EndByte: lineEnd})
			prevBlockIdx = len(rawBlocks) - 1
			continue
		}

		if prevBlockIdx >= 0 && rawBlocks[prevBlockIdx].Kind == BlockParagraph && i > 0 && strings.TrimSpace(lines[i-1]) != "" {
			rawBlocks[prevBlockIdx].EndByte = lineEnd
			continue
		}

		rawBlocks = append(rawBlocks, ContentBlock{Kind: BlockParagraph, StartByte: lineStart, EndByte: lineEnd})
		prevBlockIdx = len(rawBlocks) - 1
	}

	blocks := makeContiguousBlocks(rawBlocks, len(text))
	sections := buildSections(blocks)
	return DocumentStructure{Blocks: blocks, Sections: sections}
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
				hLen := length
				if colon := strings.IndexByte(trimmed, ':'); colon > 0 && colon < 60 {
					hLen = utf16Length(trimmed[:colon])
				}
				in.styles = append(in.styles,
					protocol.TextStyle{Start: start, Len: hLen, Style: "c_15a85f"},
					protocol.TextStyle{Start: start, Len: hLen, Style: "b"},
				)
			case isEmojiReportHeading(trimmed):
				in.styles = append(in.styles,
					protocol.TextStyle{Start: start, Len: length, Style: "c_f27806"},
					protocol.TextStyle{Start: start, Len: length, Style: "b"},
				)
			default:
				if relOffset, byteLen := extractBulletLabelRange(trimmed); byteLen > 0 {
					labelStart := utf16Length(in.text[:lineStart+leadingBytes+relOffset])
					labelLen := utf16Length(trimmed[relOffset : relOffset+byteLen])
					if !hasStyleOverlap(in.styles, labelStart, labelLen) {
						in.styles = append(in.styles, protocol.TextStyle{Start: labelStart, Len: labelLen, Style: "b"})
					}
				} else if relOffset, byteLen := isStandaloneLabelHeading(trimmed); byteLen > 0 {
					labelStart := utf16Length(in.text[:lineStart+leadingBytes+relOffset])
					labelLen := utf16Length(trimmed[relOffset : relOffset+byteLen])
					if !hasStyleOverlap(in.styles, labelStart, labelLen) {
						in.styles = append(in.styles, protocol.TextStyle{Start: labelStart, Len: labelLen, Style: "b"})
					}
				}
			}
		}
		if lineEnd == len(in.text) {
			break
		}
		lineStart = lineEnd + 1
	}
	return in
}

func hasStyleOverlap(styles []protocol.TextStyle, start, length int) bool {
	end := start + length
	for _, s := range styles {
		if s.Style == "b" {
			sEnd := s.Start + s.Len
			if !(end <= s.Start || start >= sEnd) {
				return true
			}
		}
	}
	return false
}

func extractBulletLabelRange(line string) (int, int) {
	bulletLen := 0
	if strings.HasPrefix(line, "• ") {
		bulletLen = len("• ")
	} else if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		bulletLen = 2
	}
	if bulletLen == 0 {
		return 0, 0
	}
	rest := line[bulletLen:]
	colon := strings.IndexByte(rest, ':')
	if colon <= 0 || colon > 48 {
		return 0, 0
	}
	label := strings.TrimSpace(rest[:colon])
	if label == "" || strings.ContainsAny(label, "\n\r\t.,?!") {
		return 0, 0
	}
	labelStartInRest := strings.Index(rest, label)
	return bulletLen + labelStartInRest, len(label)
}

func isStandaloneLabelHeading(line string) (int, int) {
	if len(line) < 3 || len(line) > 30 || !strings.HasSuffix(line, ":") {
		return 0, 0
	}
	label := strings.TrimSpace(strings.TrimSuffix(line, ":"))
	if label == "" || strings.ContainsAny(label, "\n\r\t.,?!") {
		return 0, 0
	}
	labelStart := strings.Index(line, label)
	return labelStart, len(label)
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
