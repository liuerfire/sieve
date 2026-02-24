package storage

import (
	"context"
	"database/sql"
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

func (s *Storage) GetItems(ctx context.Context) ([]*Item, error) {
	query := `
    SELECT id, source, title, link, description, content, summary, reason, interest_level, published_at
    FROM items
    WHERE interest_level != 'exclude'
    ORDER BY published_at DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Item
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
			return nil, err
		}
		items = append(items, &item)
	}
	return items, nil
}
