package personal

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/nextlevelbuilder/goclaw/internal/channels/zalo/personal/protocol"
)

var (
	reCodeBlockFence = regexp.MustCompile("^```")
	reHeadingH1H2    = regexp.MustCompile(`^#{1,2}\s+(.+)$`)
	reHeadingH3Plus  = regexp.MustCompile(`^#{3,6}\s+(.+)$`)
	reBulletLine     = regexp.MustCompile(`^[ \t]*[-*+•][ \t]+(.*)$`)
	reNumberedLine   = regexp.MustCompile(`^[0-9]+[.)]\s+(.*)$`)
	reHorizontalRule = regexp.MustCompile(`^[ \t]*(?:---|\*\*\*|___)[ \t]*$`)
	reMentionHeading = regexp.MustCompile(`^((?:@\[[^\]]+\][^\n#]*?)\s*)(#{1,6}\s+.+)$`)
)

// renderZaloMessage parses Markdown formatting and group member mentions in a
// single synchronized pass, guaranteeing that all TMention and TextStyle offsets
// match the final plain text sent to Zalo with zero offset shift.
func renderZaloMessage(text string, members []protocol.GroupMemberInfo) mentionResolution {
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

	var builder strings.Builder
	var mentions []*protocol.TMention
	var styles []protocol.TextStyle

	rawLines := strings.Split(text, "\n")
	var lines []string
	inFence := false
	for _, rawLine := range rawLines {
		if reCodeBlockFence.MatchString(rawLine) {
			inFence = !inFence
			lines = append(lines, rawLine)
			continue
		}
		if !inFence {
			trimmed := strings.TrimSpace(rawLine)
			if m := reMentionHeading.FindStringSubmatch(trimmed); m != nil {
				p1 := strings.TrimSpace(m[1])
				p2 := strings.TrimSpace(m[2])
				if p1 != "" {
					lines = append(lines, p1)
					lines = append(lines, "")
				}
				if p2 != "" {
					lines = append(lines, p2)
				}
				continue
			}
		}
		lines = append(lines, rawLine)
	}

	inCodeBlock := false

	for i, rawLine := range lines {
		// 1. Code blocks: preserve verbatim content without markup or mention resolution
		if reCodeBlockFence.MatchString(rawLine) {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			builder.WriteString(rawLine)
			if i < len(lines)-1 {
				builder.WriteByte('\n')
			}
			continue
		}

		// 2. Horizontal rules: skip
		if reHorizontalRule.MatchString(rawLine) {
			continue
		}

		// 3. Blockquotes: strip prefix
		line := rawLine
		if strings.HasPrefix(line, "> ") {
			line = strings.TrimPrefix(line, "> ")
		} else if strings.HasPrefix(line, ">") {
			line = strings.TrimPrefix(line, ">")
		}

		trimmed := strings.TrimSpace(line)

		// 4. Headings: H1 / H2 -> Green Bold (c_15a85f + b)
		if match := reHeadingH1H2.FindStringSubmatch(line); match != nil {
			headingText := strings.TrimSpace(match[1])
			hStart := utf16Length(builder.String())
			parseInline(headingText, &builder, &mentions, &styles, byID, nameCount)
			hEnd := utf16Length(builder.String())
			if hEnd > hStart {
				styles = append(styles,
					protocol.TextStyle{Start: hStart, Len: hEnd - hStart, Style: "c_15a85f"},
					protocol.TextStyle{Start: hStart, Len: hEnd - hStart, Style: "b"},
				)
			}
			if i < len(lines)-1 {
				builder.WriteByte('\n')
			}
			continue
		}

		// 5. Headings: H3+ -> Orange Bold (c_f27806 + b)
		if match := reHeadingH3Plus.FindStringSubmatch(line); match != nil {
			headingText := strings.TrimSpace(match[1])
			hStart := utf16Length(builder.String())
			parseInline(headingText, &builder, &mentions, &styles, byID, nameCount)
			hEnd := utf16Length(builder.String())
			if hEnd > hStart {
				styles = append(styles,
					protocol.TextStyle{Start: hStart, Len: hEnd - hStart, Style: "c_f27806"},
					protocol.TextStyle{Start: hStart, Len: hEnd - hStart, Style: "b"},
				)
			}
			if i < len(lines)-1 {
				builder.WriteByte('\n')
			}
			continue
		}

		// 6. Bullet items
		if match := reBulletLine.FindStringSubmatch(line); match != nil {
			bulletContent := match[1]
			builder.WriteString("• ")

			// Check for bullet label before colon (e.g. "Label: details" or "**Label**: details")
			colonIdx := strings.IndexByte(bulletContent, ':')
			if colonIdx > 0 {
				rawLabel := bulletContent[:colonIdx]
				cleanLabel := strings.TrimSpace(rawLabel)
				// If label has reasonable length and does not contain sentence delimiters
				if cleanLabel != "" && utf8.RuneCountInString(cleanLabel) <= 60 && !strings.ContainsAny(cleanLabel, ".,?!") {
					lblStart := utf16Length(builder.String())
					parseInline(rawLabel, &builder, &mentions, &styles, byID, nameCount)
					lblEnd := utf16Length(builder.String())
					// Ensure the label is bold (merging will unify with any inner bold tags)
					styles = append(styles, protocol.TextStyle{Start: lblStart, Len: lblEnd - lblStart, Style: "b"})
					// Parse the colon and the rest of the bullet content
					parseInline(bulletContent[colonIdx:], &builder, &mentions, &styles, byID, nameCount)
					if i < len(lines)-1 {
						builder.WriteByte('\n')
					}
					continue
				}
			}

			parseInline(bulletContent, &builder, &mentions, &styles, byID, nameCount)
			if i < len(lines)-1 {
				builder.WriteByte('\n')
			}
			continue
		}

		// 7. Standalone short label line (e.g. "Nguồn:", "Lưu ý:")
		if strings.HasSuffix(trimmed, ":") && utf8.RuneCountInString(trimmed) <= 40 {
			rawLabel := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
			if rawLabel != "" && !strings.ContainsAny(rawLabel, ".,?!") {
				lblStart := utf16Length(builder.String())
				parseInline(rawLabel, &builder, &mentions, &styles, byID, nameCount)
				lblEnd := utf16Length(builder.String())
				styles = append(styles, protocol.TextStyle{Start: lblStart, Len: lblEnd - lblStart, Style: "b"})
				builder.WriteString(":")
				if i < len(lines)-1 {
					builder.WriteByte('\n')
				}
				continue
			}
		}

		// 8. Unformatted Numbered report heading (e.g. "1. Báo cáo kiểm kho")
		if reNumberedLine.MatchString(trimmed) && utf8.RuneCountInString(trimmed) <= 60 && !strings.Contains(trimmed, ":") {
			hStart := utf16Length(builder.String())
			parseInline(trimmed, &builder, &mentions, &styles, byID, nameCount)
			hEnd := utf16Length(builder.String())
			if hEnd > hStart {
				styles = append(styles,
					protocol.TextStyle{Start: hStart, Len: hEnd - hStart, Style: "c_15a85f"},
					protocol.TextStyle{Start: hStart, Len: hEnd - hStart, Style: "b"},
				)
			}
			if i < len(lines)-1 {
				builder.WriteByte('\n')
			}
			continue
		}

		// 9. Unformatted Emoji report heading (e.g. "📦 Chi tiết kiểm kho")
		if isEmojiReportHeading(trimmed) && utf8.RuneCountInString(trimmed) <= 60 {
			hStart := utf16Length(builder.String())
			parseInline(trimmed, &builder, &mentions, &styles, byID, nameCount)
			hEnd := utf16Length(builder.String())
			if hEnd > hStart {
				styles = append(styles,
					protocol.TextStyle{Start: hStart, Len: hEnd - hStart, Style: "c_f27806"},
					protocol.TextStyle{Start: hStart, Len: hEnd - hStart, Style: "b"},
				)
			}
			if i < len(lines)-1 {
				builder.WriteByte('\n')
			}
			continue
		}

		// 10. General text line
		parseInline(line, &builder, &mentions, &styles, byID, nameCount)
		if i < len(lines)-1 {
			builder.WriteByte('\n')
		}
	}

	resultText := strings.TrimRight(builder.String(), "\n")
	totalLen := utf16Length(resultText)

	// Normalize and merge overlapping/adjacent styles
	normalizedStyles := normalizeAndMergeStyles(styles, totalLen)

	return mentionResolution{
		text:     resultText,
		mentions: mentions,
		styles:   normalizedStyles,
	}
}

