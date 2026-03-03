// Package storage provides SQLite persistence for RSS items with WAL mode support.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"iter"
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
`
	if _, err := db.ExecContext(ctx, indexSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create indexes: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) SaveItem(ctx context.Context, item *Item) error {
	query := `
    INSERT OR REPLACE INTO items (
        id, source, title, link, description, content, summary, thought, reason, interest_level, is_read, published_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
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
		item.PublishedAt,
	)
	return err
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
    SELECT id, source, title, link, description, content, summary, thought, reason, interest_level, is_read, published_at
    FROM items
    WHERE interest_level != 'exclude'
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
    SELECT id, source, title, link, description, content, summary, thought, reason, interest_level, is_read, published_at
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

func (s *Storage) DeleteItem(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM items WHERE id = ?", id)
	return err
}
