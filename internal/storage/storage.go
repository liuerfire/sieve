// Package storage provides SQLite persistence for RSS items with WAL mode support.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"iter"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Item struct {
	ID            string
	Source        string
	Title         string
	Link          string
	Description   string
	Content       string
	Summary       string
	Thought       string
	Reason        string
	InterestLevel string
	IsRead        bool
	PublishedAt   time.Time
	Saved         bool
	SavedAt       *time.Time
	// UserInterestOverride records explicit user choice over AI classification.
	UserInterestOverride *string
	DuplicateOf          *string
}

type SearchFilters struct {
	Source string
	Level  string
	Saved  *bool
}

type ItemStats struct {
	TotalVisible  int `json:"total_visible"`
	Saved         int `json:"saved"`
	HighInterest  int `json:"high_interest"`
	UnreadVisible int `json:"unread_visible"`
	Interest      int `json:"interest"`
	Uninterested  int `json:"uninterested"`
}

type Storage struct {
	db *sql.DB
}

func InitDB(ctx context.Context, path string) (*Storage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Optimize SQLite for concurrent access and performance
	// With WAL mode: 1 writer + multiple readers allowed
	// Set to 4 connections: 1 for writes, 3 for concurrent reads
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(time.Hour)

	// Enable WAL mode
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}

	schema := `
    CREATE TABLE IF NOT EXISTS items (
        id TEXT PRIMARY KEY,
        source TEXT,
        title TEXT,
        link TEXT,
        description TEXT,
        content TEXT,
        summary TEXT,
        thought TEXT,
        reason TEXT,
        interest_level TEXT,
        is_read BOOLEAN DEFAULT 0,
        saved BOOLEAN DEFAULT 0,
        saved_at DATETIME,
        user_interest_override TEXT,
        duplicate_of TEXT,
        published_at DATETIME,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, err
	}

	indexSchema := `
    CREATE INDEX IF NOT EXISTS idx_interest_level ON items(interest_level);
    CREATE INDEX IF NOT EXISTS idx_published_at ON items(published_at DESC);
    CREATE INDEX IF NOT EXISTS idx_source ON items(source);
    CREATE INDEX IF NOT EXISTS idx_saved ON items(saved, saved_at DESC);
    CREATE INDEX IF NOT EXISTS idx_user_interest_override ON items(user_interest_override);
`
	if _, err := db.ExecContext(ctx, indexSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create indexes: %w", err)
	}

	ftsSchema := `
    CREATE VIRTUAL TABLE IF NOT EXISTS items_fts USING fts5(
        id UNINDEXED,
        title,
        description,
        content,
        summary,
        source
    );`
	if _, err := db.ExecContext(ctx, ftsSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create fts table: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) SaveItem(ctx context.Context, item *Item) error {
	query := `
    INSERT OR REPLACE INTO items (
        id, source, title, link, description, content, summary, thought, reason,
        interest_level, is_read, saved, saved_at, user_interest_override, duplicate_of, published_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	if _, err := s.db.ExecContext(ctx, query,
		item.ID,
		item.Source,
		item.Title,
		item.Link,
		item.Description,
		item.Content,
		item.Summary,
		item.Thought,
		item.Reason,
		item.InterestLevel,
		item.IsRead,
		item.Saved,
		item.SavedAt,
		item.UserInterestOverride,
		item.DuplicateOf,
		item.PublishedAt,
	); err != nil {
		return err
	}

	return s.upsertFTS(ctx, item)
}

func (s *Storage) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM items WHERE id = ?)`
	err := s.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}

func (s *Storage) AllItems(ctx context.Context) iter.Seq2[*Item, error] {
	return func(yield func(*Item, error) bool) {
		query := `
    SELECT id, source, title, link, description, content, summary, thought, reason,
           COALESCE(user_interest_override, interest_level) AS interest_level,
           is_read, saved, saved_at, user_interest_override, duplicate_of, published_at
    FROM items
    WHERE COALESCE(user_interest_override, interest_level) != 'exclude'
    ORDER BY published_at DESC`

		rows, err := s.db.QueryContext(ctx, query)
		if err != nil {
			yield(nil, err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var item Item
			err := rows.Scan(
				&item.ID,
				&item.Source,
				&item.Title,
				&item.Link,
				&item.Description,
				&item.Content,
				&item.Summary,
				&item.Thought,
				&item.Reason,
				&item.InterestLevel,
				&item.IsRead,
				&item.Saved,
				&item.SavedAt,
				&item.UserInterestOverride,
				&item.DuplicateOf,
				&item.PublishedAt,
			)
			if err != nil {
				if !yield(nil, err) {
					return
				}
				continue
			}
			if !yield(&item, nil) {
				return
			}
		}
	}
}

func (s *Storage) GetItems(ctx context.Context, limit, offset int) ([]*Item, error) {
	query := `
    SELECT id, source, title, link, description, content, summary, thought, reason,
           COALESCE(user_interest_override, interest_level) AS interest_level,
           is_read, saved, saved_at, user_interest_override, duplicate_of, published_at
    FROM items
    ORDER BY published_at DESC
    LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		var it Item
		err := rows.Scan(
			&it.ID,
			&it.Source,
			&it.Title,
			&it.Link,
			&it.Description,
			&it.Content,
			&it.Summary,
			&it.Thought,
			&it.Reason,
			&it.InterestLevel,
			&it.IsRead,
			&it.Saved,
			&it.SavedAt,
			&it.UserInterestOverride,
			&it.DuplicateOf,
			&it.PublishedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, &it)
	}
	return items, nil
}

