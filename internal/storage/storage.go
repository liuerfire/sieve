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
	Reason        string
	InterestLevel string
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
	// Set connection limits: 1 for writing (SQLite requirement), multiple for reading if needed
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
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
        reason TEXT,
        interest_level TEXT,
        published_at DATETIME,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, err
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) SaveItem(ctx context.Context, item *Item) error {
	query := `
    INSERT OR REPLACE INTO items (
        id, source, title, link, description, content, summary, reason, interest_level, published_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		item.ID,
		item.Source,
		item.Title,
		item.Link,
		item.Description,
		item.Content,
		item.Summary,
		item.Reason,
		item.InterestLevel,
		item.PublishedAt,
	)
	return err
}

func (s *Storage) AllItems(ctx context.Context) iter.Seq2[*Item, error] {
	return func(yield func(*Item, error) bool) {
		query := `
    SELECT id, source, title, link, description, content, summary, reason, interest_level, published_at
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
				&item.Reason,
				&item.InterestLevel,
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