type inlineTokenType int

const (
	tokInlineCode inlineTokenType = iota
	tokMention
	tokOrangeHighlight // ***text*** or ___text___
	tokBold            // **text** or __text__
	tokLink            // [text](url)
	tokImage           // ![alt](url)
	tokStrikethrough   // ~~text~~
)

type tokenSpan struct {
	tokenType inlineTokenType
	start     int
	end       int
	inner1    string
	inner2    string
}

var (
	reInlineCode    = regexp.MustCompile("`([^`\n]+)`")
	reMentionMarker = regexp.MustCompile(`@\[([^|\]\n]+)\|([^\]\n]+)\]`)
	reOrangeStar    = regexp.MustCompile(`\*{3}([^*\n]+)\*{3}`)
	reOrangeUnder   = regexp.MustCompile(`_{3}([^_\n]+)_{3}`)
	reBoldStar      = regexp.MustCompile(`\*{2}([^*\n]+)\*{2}`)
	reBoldUnder     = regexp.MustCompile(`_{2}([^_\n]+)_{2}`)
	reMarkdownLink  = regexp.MustCompile(`\[([^\]\n]+)\]\(([^)\n]+)\)`)
	reMarkdownImg   = regexp.MustCompile(`!\[([^\]\n]*)\]\(([^)\n]+)\)`)
	reStrike        = regexp.MustCompile(`~~([^~\n]+)~~`)
)

