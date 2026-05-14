package seahorse

import (
	"database/sql"
	"time"
)

// Store provides SQLite storage for seahorse.
type Store struct {
	db *sql.DB
}

// CreateSummaryInput holds parameters for creating a summary.
type CreateSummaryInput struct {
	ConversationID       int64
	Kind                 SummaryKind
	Depth                int
	Content              string
	TokenCount           int
	EarliestAt           *time.Time
	LatestAt             *time.Time
	DescendantCount      int
	DescendantTokenCount int
	SourceMessageTokens  int
	Model                string
	ParentIDs            []string // For condensed: child summary IDs being condensed
}

