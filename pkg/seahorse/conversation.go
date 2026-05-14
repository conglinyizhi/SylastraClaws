package seahorse

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// --- Conversation Operations ---

// GetOrCreateConversation returns the conversation for a sessionKey, creating if needed.
func (s *Store) GetOrCreateConversation(ctx context.Context, sessionKey string) (*Conversation, error) {
	// Try to get first
	conv, err := s.GetConversationBySessionKey(ctx, sessionKey)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		return conv, nil
	}

	// Create
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO conversations (session_key) VALUES (?)",
		sessionKey,
	)
	if err != nil {
		// Race: another goroutine may have inserted
		if isUniqueViolation(err) {
			return s.GetConversationBySessionKey(ctx, sessionKey)
		}
		return nil, fmt.Errorf("create conversation: %w", err)
	}
	id, _ := result.LastInsertId()
	return &Conversation{
		ConversationID: id,
		SessionKey:     sessionKey,
	}, nil
}

// GetConversationBySessionKey retrieves a conversation by session key.
func (s *Store) GetConversationBySessionKey(ctx context.Context, sessionKey string) (*Conversation, error) {
	var conv Conversation
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx,
		"SELECT conversation_id, session_key, created_at, updated_at FROM conversations WHERE session_key = ?",
		sessionKey,
	).Scan(&conv.ConversationID, &conv.SessionKey, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get conversation by session key: %w", err)
	}
	conv.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	conv.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	return &conv, nil
}

// GetSessionStatus returns status for a specific session.
func (s *Store) GetSessionStatus(ctx context.Context, sessionKey string) (*SessionStatus, error) {
	conv, err := s.GetConversationBySessionKey(ctx, sessionKey)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, nil
	}

	msgCount, _ := s.GetMessageCount(ctx, conv.ConversationID)
	sumCount, _ := s.getSummaryCount(ctx, conv.ConversationID)
	tokenCount, _ := s.GetContextTokenCount(ctx, conv.ConversationID)

	oldest, newest, _ := s.getMessageTimeRange(ctx, conv.ConversationID)

	return &SessionStatus{
		SessionKey:     conv.SessionKey,
		ConversationID: conv.ConversationID,
		Messages:       msgCount,
		TotalTokens:    tokenCount,
		Summaries:      sumCount,
		OldestAt:       oldest,
		NewestAt:       newest,
	}, nil
}

// GetAllSessionStatuses returns status for all sessions.
func (s *Store) GetAllSessionStatuses(ctx context.Context) ([]SessionStatus, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT session_key FROM conversations")
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var statuses []SessionStatus
	for rows.Next() {
		var sessionKey string
		if err := rows.Scan(&sessionKey); err != nil {
			continue
		}
		status, err := s.GetSessionStatus(ctx, sessionKey)
		if err != nil {
			continue
		}
		if status != nil {
			statuses = append(statuses, *status)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	return statuses, nil
}

func (s *Store) getSummaryCount(ctx context.Context, convID int64) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM summaries WHERE conversation_id = ?",
		convID,
	).Scan(&count)
	return count, err
}

func (s *Store) getMessageTimeRange(ctx context.Context, convID int64) (time.Time, time.Time, error) {
	var minTime, maxTime string
	err := s.db.QueryRowContext(ctx,
		"SELECT MIN(created_at), MAX(created_at) FROM messages WHERE conversation_id = ?",
		convID,
	).Scan(&minTime, &maxTime)
	if err != nil || minTime == "" {
		return time.Time{}, time.Time{}, err
	}
	oldest, _ := time.Parse("2006-01-02 15:04:05", minTime)
	newest, _ := time.Parse("2006-01-02 15:04:05", maxTime)
	return oldest, newest, nil
}