func findNextInlineToken(s string, from int) (tokenSpan, bool) {
	sub := s[from:]
	if sub == "" {
		return tokenSpan{}, false
	}

	earliest := tokenSpan{start: len(s) + 1}
	found := false

	check := func(re *regexp.Regexp, t inlineTokenType) {
		loc := re.FindStringSubmatchIndex(sub)
		if loc != nil {
			absStart := from + loc[0]
			absEnd := from + loc[1]
			if absStart < earliest.start {
				found = true
				earliest.start = absStart
				earliest.end = absEnd
				earliest.tokenType = t
				if len(loc) >= 4 && loc[2] >= 0 {
					earliest.inner1 = sub[loc[2]:loc[3]]
				}
				if len(loc) >= 6 && loc[4] >= 0 {
					earliest.inner2 = sub[loc[4]:loc[5]]
				}
			}
		}
	}

	check(reMarkdownImg, tokImage)
	check(reInlineCode, tokInlineCode)
	check(reMentionMarker, tokMention)
	check(reOrangeStar, tokOrangeHighlight)
	check(reOrangeUnder, tokOrangeHighlight)
	check(reBoldStar, tokBold)
	check(reBoldUnder, tokBold)
	check(reMarkdownLink, tokLink)
	check(reStrike, tokStrikethrough)

	return earliest, found
}

