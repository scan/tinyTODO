package model

import (
	"encoding/json"
	"time"
)

type Item struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   *string   `json:"content,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

func (Item) IsNode() {}

func NewItemFromRedisEntry(str []byte) (Item, error) {
	var item Item
	if err := json.Unmarshal(str, &item); err != nil {
		return Item{}, err
	}
	return item, nil
}

func (i Item) AsRedisEntry() ([]byte, error) {
	return json.Marshal(i)
}