func (s *Storage) UpdateLevel(ctx context.Context, id string, level string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE items SET interest_level = ? WHERE id = ?", level, id)
	return err
}

func (s *Storage) UpdateReadStatus(ctx context.Context, id string, read bool) error {
	_, err := s.db.ExecContext(ctx, "UPDATE items SET is_read = ? WHERE id = ?", read, id)
	return err
}

func (s *Storage) UpdateSavedStatus(ctx context.Context, id string, saved bool) error {
	if saved {
		_, err := s.db.ExecContext(ctx, "UPDATE items SET saved = 1, saved_at = CURRENT_TIMESTAMP WHERE id = ?", id)
		return err
	}
	_, err := s.db.ExecContext(ctx, "UPDATE items SET saved = 0, saved_at = NULL WHERE id = ?", id)
	return err
}

func (s *Storage) UpdateUserInterestOverride(ctx context.Context, id string, level *string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE items SET user_interest_override = ? WHERE id = ?", level, id)
	return err
}

func (s *Storage) DeleteItem(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM items WHERE id = ?", id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, "DELETE FROM items_fts WHERE id = ?", id)
	return err
}

func (s *Storage) SearchItems(ctx context.Context, q string, limit int, filters SearchFilters) ([]*Item, error) {
	if limit <= 0 {
		limit = 50
	}
	base := `
    SELECT i.id, i.source, i.title, i.link, i.description, i.content, i.summary, i.thought, i.reason,
           COALESCE(i.user_interest_override, i.interest_level) AS interest_level,
           i.is_read, i.saved, i.saved_at, i.user_interest_override, i.duplicate_of, i.published_at
    FROM items i
    JOIN items_fts f ON f.id = i.id
    WHERE COALESCE(i.user_interest_override, i.interest_level) != 'exclude'`
	args := make([]any, 0, 5)

	if strings.TrimSpace(q) != "" {
		base += " AND items_fts MATCH ?"
		args = append(args, q)
	}
	if filters.Source != "" {
		base += " AND i.source = ?"
		args = append(args, filters.Source)
	}
	if filters.Level != "" {
		base += " AND COALESCE(i.user_interest_override, i.interest_level) = ?"
		args = append(args, filters.Level)
	}
	if filters.Saved != nil {
		base += " AND i.saved = ?"
		args = append(args, *filters.Saved)
	}
	base += " ORDER BY i.published_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(
			&it.ID,
			&it.Source,
			&it.Title,
			&it.Link,
			&it.Description,
			&it.Content,
			&it.Summary,
			&it.Thought,
			&it.Reason,
			&it.InterestLevel,
			&it.IsRead,
			&it.Saved,
			&it.SavedAt,
			&it.UserInterestOverride,
			&it.DuplicateOf,
			&it.PublishedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &it)
	}

	return items, nil
}

func (s *Storage) DigestItems(ctx context.Context, since time.Time, limit int) ([]*Item, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
    SELECT id, source, title, link, description, content, summary, thought, reason,
           COALESCE(user_interest_override, interest_level) AS interest_level,
           is_read, saved, saved_at, user_interest_override, duplicate_of, published_at
    FROM items
    WHERE saved = 1 OR (COALESCE(user_interest_override, interest_level) = 'high_interest' AND published_at >= ?)
    ORDER BY COALESCE(saved_at, published_at) DESC
    LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(
			&it.ID,
			&it.Source,
			&it.Title,
			&it.Link,
			&it.Description,
			&it.Content,
			&it.Summary,
			&it.Thought,
			&it.Reason,
			&it.InterestLevel,
			&it.IsRead,
			&it.Saved,
			&it.SavedAt,
			&it.UserInterestOverride,
			&it.DuplicateOf,
			&it.PublishedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &it)
	}

	return items, nil
}

func (s *Storage) ListSources(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
    SELECT DISTINCT source
    FROM items
    WHERE TRIM(COALESCE(source, '')) != ''
    ORDER BY source ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []string
	for rows.Next() {
		var source string
		if err := rows.Scan(&source); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, nil
}

func (s *Storage) ItemStats(ctx context.Context) (*ItemStats, error) {
	const q = `
    SELECT
      SUM(CASE WHEN COALESCE(user_interest_override, interest_level) != 'exclude' THEN 1 ELSE 0 END) AS total_visible,
      SUM(CASE WHEN saved = 1 THEN 1 ELSE 0 END) AS saved,
      SUM(CASE WHEN COALESCE(user_interest_override, interest_level) = 'high_interest' THEN 1 ELSE 0 END) AS high_interest,
      SUM(CASE WHEN COALESCE(user_interest_override, interest_level) != 'exclude' AND is_read = 0 THEN 1 ELSE 0 END) AS unread_visible,
      SUM(CASE WHEN COALESCE(user_interest_override, interest_level) = 'interest' THEN 1 ELSE 0 END) AS interest,
      SUM(CASE WHEN COALESCE(user_interest_override, interest_level) = 'uninterested' THEN 1 ELSE 0 END) AS uninterested
    FROM items`

	var st ItemStats
	if err := s.db.QueryRowContext(ctx, q).Scan(
		&st.TotalVisible,
		&st.Saved,
		&st.HighInterest,
		&st.UnreadVisible,
		&st.Interest,
		&st.Uninterested,
	); err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *Storage) upsertFTS(ctx context.Context, item *Item) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM items_fts WHERE id = ?", item.ID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO items_fts (id, title, description, content, summary, source) VALUES (?, ?, ?, ?, ?, ?)",
		item.ID,
		item.Title,
		item.Description,
		item.Content,
		item.Summary,
		item.Source,
	)
	return err
}
