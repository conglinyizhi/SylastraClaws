package seahorse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// --- Message Operations ---

// AddMessage appends a message to a conversation.
func (s *Store) AddMessage(ctx context.Context, convID int64, role, content string, tokenCount int) (*Message, error) {
	return s.AddMessageWithReasoning(ctx, convID, role, content, "", tokenCount)
}

// AddMessageWithReasoning appends a message with reasoning content to a conversation.
func (s *Store) AddMessageWithReasoning(
	ctx context.Context,
	convID int64,
	role, content, reasoningContent string,
	tokenCount int,
) (*Message, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO messages (conversation_id, role, content, reasoning_content, token_count) VALUES (?, ?, ?, ?, ?)",
		convID, role, content, reasoningContent, tokenCount,
	)
	if err != nil {
		return nil, fmt.Errorf("add message: %w", err)
	}
	id, _ := result.LastInsertId()
	return &Message{
		ID:               id,
		ConversationID:   convID,
		Role:             role,
		Content:          content,
		ReasoningContent: reasoningContent,
		TokenCount:       tokenCount,
	}, nil
}

// partsToReadableContent derives a readable text summary from message parts.
// This ensures FTS5 indexing and summary formatting can access tool call information.
func partsToReadableContent(parts []MessagePart) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteString("\n")
		}
		switch p.Type {
		case "text":
			b.WriteString(p.Text)
		case "tool_use":
			fmt.Fprintf(&b, "[tool_use: %s, args: %s]", p.Name, p.Arguments)
		case "tool_result":
			fmt.Fprintf(&b, "[tool_result for %s: %s]", p.ToolCallID, p.Text)
		case "media":
			fmt.Fprintf(&b, "[media: %s (%s)]", p.MediaURI, p.MimeType)
		default:
			if p.Text != "" {
				b.WriteString(p.Text)
			}
		}
	}
	return b.String()
}

// AddMessageWithParts adds a message with structured parts.
func (s *Store) AddMessageWithParts(
	ctx context.Context,
	convID int64,
	role string,
	parts []MessagePart,
	tokenCount int,
) (*Message, error) {
	return s.AddMessageWithPartsAndReasoning(ctx, convID, role, parts, "", tokenCount)
}

// AddMessageWithPartsAndReasoning adds a message with structured parts and reasoning content.
func (s *Store) AddMessageWithPartsAndReasoning(
	ctx context.Context,
	convID int64,
	role string,
	parts []MessagePart,
	reasoningContent string,
	tokenCount int,
) (*Message, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Derive readable content from Parts for FTS5 indexing and summary formatting
	readableContent := partsToReadableContent(parts)

	result, err := tx.ExecContext(ctx,
		"INSERT INTO messages (conversation_id, role, content, reasoning_content, token_count) VALUES (?, ?, ?, ?, ?)",
		convID, role, readableContent, reasoningContent, tokenCount,
	)
	if err != nil {
		return nil, fmt.Errorf("add message: %w", err)
	}
	msgID, _ := result.LastInsertId()

	for i, p := range parts {
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO message_parts (message_id, type, text, name, arguments, tool_call_id, media_uri, mime_type, ordinal)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			msgID,
			p.Type,
			p.Text,
			p.Name,
			p.Arguments,
			p.ToolCallID,
			p.MediaURI,
			p.MimeType,
			i,
		)
		if err != nil {
			return nil, fmt.Errorf("add message part %d: %w", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// Return message with parts
	msg := &Message{
		ID:               msgID,
		ConversationID:   convID,
		Role:             role,
		ReasoningContent: reasoningContent,
		TokenCount:       tokenCount,
		Parts:            make([]MessagePart, len(parts)),
	}
	for i, p := range parts {
		p.MessageID = msgID
		msg.Parts[i] = p
	}
	return msg, nil
}

// GetMessages retrieves messages for a conversation.
func (s *Store) GetMessages(ctx context.Context, convID int64, limit int, beforeID int64) ([]Message, error) {
	query := "SELECT message_id, conversation_id, role, content, reasoning_content, token_count, created_at FROM messages WHERE conversation_id = ?"
	args := []any{convID}
	if beforeID > 0 {
		query += " AND message_id < ?"
		args = append(args, beforeID)
	}
	query += " ORDER BY message_id ASC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var msg Message
		var createdAt string
		if err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.Role,
			&msg.Content,
			&msg.ReasoningContent,
			&msg.TokenCount,
			&createdAt,
		); err != nil {
			return nil, err
		}
		msg.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		msgs = append(msgs, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load parts for all messages
	for i := range msgs {
		parts, err := s.loadMessageParts(ctx, msgs[i].ID)
		if err != nil {
			return nil, err
		}
		msgs[i].Parts = parts
	}

	return msgs, nil
}

// GetMessageCount returns total message count for a conversation.
func (s *Store) GetMessageCount(ctx context.Context, convID int64) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT count(*) FROM messages WHERE conversation_id = ?", convID,
	).Scan(&count)
	return count, err
}

// GetMessageByID retrieves a single message by ID.
func (s *Store) GetMessageByID(ctx context.Context, messageID int64) (*Message, error) {
	var msg Message
	var createdAt string
	err := s.db.QueryRowContext(
		ctx,
		"SELECT message_id, conversation_id, role, content, reasoning_content, token_count, created_at FROM messages WHERE message_id = ?",
		messageID,
	).Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.ReasoningContent, &msg.TokenCount, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message %d not found", messageID)
	}
	if err != nil {
		return nil, err
	}
	msg.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	msg.Parts, _ = s.loadMessageParts(ctx, msg.ID)
	return &msg, nil
}

// UpdateMessageReasoningContent updates reasoning_content for an existing message.
func (s *Store) UpdateMessageReasoningContent(ctx context.Context, messageID int64, reasoningContent string) error {
	result, err := s.db.ExecContext(
		ctx,
		"UPDATE messages SET reasoning_content = ? WHERE message_id = ?",
		reasoningContent,
		messageID,
	)
	if err != nil {
		return fmt.Errorf("update message reasoning_content: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update message reasoning_content rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("message %d not found", messageID)
	}
	return nil
}

func (s *Store) loadMessageParts(ctx context.Context, msgID int64) ([]MessagePart, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT part_id, message_id, type, text, name, arguments, tool_call_id, media_uri, mime_type
		 FROM message_parts WHERE message_id = ? ORDER BY ordinal`,
		msgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []MessagePart
	for rows.Next() {
		var p MessagePart
		if err := rows.Scan(&p.ID, &p.MessageID, &p.Type, &p.Text, &p.Name, &p.Arguments,
			&p.ToolCallID, &p.MediaURI, &p.MimeType); err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return parts, nil
}