func parseInline(
	s string,
	builder *strings.Builder,
	mentions *[]*protocol.TMention,
	styles *[]protocol.TextStyle,
	byID map[string]string,
	nameCount map[string]int,
) {
	pos := 0
	for pos < len(s) {
		tok, ok := findNextInlineToken(s, pos)
		if !ok {
			builder.WriteString(s[pos:])
			break
		}

		if tok.start > pos {
			builder.WriteString(s[pos:tok.start])
		}

		switch tok.tokenType {
		case tokInlineCode:
			// Output verbatim code without styling or mention parsing
			builder.WriteString(tok.inner1)

		case tokMention:
			uid := strings.TrimSpace(tok.inner1)
			fallbackName := strings.TrimSpace(tok.inner2)
			name, known := byID[uid]
			if known && nameCount[name] == 1 {
				mPos := utf16Length(builder.String())
				mentionText := "@" + name
				builder.WriteString(mentionText)
				*mentions = append(*mentions, &protocol.TMention{
					UID:  uid,
					Pos:  mPos,
					Len:  utf16Length(mentionText),
					Type: protocol.MentionEach,
				})
			} else {
				// Fallback plain text mention (no TMention notification)
				builder.WriteString("@" + fallbackName)
			}

		case tokOrangeHighlight:
			sStart := utf16Length(builder.String())
			parseInline(tok.inner1, builder, mentions, styles, byID, nameCount)
			sEnd := utf16Length(builder.String())
			if sEnd > sStart {
				*styles = append(*styles,
					protocol.TextStyle{Start: sStart, Len: sEnd - sStart, Style: "c_f27806"},
					protocol.TextStyle{Start: sStart, Len: sEnd - sStart, Style: "b"},
				)
			}

		case tokBold:
			sStart := utf16Length(builder.String())
			parseInline(tok.inner1, builder, mentions, styles, byID, nameCount)
			sEnd := utf16Length(builder.String())
			if sEnd > sStart {
				*styles = append(*styles, protocol.TextStyle{Start: sStart, Len: sEnd - sStart, Style: "b"})
			}

		case tokLink:
			builder.WriteString(tok.inner1)
			if tok.inner2 != "" {
				builder.WriteString(" (" + tok.inner2 + ")")
			}

		case tokImage:
			// Strip images

		case tokStrikethrough:
			parseInline(tok.inner1, builder, mentions, styles, byID, nameCount)
		}

		pos = tok.end
	}
}

// normalizeAndMergeStyles merges contiguous or overlapping bold spans and ensures
// no conflicting color spans (c_15a85f vs c_f27806) exist on the same character range.
func normalizeAndMergeStyles(styles []protocol.TextStyle, maxLen int) []protocol.TextStyle {
	if len(styles) == 0 {
		return nil
	}

	var boldSpans []protocol.TextStyle
	var colorSpans []protocol.TextStyle

	for _, s := range styles {
		if s.Len <= 0 || s.Start >= maxLen {
			continue
		}
		if s.Start+s.Len > maxLen {
			s.Len = maxLen - s.Start
		}
		if s.Style == "b" {
			boldSpans = append(boldSpans, s)
		} else if strings.HasPrefix(s.Style, "c_") {
			colorSpans = append(colorSpans, s)
		}
	}

	// 1. Merge overlapping or contiguous bold spans
	sort.Slice(boldSpans, func(i, j int) bool {
		if boldSpans[i].Start == boldSpans[j].Start {
			return boldSpans[i].Len > boldSpans[j].Len
		}
		return boldSpans[i].Start < boldSpans[j].Start
	})

	var mergedBold []protocol.TextStyle
	for _, s := range boldSpans {
		if len(mergedBold) == 0 {
			mergedBold = append(mergedBold, s)
			continue
		}
		last := &mergedBold[len(mergedBold)-1]
		if s.Start <= last.Start+last.Len {
			newEnd := max(last.Start+last.Len, s.Start+s.Len)
			last.Len = newEnd - last.Start
		} else {
			mergedBold = append(mergedBold, s)
		}
	}

	// 2. Prevent conflicting color spans (earlier color wins)
	var finalColors []protocol.TextStyle
	for _, c := range colorSpans {
		cEnd := c.Start + c.Len
		// Check overlap with existing colors
		overlaps := false
		for _, existing := range finalColors {
			eEnd := existing.Start + existing.Len
			if !(cEnd <= existing.Start || c.Start >= eEnd) {
				overlaps = true
				break
			}
		}
		if !overlaps {
			finalColors = append(finalColors, c)
		}
	}

	out := append(mergedBold, finalColors...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Start == out[j].Start {
			return out[i].Style < out[j].Style
		}
		return out[i].Start < out[j].Start
	})

	return out
}
