package personal

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
	"github.com/nextlevelbuilder/goclaw/internal/channels/typing"
	"github.com/nextlevelbuilder/goclaw/internal/channels/zalo"
	"github.com/nextlevelbuilder/goclaw/internal/channels/zalo/personal/protocol"
)

// Keep a safety margin below the Zalo transport ceiling. It lets reports split
// at meaningful section boundaries instead of leaving a tiny trailing bubble.
const maxTextLength = 1600

// Send delivers an outbound message to a Zalo chat.
func (c *Channel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	sess := c.session()
	if !c.IsRunning() || sess == nil {
		return fmt.Errorf("zalo_personal channel not running")
	}

	// Strip markdown — Zalo does not support any markup rendering.
	// Stop typing indicator before sending response
	if ctrl, ok := c.typingCtrls.LoadAndDelete(msg.ChatID); ok {
		ctrl.(*typing.Controller).Stop()
	}

	threadType := protocol.ThreadTypeUser
	if c.IsGroupApproved(msg.ChatID) {
		threadType = protocol.ThreadTypeGroup
	} else if msg.Metadata != nil {
		if _, ok := msg.Metadata["group_id"]; ok {
			threadType = protocol.ThreadTypeGroup
			c.MarkGroupApproved(msg.ChatID)
		}
	}

	// Send media attachments.
	for _, media := range msg.Media {
		if protocol.IsImageFile(media.URL) {
			if err := c.sendImage(ctx, sess, msg.ChatID, threadType, media.URL, media.Caption); err != nil {
				slog.Warn("zalo_personal: failed to send image", "path", media.URL, "error", err)
			}
		} else {
			if err := c.sendFile(ctx, sess, msg.ChatID, threadType, media.URL); err != nil {
				slog.Warn("zalo_personal: failed to send file", "path", media.URL, "error", err)
			}
		}
	}

	// Send text content (if any remains after media).
	if msg.Content != "" {
		return c.sendChunkedText(ctx, sess, msg.ChatID, threadType, msg.Content)
	}
	return nil
}

// sendImage uploads and sends an image file to a Zalo thread.
func (c *Channel) sendImage(ctx context.Context, sess *protocol.Session, chatID string, threadType protocol.ThreadType, filePath, caption string) error {
	upload, err := protocol.UploadImage(ctx, sess, chatID, threadType, filePath)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	_, err = protocol.SendImage(ctx, sess, chatID, threadType, upload, caption)
	return err
}

// sendFile uploads and sends a file to a Zalo thread.
func (c *Channel) sendFile(ctx context.Context, sess *protocol.Session, chatID string, threadType protocol.ThreadType, filePath string) error {
	ln := c.getListener()
	if ln == nil {
		return fmt.Errorf("listener not available for file upload")
	}
	upload, err := protocol.UploadFile(ctx, sess, ln, chatID, threadType, filePath)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	_, err = protocol.SendFile(ctx, sess, chatID, threadType, upload)
	return err
}

func (c *Channel) sendChunkedText(ctx context.Context, sess *protocol.Session, chatID string, threadType protocol.ThreadType, text string) error {
	plainText, markdownStyles := stripMarkdownWithStyles(text)
	resolved := mentionResolution{text: plainText, styles: markdownStyles}
	if threadType == protocol.ThreadTypeGroup {
		resolved = c.resolveGroupMentions(ctx, chatID, plainText)
		resolved.styles = append(resolved.styles, markdownStyles...)
	}
	resolved = addZaloReportStyles(resolved)
	for _, chunk := range splitMentionResolution(resolved, maxTextLength) {
		messageID, err := protocol.SendMessageWithFormatting(ctx, sess, chatID, threadType, chunk.text, chunk.mentions, chunk.styles)
		if err != nil {
			return err
		}
		slog.Info("zalo_personal: text sent",
			"chat_id", chatID,
			"thread_type", threadType,
			"bytes", len(chunk.text),
			"message_id", messageID,
		)
		if len(chunk.mentions) > 0 {
			slog.Info("zalo_personal: native mention text sent",
				"group_id", chatID,
				"mention_count", len(chunk.mentions),
				"message_id", messageID,
			)
		}
	}
	return nil
}

var (
	reH1H2          = regexp.MustCompile(`(?m)^#{1,2}\s+(.+)$`)
	reH3Plus        = regexp.MustCompile(`(?m)^#{3,6}\s+(.+)$`)
	reInlineStyles  = regexp.MustCompile(`(?s)(?:\*{3}(.+?)\*{3}|_{3}(.+?)_{3}|\*{2}(.+?)\*{2}|_{2}(.+?)_{2})`)
)

// stripMarkdownWithStyles keeps the visible text clean while preserving the
// emphasis requested by the model as native Zalo styles.
func stripMarkdownWithStyles(text string) (string, []protocol.TextStyle) {
	plain := zalo.StripMarkdown(text)
	if plain == "" {
		return plain, nil
	}

	styles := make([]protocol.TextStyle, 0)

	// 1. Headings
	// H1 / H2 -> Green Bold (c_15a85f + b)
	searchFromH1 := 0
	for _, match := range reH1H2.FindAllStringSubmatch(text, -1) {
		content := zalo.StripMarkdown(strings.TrimSpace(match[1]))
		if content == "" {
			continue
		}
		index := strings.Index(plain[searchFromH1:], content)
		if index < 0 {
			continue
		}
		index += searchFromH1
		styles = append(styles,
			protocol.TextStyle{Start: utf16Length(plain[:index]), Len: utf16Length(content), Style: "c_15a85f"},
			protocol.TextStyle{Start: utf16Length(plain[:index]), Len: utf16Length(content), Style: "b"},
		)
		searchFromH1 = index + len(content)
	}

	// H3+ -> Orange Bold (c_f27806 + b)
	searchFromH3 := 0
	for _, match := range reH3Plus.FindAllStringSubmatch(text, -1) {
		content := zalo.StripMarkdown(strings.TrimSpace(match[1]))
		if content == "" {
			continue
		}
		index := strings.Index(plain[searchFromH3:], content)
		if index < 0 {
			continue
		}
		index += searchFromH3
		styles = append(styles,
			protocol.TextStyle{Start: utf16Length(plain[:index]), Len: utf16Length(content), Style: "c_f27806"},
			protocol.TextStyle{Start: utf16Length(plain[:index]), Len: utf16Length(content), Style: "b"},
		)
		searchFromH3 = index + len(content)
	}

	// 2. Inline styles:
	// ***text*** -> Orange Bold (c_f27806 + b)
	// **text** -> Native Bold Black (b)
	searchFromInline := 0
	for _, match := range reInlineStyles.FindAllStringSubmatch(text, -1) {
		isOrange := match[1] != "" || match[2] != ""
		content := match[1]
		if content == "" {
			content = match[2]
		}
		if content == "" {
			content = match[3]
		}
		if content == "" {
			content = match[4]
		}
		if content == "" {
			continue
		}
		index := strings.Index(plain[searchFromInline:], content)
		if index < 0 {
			continue
		}
		index += searchFromInline
		if isOrange {
			styles = append(styles,
				protocol.TextStyle{Start: utf16Length(plain[:index]), Len: utf16Length(content), Style: "c_f27806"},
				protocol.TextStyle{Start: utf16Length(plain[:index]), Len: utf16Length(content), Style: "b"},
			)
		} else {
			styles = append(styles,
				protocol.TextStyle{Start: utf16Length(plain[:index]), Len: utf16Length(content), Style: "b"},
			)
		}
		searchFromInline = index + len(content)
	}
	return plain, styles
}
