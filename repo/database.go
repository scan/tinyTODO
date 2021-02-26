package repo

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"

	sqldblogger "github.com/simukti/sqldb-logger"
	"github.com/simukti/sqldb-logger/logadapter/zapadapter"
	"go.uber.org/zap"

	"github.com/scan/tinyTODO/model"
)

type Repository interface {
	Close() error

	InsertItem(context.Context, *model.Item) error
	RemoveItem(context.Context, string) error

	LoadItemsBefore(context.Context, int, int, time.Time) ([]*model.Item, error)
}

type repo struct {
	db *sql.DB
}

const createTableStatement = `
CREATE TABLE IF NOT EXISTS items (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NULL DEFAULT NULL,
    createdAt DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_created ON items(createdAt);
`

const insertItemStatement = `INSERT INTO items (id, title, content, createdAt) VALUES (?, ?, ?, ?)`

const removeItemStatement = `DELETE FROM items WHERE id = ?`

const queryItemsStatement = `SELECT id, title, content, createdAt FROM items WHERE createdAt <= ? ORDER BY createdAt DESC LIMIT ? OFFSET ?`

func Open(logger *zap.Logger) (Repository, error) {
	dsn := "./tinyTODO.db"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	loggerAdapter := zapadapter.New(logger)
	db = sqldblogger.OpenDriver(dsn, db.Driver(), loggerAdapter)

	statement, err := db.Prepare(createTableStatement)
	if err != nil {
		return nil, err
	}
	if _, err := statement.Exec(); err != nil {
		return nil, err
	}

	return &repo{db: db}, nil
}

func (r *repo) Close() error {
	return r.db.Close()
}

func (r *repo) InsertItem(ctx context.Context, item *model.Item) error {
	statement, err := r.db.PrepareContext(ctx, insertItemStatement)
	if err != nil {
		return err
	}

	if _, err := statement.ExecContext(ctx, item.ID, item.Title, item.Content, item.CreatedAt); err != nil {
		return err
	}

	return nil
}

func (r *repo) RemoveItem(ctx context.Context, id string) error {
	statement, err := r.db.PrepareContext(ctx, removeItemStatement)
	if err != nil {
		return err
	}

	if _, err := statement.ExecContext(ctx, id); err != nil {
		return err
	}

	return nil
}

func (r *repo) LoadItemsBefore(ctx context.Context, offset, limit int, before time.Time) ([]*model.Item, error) {
	rows, err := r.db.QueryContext(ctx, queryItemsStatement, before, limit, offset)
	if err != nil {
		return []*model.Item{}, err
	}
	defer rows.Close()

	items := make([]*model.Item, 0, limit)
	for rows.Next() {
		var title, id string
		var content *string
		var createdAt time.Time

		if err := rows.Scan(&id, &title, &content, &createdAt); err != nil {
			return []*model.Item{}, err
		}

		items = append(items, &model.Item{
			ID:        id,
			Title:     title,
			Content:   content,
			CreatedAt: createdAt,
		})
	}

	return items, nil
}
