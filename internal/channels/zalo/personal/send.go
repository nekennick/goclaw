package personal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
	"github.com/nextlevelbuilder/goclaw/internal/channels/typing"
	"github.com/nextlevelbuilder/goclaw/internal/channels/zalo/personal/protocol"
)

// Keep a safety margin below the Zalo transport ceiling. It lets reports split
// at meaningful section boundaries instead of leaving a tiny trailing bubble.
const maxTextLength = 2000

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
	var members []protocol.GroupMemberInfo
	if threadType == protocol.ThreadTypeGroup {
		var err error
		members, err = c.groupMembersForSend(ctx, chatID)
		if err != nil {
			slog.Warn("zalo_personal: unable to verify group members; sending without native mentions", "group_id", chatID, "error", err)
		}
	}
	resolved := renderZaloMessage(text, members)
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

// stripMarkdownWithStyles keeps the visible text clean while preserving the
// emphasis requested by the model as native Zalo styles.
func stripMarkdownWithStyles(text string) (string, []protocol.TextStyle) {
	rendered := renderZaloMessage(text, nil)
	return rendered.text, rendered.styles
}
