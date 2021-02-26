package repo

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/scan/tinyTODO/model"
)

type Repository interface {
	Close() error

	InsertItem(*model.Item) error
	RemoveItem(string) error

	LoadItemsBefore(int, int, time.Time) ([]*model.Item, error)
}

type repo struct {
	db *sql.DB
}

const createTableStatement = `
CREATE TABLE IF NOT EXISTS items (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NULL DEFAULT NULL,
    createdAt TEXT NOT NULL
)
`

func Open() (Repository, error) {
	db, err := sql.Open("sqlite3", "./tinyTODO.db")
	if err != nil {
		return nil, err
	}

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

func (r *repo) InsertItem(item *model.Item) error {
	return nil
}

func (t *repo) RemoveItem(id string) error {
	return nil
}

func (t *repo) LoadItemsBefore(offset, limit int, before time.Time) ([]*model.Item, error) {
	return nil, nil
}
